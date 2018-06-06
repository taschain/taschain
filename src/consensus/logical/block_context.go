package logical

import (
	"common"
	"core"
	"log"
	"math/big"
	"time"
	"fmt"
	"sync"
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

	currentVerifyContext *VerifyContext					//当前铸块的verifycontext

	lock sync.RWMutex

	Proc    *Processer   //处理器
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

func (bc *BlockContext) GetOrNewVerifyContext(bh *core.BlockHeader) (int32, *VerifyContext) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	if idx, vctx := bc.getVerifyContext(bh.Height, bh.PreHash); vctx == nil {
		vctx = newVerifyContext(bc, bh.Height, bh.PreTime, bh.PreHash)
		bc.verifyContexts = append(bc.verifyContexts, vctx)
		return int32(len(bc.verifyContexts)-1), vctx
	} else {
		return idx, vctx
	}
}

func (bc *BlockContext) RemoveVerifyContexts(vctx []*VerifyContext)  {
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
func (bc *BlockContext) receiveTrans(ths []common.Hash) []*SlotContext {
	slots := make([]*SlotContext, 0)

	for _, v := range bc.SafeGetVerifyContexts() {
		fullSlots := v.ReceiveTrans(ths)
		slots = append(slots, fullSlots...)
	}
	return slots
}

type QN_QUERY_SLOT_RESULT int //根据QN查找插槽结果枚举

const (
	QQSR_EMPTY_SLOT                     QN_QUERY_SLOT_RESULT = iota //找到一个空槽
	QQSR_REPLACE_SLOT                                               //找到一个能替换（QN值更低）的槽
	QQSR_EXIST_SLOT                            						//该QN对应的插槽已存在
	QQSR_NO_UNKNOWN                                                 //未知结果
)


func (bc *BlockContext) castingInfo() string {
	vctx := bc.currentVerifyContext
	if vctx != nil {
		return fmt.Sprintf("status=%v, castHeight=%v, prevHash=%v, prevTime=%v, signedMaxQN=%v", vctx.consensusStatus, vctx.castHeight, vctx.prevHash, vctx.prevTime.String(), vctx.signedMaxQN)
	} else {
		return "not in casting!"
	}
}

func (bc *BlockContext) Reset()  {
    bc.lock.Lock()
    defer bc.lock.Unlock()

    bc.reset()
}

//铸块上下文复位，在某个高度轮到当前组铸块时调用。
//to do : 还是索性重新生成。
func (bc *BlockContext) reset() {
	log.Printf("begin BlockContext::Reset...\n")

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
func (bc *BlockContext) StartCast(castHeight uint64, preTime time.Time, preHash common.Hash, immediatelyTriggerCheck bool) {

	log.Printf("proc(%v) begin startCast, trigger %v...\n", preTime.Format(time.Stamp), immediatelyTriggerCheck)
	bc.lock.Lock()
	defer bc.lock.Unlock()

	//bc.PreTime = prevTime //上一块的铸块成功时间
	//bc.ConsensusStatus = CBCS_CURRENT
	//bc.SignedMaxQN = INVALID_QN //等待第一个有效铸块
	//bc.PrevHash = prevHash
	//bc.CastHeight = castHeight
	//bc.Slots = *new([MAX_SYNC_CASTORS]*SlotContext)
	//bc.resetSlotContext()


	if _, verifyCtx := bc.getVerifyContext(castHeight, preHash); verifyCtx != nil {
		verifyCtx.Rebase(bc, castHeight, preTime, preHash)
		bc.currentVerifyContext = verifyCtx
	} else {
		verifyCtx = newVerifyContext(bc, castHeight, preTime, preHash)
		bc.verifyContexts = append(bc.verifyContexts, verifyCtx)
		bc.currentVerifyContext = verifyCtx
	}

	bc.Proc.Ticker.StartTickerRoutine(bc.getKingCheckRoutineName(), immediatelyTriggerCheck)
	log.Printf("startCast end. castInfo=%v\n", bc.castingInfo())
	return
}

//节点所在组成为当前铸块组
//该函数会被多次重入，需要做容错处理。
//在某个高度第一次进入时会启动定时器
//func (bc *BlockContext) BeingCastGroup(cgs *CastGroupSummary) (cast bool) {
//	//var chainHeight uint64
//	//if !PROC_TEST_MODE {
//	//	chainHeight = bc.Proc.MainChain.QueryTopBlock().Height
//	//}
//
//	castHeight := cgs.BlockHeight
//	preTime := cgs.PreTime
//	preHash := cgs.PreHash
//
//	if !cgs.GroupID.IsEqual(bc.MinerID.gid) {
//		log.Printf("cast group=%v, bc group=%v, diff failed.\n", GetIDPrefix(cgs.GroupID), GetIDPrefix(bc.MinerID.gid))
//		return false
//	}
//
//	//if castHeight > chainHeight+MAX_UNKNOWN_BLOCKS {
//	//	//不在合法的铸块高度内
//	//	log.Printf("height failed, chainHeight=%v, castHeight=%v.\n", chainHeight, castHeight)
//	//	//panic("BlockContext::BeingCastGroup height failed.")
//	//	return false
//	//}
//
//	bc.lock.Lock()
//	defer bc.lock.Unlock()
//
//	log.Printf("BeginCastGroup: bc.IsCasting=%v, bc.consensusStatus=%v, bc.castHeight=%v, castHeight=%v, bc.Pretime=%v, prevTime=%v, bc.PrevHash=%v, prevHash=%v\n", bc.isCasting(), bc.ConsensusStatus, bc.CastHeight, castHeight, bc.PreTime, preTime, bc.PrevHash, preHash)
//
//	if bc.maxQNCasted() && bc.CastHeight == castHeight { //如果已出最高qn块, 直接返回
//		log.Println("BeginCastGroup: max qn casted in this height: ", castHeight)
//		return false
//	}
//
//	if bc.isCasting() {	//如果在铸块中
//		if bc.CastHeight == castHeight {
//			if bc.PrevHash != preHash {
//				log.Printf("prevHash diff found, bc.prevHash=%v, prevHash=%v!!!!!!!!!!!\n", bc.PrevHash, preHash)
//			}
//			log.Printf("already in casting height %v\n", castHeight)
//		} else if bc.CastHeight > castHeight {
//			log.Printf("already in casting higher block, current castHeight=%v, request castHeight=%v\n", bc.CastHeight, castHeight)
//			return false
//		} else {
//			bc.castRebase(castHeight, preTime, preHash, false)
//		}
//	} else { //不在铸块, 则开启铸块
//		bc.castRebase(castHeight, preTime, preHash, true)
//	}
//
//	return true
//}


//计算当前铸块人位置和QN
func (bc *BlockContext) calcCastor(vctx *VerifyContext) (int32, int64) {
	var index int32 = -1
	//
	//d := time.Since(vctx.prevTime)
	//
	//max := vctx.getMaxCastTime()
	//
	//var secs = int64(d.Seconds())
	//if secs < max { //在组铸块共识时间窗口内
	log.Printf("calcCastor: castInfo=%v\n", bc.castingInfo())
	qn := vctx.calcQN()
	if qn < 0 {
		log.Printf("calcCastor qn negative found! qn=%v\n", qn)
		return index, qn
	}
	firstKing := bc.getFirstCastor(vctx.prevHash) //取得第一个铸块人位置
	log.Printf("mem_count=%v, first King pos=%v, qn=%v, cur King pos=%v.\n", bc.GroupMembers, firstKing, qn, int64(firstKing)+qn)
	if firstKing >= 0 && bc.GroupMembers > 0 {
		index = int32((qn + int64(firstKing)) % int64(bc.GroupMembers))
		log.Printf("real cur King pos(MOD mem_count)=%v.\n", index)
	} else {
		qn = -1
	}
	//} else {
	//	log.Printf("bc::calcCastor failed, out of group max cast time, PreTime=%v, escape seconds=%v.!!!\n", vctx.prevTime.Format(time.Stamp), secs)
	//}
	return index, qn
}
//取得第一个铸块人在组内的位置
func (bc *BlockContext) getFirstCastor(prevHash common.Hash) int32 {
	var index int32 = -1
	biHash := prevHash.Big()
	if biHash.BitLen() > 0 && bc.GroupMembers > 0 {
		index = int32(biHash.Mod(biHash, big.NewInt(int64(bc.GroupMembers))).Int64())
	}
	return index
}


//定时器例行处理
//如果返回false, 则关闭定时器
func (bc *BlockContext) kingTickerRoutine() bool {
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
	max := vctx.getMaxCastTime()

	if int64(d.Seconds()) >= max { //超过了组最大铸块时间
		log.Printf("proc(%v) end kingTickerRoutine, out of max group cast time, time=%v secs, castInfo=%v.\n", bc.Proc.getPrefix(), d.Seconds(), bc.castingInfo())
		//bc.reset()
		vctx.setTimeout()
		return false
	} else {
		//当前组仍在有效铸块共识时间内
		//检查自己是否成为铸块人
		index, qn := bc.calcCastor(vctx) //当前铸块人（KING）和QN值
		if index < 0 {
			log.Printf("kingTickerRoutine: calcCastor index =%v\n", index)
			return false
		}
		if vctx.signedMaxQN != INVALID_QN && qn <= vctx.signedMaxQN {	//已经铸出了更大的qn
			log.Printf("kingTickerRoutine: already cast maxer qn! height=%v, signMaxQN=%v, calcQn=%v\n", vctx.castHeight, vctx.signedMaxQN, qn)
			return false
		}
		bc.Proc.kingCheckAndCast(bc, vctx, index, qn)
		log.Printf("proc(%v) end kingTickerRoutine, KING_POS=%v, qn=%v.\n", bc.Proc.getPrefix(), index, qn)
		return true
	}
	return true
}
