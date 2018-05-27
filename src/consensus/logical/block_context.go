package logical

import (
	"common"
	"consensus/groupsig"
	"core"
	"log"
	"math/big"
	"time"
	"vm/common/math"
	"fmt"
	"sync"
)


//组铸块共识状态（针对某个高度而言）
type CAST_BLOCK_CONSENSUS_STATUS int

const (
	CBCS_IDLE           CAST_BLOCK_CONSENSUS_STATUS = iota //非当前组
	CBCS_CURRENT                                           //成为当前铸块组
	CBCS_CASTING                                           //至少收到一块组内共识数据
	CBCS_BLOCKED                                           //组内已有铸块完成（已通知到组外）
	CBCS_MAX_QN_BLOCKED                                    //组内最大铸块完成（已通知到组外），该高度铸块结束
	CBCS_TIMEOUT                                           //组铸块超时
)


///////////////////////////////////////////////////////////////////////////////
//组铸块共识上下文结构（一个高度有一个上下文，一个组的不同铸块高度不重用）
type BlockContext struct {
	Version         uint
	PreTime         time.Time                   //所属组的当前铸块起始时间戳(组内必须一致，不然时间片会异常，所以直接取上个铸块完成时间)
	//CCTimer         time.Ticker                 //共识定时器
	//TickerRunning	bool
	SignedMaxQN     int64                       //组内已铸出的最大QN值的块
	ConsensusStatus CAST_BLOCK_CONSENSUS_STATUS //铸块状态
	PrevHash        common.Hash                 //上一块哈希值
	CastHeight      uint64                      //待铸块高度
	GroupMembers    uint                        //组成员数量
	//Threshold       uint                           //百分比（0-100）
	Slots [MAX_SYNC_CASTORS]*SlotContext //铸块槽列表

	lock sync.RWMutex

	Proc    *Processer   //处理器
	MinerID GroupMinerID //矿工ID和所属组ID
	pos     int          //矿工在组内的排位
}

//切换到铸块高度
//func (bc *BlockContext) Switch2Height(cgs CastGroupSummary) bool {
//	bc.lock.Lock()
//	defer bc.lock.Unlock()
//
//	log.Printf("begin bc::Switch2Height, cur height=%v, new height=%v...\n", bc.CastHeight, cgs.BlockHeight)
//	var switched bool
//	if !cgs.GroupID.IsEqual(bc.MinerID.gid) {
//		log.Printf("cast group=%v, bc group=%v, diff failed.\n", GetIDPrefix(cgs.GroupID), GetIDPrefix(bc.MinerID.gid))
//		return false
//	}
//	if cgs.BlockHeight == bc.CastHeight { //已经在当前高度
//		log.Printf("already in this height, return true direct.\n")
//		return true
//	}
//	if cgs.BlockHeight < bc.CastHeight {
//		log.Printf("cast height-%v, bc height=%v, less failed..\n", cgs.BlockHeight, bc.CastHeight)
//		return false
//	}
//	bc.reset()
//	bc.CastHeight = cgs.BlockHeight
//	bc.PreTime = cgs.PreTime
//	bc.PrevHash = cgs.PreHash
//	bc.ConsensusStatus = CBCS_CURRENT
//	switched = true
//	log.Printf("end bc::Switch2Height, switched=%v.\n", switched)
//	return switched
//}

func (bc *BlockContext) Init(mid GroupMinerID) {
	bc.MinerID = mid
	bc.reset()
}

func (bc *BlockContext) getKingCheckRoutineName() string {
    return "king_check_routine_" + GetIDPrefix(bc.MinerID.gid)
}

func (bc *BlockContext) alreadyInCasting(height uint64, preHash common.Hash) bool {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
    return bc.isCasting() && bc.CastHeight == height && bc.PrevHash == preHash
}


//检查是否要处理某个铸块槽
//返回true需要处理，返回false可以丢弃。
func (bc *BlockContext) needHandleQN(qn uint) bool {
	if bc.SignedMaxQN == INVALID_QN { //当前该组还没有铸出过块
		return true
	} else { //当前该组已经有成功的铸块（来自某个铸块槽）
		return qn > uint(bc.SignedMaxQN)
	}
}

//完成（上链，向组外广播）某个铸块槽后更新当前高度的最小QN值
func (bc *BlockContext) signedUpdateMinQN(qn uint) bool {
	b := bc.needHandleQN(qn)
	if b {
		bc.SignedMaxQN = int64(qn)
	}
	return b
}

//完成某个铸块槽的铸块（上链，组外广播）后，更新组的当前高度铸块状态
func (bc *BlockContext) castedUpdateStatus(qn uint) bool {
	log.Printf("castedUpdateStatus before status=%v, qn=%v\n", bc.ConsensusStatus, qn)
	bc.lock.Lock()
	defer bc.lock.Unlock()

	bc.signedUpdateMinQN(qn)

	st := bc.ConsensusStatus

	switch st {
	case CBCS_IDLE, CBCS_TIMEOUT, CBCS_MAX_QN_BLOCKED:	//不在铸块周期或已经铸出最大块
		return false
	case CBCS_CASTING, CBCS_CURRENT, CBCS_BLOCKED:
		if qn == uint(MAX_QN) {
			bc.ConsensusStatus = CBCS_MAX_QN_BLOCKED
		} else {
			bc.ConsensusStatus = CBCS_BLOCKED
		}
		return true
	default:
		return true
	}

}

func (bc *BlockContext) maxQNCasted() bool {
    return bc.ConsensusStatus == CBCS_MAX_QN_BLOCKED
}

func (bc *BlockContext) PrintSlotInfo() string {
	var str string
	for i, v := range bc.Slots {
		if v.QueueNumber != INVALID_QN {
			str += fmt.Sprintf("slot %v: qn=%v, status=%v, msgs=%v, tf=%v. ", i, v.QueueNumber, v.SlotStatus, len(v.MapWitness), v.TransFulled)
		}
	}
	if len(str) == 0 {
		str = "all slot empty."
	}
	return str
}

//检查是否有空槽可以接纳一个铸块槽
//如果还有空槽，返回空槽序号。如果没有空槽，返回-1.
func (bc *BlockContext) findEmptySlot() int32 {
	for i, v := range bc.Slots {
		if v.QueueNumber == INVALID_QN {
			return int32(i)
		}
	}
	return -1
}

//检查目前在处理中的QN值最高的铸块槽。
//返回QN值最高的铸块槽的序号和QN值。如果当前全部是空槽，序号和QN值都返回-1.
func (bc *BlockContext) findMinQNSlot() (int32, int64) {
	var index int32 = -1
	var minQN int64 = math.MaxInt64
	for i, v := range bc.Slots {
		if v.QueueNumber < minQN {
			minQN = v.QueueNumber
			index = int32(i)
		}
	}
	return index, minQN
}

//检查是否有指定QN值的铸块槽
//返回：int32:铸块槽序号（没找到返回-1），bool：该铸块槽是否收到出块人消息（在铸块槽序号>=0时有意义）
func (bc *BlockContext) findCastSlot(qn int64) (int32) {
	for i, v := range bc.Slots {
		if v != nil && v.QueueNumber == qn {
			return int32(i)
		}
	}
	return -1
}

//（网络接收）新到交易集通知
//返回不再缺失交易的QN槽列表
func (bc *BlockContext) ReceTrans(ths []common.Hash) []*SlotContext {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	slots := make([]*SlotContext, 0)
	for _, v := range bc.Slots {
		if v != nil && v.QueueNumber != INVALID_QN {
			before, after := v.ReceTrans(ths)
			if !before && after { //该插槽已不再有缺失的交易
				slots = append(slots, v)
			}
		}
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

func (bc *BlockContext) getSlotByQN(qn int64) *SlotContext {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	i := bc.findCastSlot(qn)
	if i >= 0 {
		return bc.Slots[i]
	} else {
		return nil
	}
}

func (bc *BlockContext) getQNs() []int64 {
	qns := make([]int64, 0)
	for _, sc := range bc.Slots {
		qns = append(qns, sc.QueueNumber)
	}
	return qns
}

//根据QN优先级规则，尝试找到有效的插槽
func (bc *BlockContext) consensusFindSlot(qn int64) (int32, QN_QUERY_SLOT_RESULT) {
	var minQN int64 = -1
	i := bc.findCastSlot(qn)
	if i >= 0 { //该qn的槽已存在
		log.Printf("prov(%v) exist slot qn=%v, msg_count=%v.\n", bc.Proc.getPrefix(), qn, bc.Slots[i].MessageSize())
		return i, QQSR_EXIST_SLOT
	} else {
		i = bc.findEmptySlot()
		if i >= 0 { //找到空槽
			log.Printf("prov(%v) found empty slot_index=%v.\n", bc.Proc.getPrefix(), i)
			return i, QQSR_EMPTY_SLOT
		} else {
			i, minQN = bc.findMinQNSlot() //取得最小槽
			log.Printf("prov(%v) slot fulled, exist minQN=%v, slot_index=%v, new_qn=%v.\n", bc.Proc.getPrefix(), minQN, i, qn)
			if qn > minQN { //最小槽的QN比新的QN小, 替换之
				return i, QQSR_REPLACE_SLOT
			}
		}
	}
	return -1, QQSR_NO_UNKNOWN
}

//计算QN
func (bc *BlockContext) calcQN() int64 {
	max := bc.getMaxCastTime()
	diff := time.Since(bc.PreTime).Seconds() //从上个铸块完成到现在的时间（秒）
	log.Printf("calcQN, time_begin=%v, diff=%v, max=%v.\n", bc.PreTime.Format(time.Stamp), diff, max)

	var qn int64
	if bc.baseOnGeneisBlock() {
		qn = max / int64(MAX_USER_CAST_TIME) - int64(diff) / int64(MAX_USER_CAST_TIME)
	} else {
		d := int64(diff) + int64(MAX_GROUP_BLOCK_TIME) - max
		qn = int64(MAX_QN) - d / int64(MAX_USER_CAST_TIME)
	}

	return qn
}


//铸块共识消息处理函数
//cv：铸块共识数据，出块消息或验块消息生成的ConsensusBlockSummary.
//=0, 接受; =1,接受，达到阈值；<0, 不接受。
func (bc *BlockContext) acceptCV(bh core.BlockHeader, si SignData) CAST_BLOCK_MESSAGE_RESULT {
	log.Printf("begin BlockContext::acceptCV, height=%v, qn=%v...\n", bh.Height, bh.QueueNumber)
	calcQN := bc.calcQN()
	if calcQN < 0 || bh.QueueNumber < 0 { //时间窗口异常
		log.Printf("proc(%v) acceptCV failed(time windwos ERROR), calcQN=%v, qn=%v.\n", bc.Proc.getPrefix(), calcQN, bh.QueueNumber)
		return CBMR_ERROR_ARG
	}
	if uint64(calcQN) > bh.QueueNumber { //未轮到该QN出块
		log.Printf("proc(%v) acceptCV failed(qn ERROR), calcQN=%v, qn=%v.\n", bc.Proc.getPrefix(), calcQN, bh.QueueNumber)
		return CMBR_IGNORE_QN_FUTURE
	}

	if !bc.needHandleQN(uint(bh.QueueNumber)) { //该组已经铸出过QN值更大的块
		return CMBR_IGNORE_MAX_QN_SIGNED
	}

	i, info := bc.consensusFindSlot(int64(bh.QueueNumber))
	log.Printf("proc(%v) consensusFindSlot, qn=%v, i=%v, info=%v.\n", bc.Proc.getPrefix(), bh.QueueNumber, i, info)
	if i < 0 { //没有找到有效的插槽
		return CMBR_IGNORE_QN_BIG_QN
	}
	//找到有效的插槽
	if info == QQSR_EMPTY_SLOT || info == QQSR_REPLACE_SLOT {
		log.Printf("proc(%v) put new_qn=%v in slot[%v], REPLACE=%v.\n", bc.Proc.getPrefix(), bh.QueueNumber, i, info == QQSR_REPLACE_SLOT)
		bc.Slots[i] = newSlotContext(bh, si)
		return CBMR_PIECE
	} else { //该QN值对应的插槽已存在
		result := bc.Slots[i].AcceptPiece(bh, si)
		log.Printf("proc(%v) bc::slot[%v] AcceptPiece result=%v, msg_count=%v.\n", bc.Proc.getPrefix(), i, result, bc.Slots[i].MessageSize())
		return result
	}
	return CMBR_ERROR_UNKNOWN
}


func (bc BlockContext) isCasting() bool {
	if bc.ConsensusStatus == CBCS_IDLE || bc.ConsensusStatus == CBCS_TIMEOUT {
		//空闲，已出权重最高的块，超时
		return false
	} else {
		return true
	}
}

//判断当前节点所在组当前是否在铸块共识中
func (bc BlockContext) IsCasting() bool {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	return bc.isCasting()
}

func (bc *BlockContext) resetSlotContext()  {
	for i := 0; i < MAX_SYNC_CASTORS; i++ {
		sc := new(SlotContext)
		sc.reset()
		bc.Slots[i] = sc
	}
}

//铸块上下文复位，在某个高度轮到当前组铸块时调用。
//to do : 还是索性重新生成。
func (bc *BlockContext) reset() {
	log.Printf("begin BlockContext::Reset...\n")
	bc.Version = CONSENSUS_VERSION
	bc.PreTime = *new(time.Time)
	//bc.CCTimer.Stop() //关闭定时器
	//bc.TickerRunning = false
	bc.ConsensusStatus = CBCS_IDLE
	bc.SignedMaxQN = INVALID_QN
	bc.PrevHash = common.Hash{}
	bc.CastHeight = 0
	bc.GroupMembers = uint(GROUP_MAX_MEMBERS)
	//bc.Threshold = SSSS_THRESHOLD
	//bc.Slots = *new([MAX_SYNC_CASTORS]*SlotContext)
	bc.resetSlotContext()
	bc.Proc.Ticker.StopTickerRoutine(bc.getKingCheckRoutineName())
	log.Printf("end BlockContext::Reset.\n")
}

//调整铸块基准
func (bc *BlockContext) castRebase(castHeight uint64, preTime time.Time, preHash common.Hash, immediatelyTriggerCheck bool) {
	log.Printf("proc(%v) begin castRebase, trigger %v...\n", preTime.Format(time.Stamp), immediatelyTriggerCheck)

	bc.PreTime = preTime //上一块的铸块成功时间
	bc.ConsensusStatus = CBCS_CURRENT
	bc.SignedMaxQN = INVALID_QN //等待第一个有效铸块
	bc.PrevHash = preHash
	bc.CastHeight = castHeight
	//bc.Slots = *new([MAX_SYNC_CASTORS]*SlotContext)
	bc.resetSlotContext()
	bc.Proc.Ticker.StartTickerRoutine(bc.getKingCheckRoutineName(), immediatelyTriggerCheck)
	log.Printf("castRebase end. bc.castHeight=%v, bc.PreHash=%v, bc.PreTime=%v\n", castHeight, preHash, preTime)
	return
}

//节点所在组成为当前铸块组
//该函数会被多次重入，需要做容错处理。
//在某个高度第一次进入时会启动定时器
func (bc *BlockContext) BeingCastGroup(cgs *CastGroupSummary) (cast bool) {
	//var chainHeight uint64
	//if !PROC_TEST_MODE {
	//	chainHeight = bc.Proc.MainChain.QueryTopBlock().Height
	//}

	castHeight := cgs.BlockHeight
	preTime := cgs.PreTime
	preHash := cgs.PreHash

	if !cgs.GroupID.IsEqual(bc.MinerID.gid) {
		log.Printf("cast group=%v, bc group=%v, diff failed.\n", GetIDPrefix(cgs.GroupID), GetIDPrefix(bc.MinerID.gid))
		return false
	}

	//if castHeight > chainHeight+MAX_UNKNOWN_BLOCKS {
	//	//不在合法的铸块高度内
	//	log.Printf("height failed, chainHeight=%v, castHeight=%v.\n", chainHeight, castHeight)
	//	//panic("BlockContext::BeingCastGroup height failed.")
	//	return false
	//}

	bc.lock.Lock()
	defer bc.lock.Unlock()

	log.Printf("BeginCastGroup: bc.IsCasting=%v, bc.ConsensusStatus=%v, bc.castHeight=%v, castHeight=%v, bc.Pretime=%v, preTime=%v, bc.PrevHash=%v, preHash=%v\n", bc.isCasting(), bc.ConsensusStatus, bc.CastHeight, castHeight, bc.PreTime, preTime, bc.PrevHash, preHash)

	if bc.maxQNCasted() && bc.CastHeight == castHeight { //如果已出最高qn块, 直接返回
		log.Println("BeginCastGroup: max qn casted in this height: ", castHeight)
		return false
	}

	if bc.isCasting() {	//如果在铸块中
		if bc.CastHeight == castHeight {
			if bc.PrevHash != preHash {
				log.Printf("preHash diff found, bc.preHash=%v, preHash=%v!!!!!!!!!!!\n", bc.PrevHash, preHash)
			}
			log.Printf("already in casting height %v\n", castHeight)
		} else if bc.CastHeight > castHeight {
			log.Printf("already in casting higher block, current castHeight=%v, request castHeight=%v\n", bc.CastHeight, castHeight)
			return false
		} else {
			bc.castRebase(castHeight, preTime, preHash, false)
		}
	} else { //不在铸块, 则开启铸块
		bc.castRebase(castHeight, preTime, preHash, true)
	}

	return true
}

//收到某个铸块人的铸块完成消息（个人铸块完成消息也是个人验证完成消息）
func (bc *BlockContext) UserCasted(bh core.BlockHeader, sd SignData) CAST_BLOCK_MESSAGE_RESULT {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	if !bc.isCasting() {
		return CMBR_IGNORE_NOT_CASTING
	}
	result := bc.acceptCV(bh, sd)
	return result
}

//收到某个验证人的验证完成消息（可能会比铸块完成消息先收到）
func (bc *BlockContext) UserVerified(bh core.BlockHeader, sd SignData) CAST_BLOCK_MESSAGE_RESULT {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	if !bc.isCasting() { //没有在组铸块共识窗口
		return CMBR_IGNORE_NOT_CASTING
	}
	result := bc.acceptCV(bh, sd) //>=0为消息正确接收
	return result
}

func (bc *BlockContext) VerifyGroupSign(cs ConsensusBlockSummary, pk groupsig.Pubkey) bool {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	//找到cs对应的槽
	i := bc.findCastSlot(cs.QueueNumber)
	if i >= 0/* && king */{
		b := bc.Slots[i].VerifyGroupSign(pk)
		return b
	}
	return false
}

func (bc *BlockContext) baseOnGeneisBlock() bool {
	geneis := bc.Proc.MainChain.QueryBlockByHeight(0)
    return bc.PrevHash == geneis.Hash
}

//计算当前铸块人位置和QN
func (bc *BlockContext) calcCastor() (int32, int64) {
	var index int32 = -1
	var qn int64 = -1
	d := time.Since(bc.PreTime)

	max := bc.getMaxCastTime()

	var secs = int64(d.Seconds())
	if secs < max { //在组铸块共识时间窗口内
		qn = bc.calcQN()
		if qn < 0 {
			log.Printf("calcCastor qn negative found! qn=%v\n", qn)
		}
		log.Println("ttttttttttt", "d", d, "pretime", bc.PreTime, "secs", secs, "MAXTIME", uint64(max), "qn", qn)
		log.Println("ttttttttttt","prehash", bc.PrevHash, "castheight", bc.CastHeight)
		firstKing := bc.getFirstCastor() //取得第一个铸块人位置
		log.Printf("mem_count=%v, first King pos=%v, qn=%v, cur King pos=%v.\n", bc.GroupMembers, firstKing, qn, int64(firstKing)+qn)
		if firstKing >= 0 && bc.GroupMembers > 0 {
			index = int32((qn + int64(firstKing)) % int64(bc.GroupMembers))
			log.Printf("real cur King pos(MOD mem_count)=%v.\n", index)
		} else {
			qn = -1
		}
	} else {
		log.Printf("bc::calcCastor failed, out of group max cast time, PreTime=%v, escape seconds=%v.!!!\n",
			bc.PreTime.Format(time.Stamp), secs)
	}
	return index, qn
}

//取得第一个铸块人在组内的位置
func (bc *BlockContext) getFirstCastor() int32 {
	var index int32 = -1
	bi_hash := bc.PrevHash.Big()
	if bi_hash.BitLen() > 0 && bc.GroupMembers > 0 {
		index = int32(bi_hash.Mod(bi_hash, big.NewInt(int64(bc.GroupMembers))).Int64())
	}
	return index
}

func (bc *BlockContext) getMaxCastTime() int64 {
    var max int64
	if bc.baseOnGeneisBlock() {
		max = math.MaxInt64
	} else {
		preBH := bc.Proc.MainChain.QueryBlockByHash(bc.PrevHash)
		if preBH == nil {
			log.Printf("[ERROR]getMaxCastTime: query pre blockheader fail! bc.castHeight=%v, bc.preHash=%v\n", bc.CastHeight, bc.PrevHash)
			return 0
		}

		max = int64(bc.CastHeight - preBH.Height) * int64(MAX_GROUP_BLOCK_TIME)
	}
    log.Printf("getMaxCastTime calc max time = %v sec\n", max)
	return max
}


//定时器例行处理
//如果返回false, 则关闭定时器
func (bc *BlockContext) kingTickerRoutine() bool {
	log.Printf("proc(%v) begin kingTickerRoutine, time=%v...\n", bc.Proc.getPrefix(), time.Now().Format(time.Stamp))
	bc.lock.Lock()
	defer bc.lock.Unlock()


	if !bc.isCasting() || bc.maxQNCasted() { //没有在组铸块共识中或已经出最高qn块
		log.Printf("proc(%v) not in casting, reset and direct return. consensus status=%v.\n", bc.Proc.getPrefix(), bc.ConsensusStatus)
		bc.reset() //提前出块完成
		return false
	}

	d := time.Since(bc.PreTime)                  //上个铸块完成到现在的时间
	max := bc.getMaxCastTime()

	if int64(d.Seconds()) >= max { //超过了组最大铸块时间
		log.Printf("proc(%v) end kingTickerRoutine, out of max group cast time, time=%v secs, status=%v.\n", bc.Proc.getPrefix(), d.Seconds(), bc.ConsensusStatus)
		//bc.reset()
		bc.ConsensusStatus = CBCS_TIMEOUT
		return false
	} else {
		//当前组仍在有效铸块共识时间内
		//检查自己是否成为铸块人
		index, qn := bc.calcCastor() //当前铸块人（KING）和QN值
		bc.Proc.checkCastRoutine(bc, index, qn)
		log.Printf("proc(%v) end kingTickerRoutine, KING_POS=%v, qn=%v.\n", bc.Proc.getPrefix(), index, qn)
		return true
	}
	return true
}
