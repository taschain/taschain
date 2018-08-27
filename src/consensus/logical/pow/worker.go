package pow

import (
	"common"
	"consensus/base"
	"consensus/groupsig"
	"consensus/model"
	"consensus/ticker"
	"math/big"
			"sync/atomic"
	"time"
	"vm/ethdb"
	"encoding/binary"
	"log"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2018/8/8 上午11:31
**  Description:
 */

const (
	STATUS_READY int32 = 0 //准备就绪
	STATUS_RUNNING = 1 //计算中
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

//用于pow计算的数据
type dataInput struct {
	data   []byte	//hash计算数据
	offset int		//nonce所在偏移量
}

func newDataInput(hash common.Hash, uid groupsig.ID) *dataInput {
	bs := uid.Serialize()
	data := hash.Bytes()
	data = append(data, bs...)
	offset := len(data)
	var nonceBit = make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBit, NONCE_MAX)
	data = append(data, nonceBit...)
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
	CmdCh            chan int
	stopCh           chan struct{}
	input            *dataInput
	BaseHash         common.Hash
	BaseHeight       uint64
	GroupMiner       *model.GroupMinerID
	Nonce            uint64
	difficulty       *Difficulty
	powStatus        int32
	powDeadline      *time.Time
	confirmStartTime *time.Time
	StartTime        time.Time
	context          *powContext
	broadcastStatus  int32
	storage          ethdb.Database
}

func NewPowWorker(db ethdb.Database, gminer *model.GroupMinerID) *PowWorker {
	w := &PowWorker{
		CmdCh:           make(chan int),
		stopCh:          make(chan struct{}),
		Nonce:           0,
		powStatus:       STATUS_STOP,
		broadcastStatus: BROADCAST_NONE,
		storage:         db,
		GroupMiner: 	gminer,

	}
	ticker.GetTickerInstance().RegisterRoutine(w.tickerName(), w.tickerRoutine, 1)
	return w
}

func (w *PowWorker) Finalize() {
    ticker.GetTickerInstance().RemoveRoutine(w.tickerName())
}

func (w *PowWorker) tickerName() string {
	return fmt.Sprintf("pow-worker-%v", w.GroupMiner.Gid.String())
}

func (w *PowWorker) setDeadline(start time.Time) {
	w.powDeadline = w.difficulty.sendPowResultTime(start)
	w.confirmStartTime = w.difficulty.sendPowConfirmTime(start)
}

func (w *PowWorker) tickerRoutine() bool {
	if time.Now().After(*w.powDeadline) {
		log.Printf("pow computation timeout")
		w.Stop()
		if time.Now().After(*w.confirmStartTime) {
			log.Printf("pow confirm start")
			w.tryConfirm()
		}
	}
	return true
}

func (w *PowWorker) Stop() {
	if w.toEnd(STATUS_STOP) {
		ticker.GetTickerInstance().StopTickerRoutine(w.tickerName())
	}
}

func (w *PowWorker) Start() {
	if atomic.CompareAndSwapInt32(&w.powStatus, STATUS_READY, STATUS_RUNNING) {
		ticker.GetTickerInstance().StartAndTriggerRoutine(w.tickerName())
		w.StartTime = time.Now()
		go w.run()
	} else if atomic.CompareAndSwapInt32(&w.powStatus, STATUS_STOP, STATUS_RUNNING) {
		ticker.GetTickerInstance().StartAndTriggerRoutine(w.tickerName())
		go w.run()
	}
}

func (w *PowWorker) Success() bool {
	return atomic.LoadInt32(&w.powStatus) == STATUS_SUCCESS
}

func (w *PowWorker) toEnd(status int32) bool {
	if atomic.CompareAndSwapInt32(&w.powStatus, STATUS_RUNNING, status) {
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

func checkDataInput(difficulty *Difficulty, nonce uint64, input *dataInput) (*big.Int, bool) {
	if nonce > NONCE_MAX {
		return nil, false
	}
	dv := computeDifficultyValue(nonce, input)
	return dv, difficulty.Satisfy(dv)
}

func CheckMinerNonce(diffculty *Difficulty, baseHash common.Hash, nonce uint64, uid groupsig.ID) (d *big.Int, ok bool) {
	input := newDataInput(baseHash, uid)
	return checkDataInput(diffculty, nonce, input)
}

func (w *PowWorker) checkNonce(nonce uint64, input *dataInput) (d *big.Int, ok bool) {
	return checkDataInput(w.difficulty, nonce, input)
}

func (w *PowWorker) checkMinerNonce(nonce uint64, uid groupsig.ID) (d *big.Int, ok bool) {
    return CheckMinerNonce(w.difficulty, w.BaseHash, nonce, uid)
}

func (w *PowWorker) run() bool {
	defer func() {
		w.stopCh <- struct{}{}
	}()
	for w.powStatus == STATUS_RUNNING && w.Nonce <= NONCE_MAX {
		w.Nonce++
		if _, ok := w.checkNonce(w.Nonce, w.input); ok {
			if w.toEnd(STATUS_SUCCESS) {
				w.CmdCh <- CMD_POW_RESULT
				return true
			}
		}
	}
	return false
}

func (w *PowWorker) IsRunning() bool {
    return atomic.LoadInt32(&w.powStatus) == STATUS_RUNNING
}

func (w *PowWorker) Prepare(baseHash common.Hash, baseHeight uint64, startTime time.Time, members int) bool {
	st := atomic.LoadInt32(&w.powStatus)
	switch st {
	case STATUS_RUNNING:
		if w.BaseHash == baseHash { //已经在运行中
			return false
		}
		w.Stop()
		<-w.stopCh	//等待run函数退出
	case STATUS_STOP:
		if w.BaseHash == baseHash {
			return true
		}
	case STATUS_SUCCESS:
		if w.BaseHash == baseHash { //已经计算出结果
			return false
		}
	}

	w.BaseHash = baseHash
	w.BaseHeight = baseHeight
	w.input = newDataInput(baseHash, w.GroupMiner.Uid)
	w.difficulty = DIFFCULTY_20_24
	w.setDeadline(startTime)
	w.powStatus = STATUS_READY
	w.broadcastStatus = BROADCAST_NONE
	w.Nonce = 0
	w.context = newPowContext(baseHash, members)
	return true
}

func (w *PowWorker) powExpire() bool {
	return time.Now().After(*w.confirmStartTime)
}

func (w *PowWorker) ComputationCost() string {
    return time.Since(w.StartTime).String()
}

func (w *PowWorker) AcceptResult(nonce uint64, baseHash common.Hash, uid groupsig.ID) *model.EnumResult {
	if baseHash != w.BaseHash {
		return ACCEPT_HASH_DIFF
	}

	var (
		dv *big.Int
		ok bool
	)
	if dv, ok = w.checkMinerNonce(nonce, uid); !ok {
		return ACCEPT_NONCE_ERR
	}
	if w.powExpire() {
		return ACCEPT_EXPIRE
	}

	r := &powResult{
		minerID: uid,
		nonce:   nonce,
		level:   w.difficulty.Level(dv),
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
		ticker.GetTickerInstance().StopTickerRoutine(w.tickerName())
	} else {
		log.Printf("pow confirm return false")
	}
}

func (w *PowWorker) AcceptConfirm(baseHash common.Hash, seq []model.MinerNonce, sender groupsig.ID, signature groupsig.Signature) *model.EnumResult {
	if baseHash != w.BaseHash {
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
	return atomic.CompareAndSwapInt32(&w.broadcastStatus, sourceStatus, targetStatus)
}

func (w *PowWorker) AcceptFinal(baseHash common.Hash, seq []model.MinerNonce, sender groupsig.ID, signature groupsig.Signature) *model.EnumResult {
	if baseHash != w.BaseHash {
		return FINAL_HASH_DIFF
	}
	if !w.powExpire() {
		return FINAL_POW_NOT_FINISEHD
	}

	for _, mn := range seq {
		if r := w.context.getResult(&mn); r == nil {
			_, ok := w.checkMinerNonce(mn.Nonce, sender)
			if !ok {
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

	w.tryConfirm()
	confirmed := w.context.getConfirmed()
	if confirmed != nil && confirmed.gSign != nil {
		if confirmed.resultHash != confirming.resultHash {
			panic("confirmed at different BaseHash")
		}
	} else {
		w.context.setConfirmed(confirming)
	}
	return FINAL_SUCCESS
}

