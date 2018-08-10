package pow

import (
	"consensus/base"
	"sync/atomic"
	"time"
	"consensus/ticker"
	"middleware/types"
		"consensus/model"
	"common"
	"math/big"
	"strconv"
	)

/*
**  Creator: pxf
**  Date: 2018/8/8 上午11:31
**  Description:
 */


const (
	STATUS_RUNNING = 1	//计算中
	STATUS_STOP    = 2	//中断
	STATUS_SUCCESS	= 3	//已经成功找出解
)

const (
	CMD_POW_RESULT = 1
	CMD_POW_CONFIRM = 2
)

var (
	ACCEPT_HASH_DIFF = model.NewEnumResult(1, "hash不一致")
	ACCEPT_NONCE_ERR = model.NewEnumResult(2, "nonce错误")
	ACCEPT_EXPIRE = model.NewEnumResult(3, "过期")
	ACCEPT_NORMAL = model.NewEnumResult(4, "正常")
	ACCEPT_DUP_ERR = model.NewEnumResult(5, "重复")
)

type WorkerCommand struct {
	Cmd   int
	Param interface{}
}

type dataInput struct {
	data   []byte
	offset int
}

func newDataInput(hash common.Hash, gminer *model.GroupMinerID) *dataInput {
	data := hash.Bytes()
	data = append(data, gminer.Uid.Serialize()...)
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
	CmdCh       chan int
	stopCh 		chan struct{}
	Input       *dataInput
	BH          *types.BlockHeader
	GroupMiner  *model.GroupMinerID
	Nonce       uint64
	Difficulty  *Difficulty
	Status      int32
	PowDeadline *time.Time
	ConfirmDeadline *time.Time
	StartTime   time.Time
	context     *powContext
}

func NewPowWorker() *PowWorker {
	return &PowWorker{
		CmdCh:     make(chan int),
		stopCh:     make(chan struct{}),
		Nonce:     0,
		Status:    STATUS_STOP,
	}
}

func (w *PowWorker) tickerName() string {
    return "pow-worker"
}

func (w *PowWorker) setDeadline(start time.Time)  {
    w.PowDeadline = w.Difficulty.powDeadline(start)
    w.ConfirmDeadline = w.Difficulty.confirmDeadline(start)
    ticker.GetTickerInstance().RegisterRoutine(w.tickerName(), w.tickerRoutine, 1)
}

func (w *PowWorker) tickerRoutine() bool {
	if time.Now().After(*w.PowDeadline) {
		w.Stop()
		if !w.context.hasConfirmed() && time.Now().After(*w.ConfirmDeadline) {
			w.context.confirm()
			w.CmdCh <- CMD_POW_CONFIRM
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

func (w *PowWorker) Start()  {
	w.checkReady()
	if atomic.CompareAndSwapInt32(&w.Status, STATUS_STOP, STATUS_RUNNING) {
		w.StartTime = time.Now()
		go w.run()
	}
}

func (w *PowWorker) Success() bool {
    return atomic.LoadInt32(&w.Status) == STATUS_SUCCESS
}

func (w *PowWorker) toFinish(status int32) bool {
	if atomic.CompareAndSwapInt32(&w.Status, STATUS_RUNNING, status) {
		ticker.GetTickerInstance().RemoveRoutine(w.tickerName())
		return true
	}
	return false
}

func (w *PowWorker) computeDifficultyValue(nonce uint64, input *dataInput) *big.Int {
	input.setNonce(w.Nonce)
	h := base.Data2CommonHash(w.Input.data)
	h = base.Data2CommonHash(h.Bytes())
	return h.Big()
}

func (w *PowWorker) checkNonce(nonce uint64, input *dataInput) (d *big.Int, ok bool) {
    dv := w.computeDifficultyValue(nonce, input)
    return dv, w.Difficulty.Satisfy(dv)
}

func (w *PowWorker) run() bool {
	defer func() {
		w.stopCh <- struct{}{}
	}()
	for w.Status == STATUS_RUNNING {
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


func (w *PowWorker) Prepare(bh *types.BlockHeader, gminer *model.GroupMinerID, powThreshold int)  {
	if atomic.LoadInt32(&w.Status) == STATUS_RUNNING {
		w.Stop()
		<- w.stopCh
	}
	w.GroupMiner = gminer
	w.BH = bh
    w.Input = newDataInput(bh.Hash, gminer)
	w.Difficulty = DIFFCULTY_20_24
	w.setDeadline(bh.CurTime)
	w.Status = int32(STATUS_STOP)
	w.Nonce = 0
	w.context = newPowContext(powThreshold)
}

func (w *PowWorker) powExpire() bool {
    return time.Now().After(*w.PowDeadline)
}

func (w *PowWorker) AcceptResult(nonce uint64, hash common.Hash, gminer *model.GroupMinerID) *model.EnumResult {
	if hash != w.BH.Hash {
		return ACCEPT_HASH_DIFF
	}
	input := newDataInput(hash, gminer)
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
		nonce: nonce,
		level: w.Difficulty.Level(dv),
		value: dv,
	}
	if w.context.add(r) {
		return ACCEPT_NORMAL
	} else {
		return ACCEPT_DUP_ERR
	}
}

func (w *PowWorker) Confirmed() bool {
    return w.context.hasConfirmed()
}

func (w *PowWorker) GetNonceSeq() []model.MinerNonce {
    mns := make([]model.MinerNonce, 0)
	for _, r := range w.context.confirmed.results {
		mns = append(mns, model.MinerNonce{MinerID: r.minerID, Nonce: r.nonce})
	}
	return mns
}