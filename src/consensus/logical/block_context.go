package logical

import (
	"common"
	"log"
	"time"
	"fmt"
	"sync"
	"middleware/types"
)

///////////////////////////////////////////////////////////////////////////////
//组铸块共识上下文结构（一个高度有一个上下文，一个组的不同铸块高度不重用）
type BlockContext struct {
	Version         uint
	//PreTime         time.Time                   //所属组的当前铸块起始时间戳(组内必须一致，不然时间片会异常，所以直接取上个铸块完成时间)
	//CCTimer         time.Ticker                 //共识定时器
	//TickerRunning	bool
	//SignedMaxQN     int64                       //组内已铸出的最大QN值的块
	//PrevHash        common.Hash                 //上一块哈希值
	//CastHeight      uint64                      //待铸块高度
	GroupMembers    uint                        //组成员数量
	//Threshold       uint                           //百分比（0-100）
	//Slots [MAX_SYNC_CASTORS]*SlotContext //铸块槽列表
	verifyContexts	[]*VerifyContext

	currentVerifyContext *VerifyContext //当前铸块的verifycontext

	lock sync.RWMutex

	Proc    *Processor   //处理器
	MinerID GroupMinerID //矿工ID和所属组ID
	pos     int          //矿工在组内的排位
}

func (bc *BlockContext) Init(mid GroupMinerID) {
	bc.MinerID = mid
	bc.verifyContexts = make([]*VerifyContext, 0)
	bc.reset()
}

func (bc *BlockContext) getKingCheckRoutineName() string {
	return "king_check_routine_" + GetIDPrefix(bc.MinerID.gid)
}

func (bc *BlockContext) alreadyInCasting(height uint64, preHash common.Hash) bool {
	vctx := bc.GetCurrentVerifyContext()
	if vctx != nil {
		vctx.lock.Lock()
		defer vctx.lock.Unlock()
		return vctx.isCasting() && !vctx.maxQNCasted() && vctx.castHeight == height && vctx.prevHash == preHash
	} else {
		return false
	}
}

func (bc *BlockContext) getVerifyContext(height uint64, preHash common.Hash) (int32, *VerifyContext) {

	for idx, ctx := range bc.verifyContexts {
		if ctx.castHeight == height && ctx.prevHash == preHash {
			return int32(idx), ctx
		}
	}
	return -1, nil
}

func (bc *BlockContext) GetCurrentVerifyContext() *VerifyContext {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	return bc.currentVerifyContext
}

func (bc *BlockContext) GetOrNewVerifyContext(bh *types.BlockHeader, preBH *types.BlockHeader) (int32, *VerifyContext) {
	expireTime := GetCastExpireTime(bh.PreTime, bh.Height - preBH.Height)

	bc.lock.Lock()
	defer bc.lock.Unlock()

	if idx, vctx := bc.getVerifyContext(bh.Height, bh.PreHash); vctx == nil {
		vctx = newVerifyContext(bc, bh.Height, expireTime, preBH)
		bc.verifyContexts = append(bc.verifyContexts, vctx)
		return int32(len(bc.verifyContexts) - 1), vctx
	} else {
		return idx, vctx
	}
}

func (bc *BlockContext) RemoveVerifyContexts(vctx []*VerifyContext) {
	if vctx == nil || len(vctx) == 0 {
		return
	}
	bc.lock.Lock()
	defer bc.lock.Unlock()

	for _, ctx := range vctx {
		idx, _ := bc.getVerifyContext(ctx.castHeight, ctx.prevHash)
		if idx < 0 {
			continue
		}
		if ctx == bc.currentVerifyContext {
			bc.reset()
		}
		bc.verifyContexts = append(bc.verifyContexts[:idx], bc.verifyContexts[idx+1:]...)
	}
}

func (bc *BlockContext) SafeGetVerifyContexts() []*VerifyContext {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	ret := make([]*VerifyContext, len(bc.verifyContexts))
	copy(ret, bc.verifyContexts)
	return ret
}

//（网络接收）新到交易集通知
//返回不再缺失交易的QN槽列表
//func (bc *BlockContext) receiveTrans(ths []common.Hash) []*SlotContext {
//	slots := make([]*SlotContext, 0)
//
//	for _, v := range bc.SafeGetVerifyContexts() {
//		fullSlots := v.ReceiveTrans(ths)
//		slots = append(slots, fullSlots...)
//	}
//	return slots
//}

type QN_QUERY_SLOT_RESULT int //根据QN查找插槽结果枚举

const (
	QQSR_EMPTY_SLOT   QN_QUERY_SLOT_RESULT = iota //找到一个空槽
	QQSR_REPLACE_SLOT                             //找到一个能替换（QN值更低）的槽
	QQSR_EXIST_SLOT                               //该QN对应的插槽已存在
	QQSR_NO_UNKNOWN                               //未知结果
)

func (bc *BlockContext) castingInfo() string {
	vctx := bc.currentVerifyContext
	if vctx != nil {
		return fmt.Sprintf("status=%v, castHeight=%v, prevHash=%v, prevTime=%v, signedMaxQN=%v", vctx.consensusStatus, vctx.castHeight, GetHashPrefix(vctx.prevHash), vctx.prevTime.String(), vctx.signedMaxQN)
	} else {
		return "not in casting!"
	}
}

func (bc *BlockContext) Reset() {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	bc.reset()
}

//铸块上下文复位，在某个高度轮到当前组铸块时调用。
//to do : 还是索性重新生成。
func (bc *BlockContext) reset() {

	bc.Version = CONSENSUS_VERSION
	//bc.PreTime = *new(time.Time)
	//bc.CCTimer.Stop() //关闭定时器
	//bc.TickerRunning = false
	//bc.consensusStatus = CBCS_IDLE
	//bc.SignedMaxQN = INVALID_QN
	//bc.PrevHash = common.Hash{}
	//bc.CastHeight = 0
	bc.currentVerifyContext = nil
	bc.GroupMembers = uint(GROUP_MAX_MEMBERS)

	bc.Proc.Ticker.StopTickerRoutine(bc.getKingCheckRoutineName())
	log.Printf("end BlockContext::Reset.\n")
}

//开始铸块
func (bc *BlockContext) StartCast(castHeight uint64, expire time.Time, baseBH *types.BlockHeader) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	//bc.PreTime = prevTime //上一块的铸块成功时间
	//bc.ConsensusStatus = CBCS_CURRENT
	//bc.SignedMaxQN = INVALID_QN //等待第一个有效铸块
	//bc.PrevHash = prevHash
	//bc.CastHeight = castHeight
	//bc.Slots = *new([MAX_SYNC_CASTORS]*SlotContext)
	//bc.resetSlotContext()

	if _, verifyCtx := bc.getVerifyContext(castHeight, baseBH.Hash); verifyCtx != nil {
		//verifyCtx.Rebase(bc, castHeight, preTime, preHash)
		bc.currentVerifyContext = verifyCtx
	} else {
		verifyCtx = newVerifyContext(bc, castHeight, expire, baseBH)
		bc.verifyContexts = append(bc.verifyContexts, verifyCtx)
		bc.currentVerifyContext = verifyCtx
	}

	bc.Proc.Ticker.StartAndTriggerRoutine(bc.getKingCheckRoutineName())
	//bc.Proc.Ticker.StartTickerRoutine(bc.getKingCheckRoutineName(), true)
	log.Printf("startCast end. castInfo=%v\n", bc.castingInfo())
	return
}

//定时器例行处理
//如果返回false, 则关闭定时器
func (bc *BlockContext) kingTickerRoutine() bool {
	if !bc.Proc.Ready() {
		return false
	}
	log.Printf("proc(%v) begin kingTickerRoutine, time=%v...\n", bc.Proc.getPrefix(), time.Now().Format(time.Stamp))

	vctx := bc.GetCurrentVerifyContext()
	if vctx == nil {
		log.Printf("kingTickerRoutine: verifyContext is nil, return!\n")
		return false
	}

	vctx.lock.Lock()
	defer vctx.lock.Unlock()

	if !vctx.isCasting() || vctx.maxQNCasted() { //没有在组铸块共识中或已经出最高qn块
		log.Printf("proc(%v) not in casting, reset and direct return. castingInfo=%v.\n", bc.Proc.getPrefix(), bc.castingInfo())
		bc.Reset() //提前出块完成
		return false
	}

	d := time.Since(vctx.prevTime) //上个铸块完成到现在的时间
	//max := vctx.getMaxCastTime()

	if vctx.castExpire() { //超过了组最大铸块时间
		log.Printf("proc(%v) end kingTickerRoutine, out of max group cast time, time=%v secs, castInfo=%v.\n", bc.Proc.getPrefix(), d.Seconds(), bc.castingInfo())
		//bc.reset()
		vctx.setTimeout()
		return false
	} else {
		//当前组仍在有效铸块共识时间内
		//检查自己是否成为铸块人
		index, qn := vctx.calcCastor() //当前铸块人（KING）和QN值
		if index < 0 {
			log.Printf("kingTickerRoutine: calcCastor index =%v\n", index)
			return false
		}
		if vctx.signedMaxQN != INVALID_QN && qn <= vctx.signedMaxQN { //已经铸出了更大的qn
			log.Printf("kingTickerRoutine: already cast maxer qn! height=%v, signMaxQN=%v, calcQn=%v\n", vctx.castHeight, vctx.signedMaxQN, qn)
			return false
		}
		bc.Proc.kingCheckAndCast(bc, vctx, index, qn)
		log.Printf("proc(%v) end kingTickerRoutine, KING_POS=%v, qn=%v.\n", bc.Proc.getPrefix(), index, qn)
		return true
	}
	return true
}

func (bc *BlockContext) getGroupSecret() *GroupSecret {
    return bc.Proc.getGroupSecret(bc.MinerID.gid)
}
