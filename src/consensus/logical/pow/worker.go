package pow

import (
	"common"
	"consensus/base"
	"consensus/groupsig"
	"consensus/model"
	"consensus/ticker"
	"math/big"
	"middleware/types"
	"strconv"
	"sync/atomic"
	"time"
	"vm/ethdb"
		)

/*
**  Creator: pxf
**  Date: 2018/8/8 上午11:31
**  Description:
 */

const (
	STATUS_RUNNING int32 = 1 //计算中
	STATUS_STOP    = 2 //中断
	STATUS_SUCCESS = 3 //已经成功找出解
)

const (
	CMD_POW_RESULT  = 1
	CMD_POW_CONFIRM = 2
)

const (
	BROADCAST_NONE    int32 = 0
	BROADCAST_CONFIRM       = 1
	BROADCAST_FINAL         = 2
	BROADCAST_PERSIST       = 3
)

const NONCE_MAX = uint64(0xffffffffffff)

var (
	ACCEPT_HASH_DIFF = model.NewEnumResult(1, "hash不一致")
	ACCEPT_NONCE_ERR = model.NewEnumResult(2, "nonce错误")
	ACCEPT_EXPIRE    = model.NewEnumResult(3, "过期")
	ACCEPT_NORMAL    = model.NewEnumResult(4, "正常")
	ACCEPT_DUP_ERR   = model.NewEnumResult(5, "重复")
)

var (
	CONFIRM_HASH_DIFF            = model.NewEnumResult(1, "hash不一致")
	CONFIRM_MINERNONCE_NOT_FOUND = model.NewEnumResult(2, "未找到相应的矿工nonce")
	CONFIRM_POW_NOT_FINISHED     = model.NewEnumResult(3, "pow计算未完成")
	CONFIRM_RESULT_HASH_DIFF     = model.NewEnumResult(4, "结果hash不一致")
	CONFIRM_SIGN_NORMAL          = model.NewEnumResult(5, "分片签名")
	CONFIRM_SIGN_RECOVERED       = model.NewEnumResult(6, "恢复出组签名")
)

var (
	FINAL_HASH_DIFF = model.NewEnumResult(1, "hash不一致")
	FINAL_POW_NOT_FINISEHD = model.NewEnumResult(2, "pow未完成")
	FINAL_ACCEPT_NONCE_FAIL    = model.NewEnumResult(3, "矿工nonce校验失败")
	FINAL_CONFIRM_FAIL    = model.NewEnumResult(4, "确认失败")
	FINAL_SUCCESS		= model.NewEnumResult(5, "成功")
)

type WorkerCommand struct {
	Cmd   int
	Param interface{}
}

type dataInput struct {
	data   []byte
	offset int
}

func newDataInput(hash common.Hash, uid groupsig.ID) *dataInput {
	data := hash.Bytes()
	data = append(data, uid.Serialize()...)
	offset := len(data)
	data = strconv.AppendUint(data, 0, 10)
	return &dataInput{
		data:   data,
		offset: offset,
	}
}

func (di *dataInput) setNonce(nonce uint64) {
	di.data[di.offset] = byte(nonce >> 56)
	di.data[di.offset+1] = byte(nonce >> 48)
	di.data[di.offset+2] = byte(nonce >> 40)
	di.data[di.offset+3] = byte(nonce >> 32)
	di.data[di.offset+4] = byte(nonce >> 24)
	di.data[di.offset+5] = byte(nonce >> 16)
	di.data[di.offset+6] = byte(nonce >> 8)
	di.data[di.offset+7] = byte(nonce)
}

type PowWorker struct {
	CmdCh           chan int
	stopCh          chan struct{}
	Input           *dataInput
	BH              *types.BlockHeader
	GroupMiner      *model.GroupMinerID
	Nonce           uint64
	Difficulty      *Difficulty
	PowStatus       int32
	PowDeadline     *time.Time
	ConfirmDeadline *time.Time
	StartTime       time.Time
	context         *powContext
	BroadcastStatus int32
	storage 		ethdb.Database
}

func NewPowWorker(db ethdb.Database) *PowWorker {
	w := &PowWorker{
		CmdCh:           make(chan int),
		stopCh:          make(chan struct{}),
		Nonce:           0,
		PowStatus:       STATUS_STOP,
		BroadcastStatus: BROADCAST_NONE,
		storage: db,
	}
	return w
}

func (w *PowWorker) tickerName() string {
	return "pow-worker"
}

func (w *PowWorker) setDeadline(start time.Time) {
	w.PowDeadline = w.Difficulty.powDeadline(start)
	w.ConfirmDeadline = w.Difficulty.confirmDeadline(start)
	ticker.GetTickerInstance().RegisterRoutine(w.tickerName(), w.tickerRoutine, 1)
}

func (w *PowWorker) tickerRoutine() bool {
	if time.Now().After(*w.PowDeadline) {
		w.Stop()
		if time.Now().After(*w.ConfirmDeadline) {
			w.tryConfirm()
		}
	}
	return true
}

func (w *PowWorker) checkReady() {
	if w.Input == nil || w.Difficulty == nil || w.PowDeadline == nil || w.BH == nil {
		panic("param not ready")
	}
}

func (w *PowWorker) Stop() {
	w.toFinish(STATUS_STOP)
}

func (w *PowWorker) Start() {
	w.checkReady()
	if atomic.CompareAndSwapInt32(&w.PowStatus, STATUS_STOP, STATUS_RUNNING) {
		w.StartTime = time.Now()
		go w.run()
	}
}

func (w *PowWorker) Success() bool {
	return atomic.LoadInt32(&w.PowStatus) == STATUS_SUCCESS
}

func (w *PowWorker) toFinish(status int32) bool {
	if atomic.CompareAndSwapInt32(&w.PowStatus, STATUS_RUNNING, status) {
		ticker.GetTickerInstance().RemoveRoutine(w.tickerName())
		return true
	}
	return false
}

func computeDifficultyValue(nonce uint64, input *dataInput) *big.Int {
	input.setNonce(nonce)
	h := base.Data2CommonHash(input.data)
	h = base.Data2CommonHash(h.Bytes())
	return h.Big()
}

func CheckMinerNonce(diffculty *Difficulty, blockHash common.Hash, minerNonce *model.MinerNonce) (d *big.Int, ok bool) {
	if minerNonce.Nonce > NONCE_MAX {
		return nil, false
	}
	input := newDataInput(blockHash, minerNonce.MinerID)
	dv := computeDifficultyValue(minerNonce.Nonce, input)
	return dv, diffculty.Satisfy(dv)
}

func (w *PowWorker) checkNonce(nonce uint64, input *dataInput) (d *big.Int, ok bool) {
	if nonce > NONCE_MAX {
		return nil, false
	}
	dv := computeDifficultyValue(nonce, input)
	return dv, w.Difficulty.Satisfy(dv)
}

func (w *PowWorker) run() bool {
	defer func() {
		w.stopCh <- struct{}{}
	}()
	for w.PowStatus == STATUS_RUNNING && w.Nonce <= NONCE_MAX {
		w.Nonce++
		if _, ok := w.checkNonce(w.Nonce, w.Input); ok {
			if w.toFinish(STATUS_SUCCESS) {
				w.CmdCh <- CMD_POW_RESULT
				return true
			}
		}
	}
	return false
}

func (w *PowWorker) IsRunning() bool {
    return atomic.LoadInt32(&w.PowStatus) == STATUS_RUNNING
}

func (w *PowWorker) IsStopped() bool {
	return atomic.LoadInt32(&w.PowStatus) == STATUS_STOP
}

func (w *PowWorker) Prepare(bh *types.BlockHeader, gminer *model.GroupMinerID, members int) bool {
	if w.IsRunning() {
		if w.BH.Hash == bh.Hash {
			return false
		}
		w.Stop()
		<-w.stopCh
	}
	w.GroupMiner = gminer
	w.BH = bh
	w.Input = newDataInput(bh.Hash, gminer.Uid)
	w.Difficulty = DIFFCULTY_20_24
	w.setDeadline(bh.CurTime)
	w.PowStatus = STATUS_STOP
	w.BroadcastStatus = BROADCAST_NONE
	w.Nonce = 0
	w.context = newPowContext(bh.Hash, members)
	return true
}

func (w *PowWorker) powExpire() bool {
	return time.Now().After(*w.PowDeadline)
}

func (w *PowWorker) AcceptResult(nonce uint64, blockHash common.Hash, gminer *model.GroupMinerID) *model.EnumResult {
	if blockHash != w.BH.Hash {
		return ACCEPT_HASH_DIFF
	}

	input := newDataInput(blockHash, gminer.Uid)
	var (
		dv *big.Int
		ok bool
	)
	if dv, ok = w.checkNonce(nonce, input); !ok {
		return ACCEPT_NONCE_ERR
	}
	if w.powExpire() {
		return ACCEPT_EXPIRE
	}

	r := &powResult{
		minerID: gminer.Uid,
		nonce:   nonce,
		level:   w.Difficulty.Level(dv),
		value:   dv,
	}
	if w.context.addResult(r) {
		return ACCEPT_NORMAL
	} else {
		return ACCEPT_DUP_ERR
	}
}

func (w *PowWorker) Confirmed() bool {
	return w.context.hasConfirmed()
}

func (w *PowWorker) GetConfirmed() *powConfirm {
	return w.context.getConfirmed()
}

func (w *PowWorker) GetConfirmedNonceSeq() []model.MinerNonce {
	return w.GetConfirmed().genNonceSeq()
}

func (w *PowWorker) tryConfirm() {
	if w.context.confirm() {
		w.CmdCh <- CMD_POW_CONFIRM
	}
}

func (w *PowWorker) AcceptConfirm(blockHash common.Hash, seq []model.MinerNonce, sender groupsig.ID, signature groupsig.Signature) *model.EnumResult {
	if blockHash != w.BH.Hash {
		return CONFIRM_HASH_DIFF
	}
	if !w.powExpire() {
		return CONFIRM_POW_NOT_FINISHED
	}
	results := make(powResults, 0)
	for _, mn := range seq {
		if r := w.context.getResult(&mn); r == nil {
			return CONFIRM_MINERNONCE_NOT_FOUND
		} else {
			results = append(results, r)
		}
	}
	confirm := w.context.newPowConfirm(results)
	w.tryConfirm()

	if confirm.resultHash != w.context.getConfirmed().resultHash {
		return CONFIRM_RESULT_HASH_DIFF
	}

	if w.context.addSign(sender, signature) {
		return CONFIRM_SIGN_RECOVERED
	} else {
		return CONFIRM_SIGN_NORMAL
	}
}

func (w *PowWorker) GetGSign() *groupsig.Signature {
	sig := w.context.getConfirmed().gSignGenerator.GetGroupSign()
	return &sig
}

func (w *PowWorker) CheckBroadcastStatus(sourceStatus int32, targetStatus int32) bool {
	return atomic.CompareAndSwapInt32(&w.BroadcastStatus, sourceStatus, targetStatus)
}

func (w *PowWorker) AcceptFinal(blockHash common.Hash, seq []model.MinerNonce, sender groupsig.ID, signature groupsig.Signature) *model.EnumResult {
	if blockHash != w.BH.Hash {
		return FINAL_HASH_DIFF
	}
	if !w.powExpire() {
		return FINAL_POW_NOT_FINISEHD
	}

	for _, mn := range seq {
		if r := w.context.getResult(&mn); r == nil {
			ret := w.AcceptResult(mn.Nonce, blockHash, &model.GroupMinerID{Uid: sender, Gid:w.GroupMiner.Gid})
			if ret != ACCEPT_NORMAL {
				return FINAL_ACCEPT_NONCE_FAIL
			}
		}
	}

	results := make(powResults, 0)
	for _, mn := range seq {
		results = append(results, w.context.getResult(&mn))
	}

	confirming := w.context.newPowConfirm(results)
	confirming.gSign = &signature

	confirmed := w.context.getConfirmed()
	if confirmed != nil && confirmed.gSign != nil {
		if confirmed.resultHash != confirming.resultHash {
			panic("confirmed at different blockHash")
		}
	} else {
		w.context.setConfirmed(confirming)
	}
	return FINAL_SUCCESS
}

