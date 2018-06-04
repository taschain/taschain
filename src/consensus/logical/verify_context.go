package logical

import (
	"time"
	"common"
	"log"
	"math"
	"core"
	"sync"
)

/*
**  Creator: pxf
**  Date: 2018/5/29 上午10:19
**  Description: 
*/

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

type VerifyContext struct {
	prevTime    time.Time
	prevHash    common.Hash
	castHeight  uint64
	signedMaxQN int64

	consensusStatus CAST_BLOCK_CONSENSUS_STATUS //铸块状态

	slots [MAX_SYNC_CASTORS]*SlotContext

	blockCtx *BlockContext

	lock sync.Mutex
}

func newVerifyContext(bc *BlockContext, castHeight uint64, preTime time.Time, preHash common.Hash) *VerifyContext {
	ctx := &VerifyContext{}
	ctx.rebase(bc, castHeight, preTime, preHash)
	return ctx
}

func (vc *VerifyContext) resetSlotContext()  {
	for i := 0; i < MAX_SYNC_CASTORS; i++ {
		sc := new(SlotContext)
		sc.reset()
		vc.slots[i] = sc
	}
}

func (vc *VerifyContext) isCasting() bool {
	if vc.consensusStatus == CBCS_IDLE || vc.consensusStatus == CBCS_TIMEOUT {
		//空闲，已出权重最高的块，超时
		return false
	} else {
		return true
	}
}

func (vc *VerifyContext) maxQNCasted() bool {
	return vc.consensusStatus == CBCS_MAX_QN_BLOCKED
}



func (vc *VerifyContext) rebase(bc *BlockContext, castHeight uint64, preTime time.Time, preHash common.Hash)  {
    vc.prevTime = preTime
    vc.prevHash = preHash
    vc.castHeight = castHeight
    vc.signedMaxQN = INVALID_QN
    vc.consensusStatus = CBCS_CURRENT
    vc.blockCtx = bc
	vc.resetSlotContext()
}

func (vc *VerifyContext) setTimeout()  {
    vc.consensusStatus = CBCS_TIMEOUT
}

func (vc *VerifyContext) baseOnGeneisBlock() bool {
	return vc.castHeight == 1
}

func (vc *VerifyContext) getMaxCastTime() int64 {
	var max int64
	if vc.baseOnGeneisBlock() {
		max = math.MaxInt64
	} else {
		preBH := vc.blockCtx.Proc.getBlockHeaderByHash(vc.prevHash)
		if preBH == nil {//TODO: handle preblock is nil
			log.Printf("[ERROR]getMaxCastTime: query pre blockheader fail! vctx.castHeight=%v, vctx.prevHash=%v\n", vc.castHeight, vc.prevHash)
			panic("[ERROR]getMaxCastTime: query pre blockheader nil!!!")
		}

		max = int64(vc.castHeight - preBH.Height) * int64(MAX_GROUP_BLOCK_TIME)
	}
	log.Printf("getMaxCastTime calc max time = %v sec\n", max)
	return max
}

//计算QN
func (vc *VerifyContext) calcQN() int64 {
	max := vc.getMaxCastTime()
	diff := time.Since(vc.prevTime).Seconds() //从上个铸块完成到现在的时间（秒）
	log.Printf("calcQN, time_begin=%v, diff=%v, max=%v.\n", vc.prevTime.Format(time.Stamp), diff, max)

	var qn int64
	if vc.baseOnGeneisBlock() {
		qn = max / int64(MAX_USER_CAST_TIME) - int64(diff) / int64(MAX_USER_CAST_TIME)
	} else {
		d := int64(diff) + int64(MAX_GROUP_BLOCK_TIME) - max
		qn = int64(MAX_QN) - d / int64(MAX_USER_CAST_TIME)
	}

	return qn
}

//检查是否有指定QN值的铸块槽
//返回：int32:铸块槽序号（没找到返回-1），bool：该铸块槽是否收到出块人消息（在铸块槽序号>=0时有意义）
func (vc *VerifyContext) findCastSlot(qn int64) (int32) {
	for i, v := range vc.slots {
		if v != nil && v.QueueNumber == qn {
			return int32(i)
		}
	}
	return -1
}

//检查目前在处理中的QN值最高的铸块槽。
//返回QN值最高的铸块槽的序号和QN值。如果当前全部是空槽，序号和QN值都返回-1.
func (vc *VerifyContext) findMinQNSlot() (int32, int64) {
	var index int32 = -1
	var minQN int64 = math.MaxInt64
	for i, v := range vc.slots {
		if v.QueueNumber < minQN {
			minQN = v.QueueNumber
			index = int32(i)
		}
	}
	return index, minQN
}

//检查是否有空槽可以接纳一个铸块槽
//如果还有空槽，返回空槽序号。如果没有空槽，返回-1.
func (vc *VerifyContext) findEmptySlot() int32 {
	for i, v := range vc.slots {
		if v.QueueNumber == INVALID_QN {
			return int32(i)
		}
	}
	return -1
}

//检查是否要处理某个铸块槽
//返回true需要处理，返回false可以丢弃。
func (vc *VerifyContext) needHandleQN(qn int64) bool {
	if vc.signedMaxQN == INVALID_QN { //当前该组还没有铸出过块
		return true
	} else { //当前该组已经有成功的铸块（来自某个铸块槽）
		return qn > vc.signedMaxQN
	}
}

//完成（上链，向组外广播）某个铸块槽后更新当前高度的最小QN值
func (vc *VerifyContext) signedUpdateMinQN(qn int64) bool {
	b := vc.needHandleQN(qn)
	if b {
		vc.signedMaxQN = qn
	}
	return b
}

//根据QN优先级规则，尝试找到有效的插槽
func (vc *VerifyContext) consensusFindSlot(qn int64) (idx int32, ret QN_QUERY_SLOT_RESULT) {
	var minQN int64 = -1

	i := vc.findCastSlot(qn)
	if i >= 0 { //该qn的槽已存在
		log.Printf("exist slot qn=%v, msg_count=%v.\n", qn, vc.slots[i].MessageSize())
		return i, QQSR_EXIST_SLOT
	} else {
		i = vc.findEmptySlot()
		if i >= 0 { //找到空槽
			log.Printf("found empty slot_index=%v.\n", i)
			return i, QQSR_EMPTY_SLOT
		} else {
			i, minQN = vc.findMinQNSlot() //取得最小槽
			log.Printf("slot fulled, exist minQN=%v, slot_index=%v, new_qn=%v.\n", minQN, i, qn)
			if qn > minQN { //最小槽的QN比新的QN小, 替换之
				return i, QQSR_REPLACE_SLOT
			}
		}
	}
	return -1, QQSR_NO_UNKNOWN
}

func (vc *VerifyContext) Rebase(bc *BlockContext, castHeight uint64, preTime time.Time, preHash common.Hash)  {
	vc.lock.Lock()
	defer vc.lock.Unlock()
	vc.rebase(bc, castHeight, preTime, preHash)
}

func (vc *VerifyContext) GetSlotByQN(qn int64) *SlotContext {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	i := vc.findCastSlot(qn)
	if i >= 0 {
		return vc.slots[i]
	} else {
		return nil
	}
}


//铸块共识消息处理函数
//cv：铸块共识数据，出块消息或验块消息生成的ConsensusBlockSummary.
//=0, 接受; =1,接受，达到阈值；<0, 不接受。
func (vc *VerifyContext) acceptCV(bh *core.BlockHeader, si *SignData) CAST_BLOCK_MESSAGE_RESULT {
	log.Printf("begin VerifyContext::acceptCV, height=%v, qn=%v...\n", bh.Height, bh.QueueNumber)
	idPrefix := vc.blockCtx.Proc.getPrefix()
	calcQN := vc.calcQN()
	if calcQN < 0 || bh.QueueNumber < 0 { //时间窗口异常
		log.Printf("proc(%v) acceptCV failed(time windwos ERROR), calcQN=%v, qn=%v.\n", idPrefix, calcQN, bh.QueueNumber)
		return CBMR_ERROR_ARG
	}
	if uint64(calcQN) > bh.QueueNumber { //未轮到该QN出块
		log.Printf("proc(%v) acceptCV failed(qn ERROR), calcQN=%v, qn=%v.\n", idPrefix, calcQN, bh.QueueNumber)
		return CMBR_IGNORE_QN_FUTURE
	}

	if !vc.needHandleQN(int64(bh.QueueNumber)) { //该组已经铸出过QN值更大的块
		return CMBR_IGNORE_MAX_QN_SIGNED
	}

	i, info := vc.consensusFindSlot(int64(bh.QueueNumber))
	log.Printf("proc(%v) consensusFindSlot, qn=%v, i=%v, info=%v.\n", idPrefix, bh.QueueNumber, i, info)
	if i < 0 { //没有找到有效的插槽
		return CMBR_IGNORE_QN_BIG_QN
	}
	//找到有效的插槽
	if info == QQSR_EMPTY_SLOT || info == QQSR_REPLACE_SLOT {
		log.Printf("proc(%v) put new_qn=%v in slot[%v], REPLACE=%v.\n", idPrefix, bh.QueueNumber, i, info == QQSR_REPLACE_SLOT)
		vc.slots[i] = newSlotContext(bh, si)
		if vc.slots[i].TransFulled {
			return CBMR_PIECE_NORMAL
		} else {
			return CBMR_PIECE_LOSINGTRANS
		}

	} else { //该QN值对应的插槽已存在
		if vc.slots[i].IsFailed() {
			return CBMR_STATUS_FAIL
		}
		result := vc.slots[i].AcceptPiece(*bh, *si)
		log.Printf("proc(%v) bc::slot[%v] AcceptPiece result=%v, msg_count=%v.\n", idPrefix, i, result, vc.slots[i].MessageSize())
		return result
	}
	return CMBR_ERROR_UNKNOWN
}

//完成某个铸块槽的铸块（上链，组外广播）后，更新组的当前高度铸块状态
func (vc *VerifyContext) CastedUpdateStatus(qn int64) bool {
	log.Printf("castedUpdateStatus before status=%v, qn=%v\n", vc.consensusStatus, qn)
	vc.lock.Lock()
	defer vc.lock.Unlock()

	vc.signedUpdateMinQN(qn)

	switch vc.consensusStatus {
	case CBCS_IDLE, CBCS_TIMEOUT, CBCS_MAX_QN_BLOCKED:	//不在铸块周期或已经铸出最大块
		return false
	case CBCS_CASTING, CBCS_CURRENT, CBCS_BLOCKED:
		if qn >= int64(MAX_QN) {
			vc.consensusStatus = CBCS_MAX_QN_BLOCKED
		} else {
			vc.consensusStatus = CBCS_BLOCKED
		}
		return true
	default:
		return true
	}

}

//收到某个验证人的验证完成消息（可能会比铸块完成消息先收到）
func (vc *VerifyContext) UserVerified(bh *core.BlockHeader, sd *SignData) CAST_BLOCK_MESSAGE_RESULT {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	result := vc.acceptCV(bh, sd) //>=0为消息正确接收
	return result
}

//（网络接收）新到交易集通知
//返回不再缺失交易的QN槽列表
func (vc *VerifyContext) ReceiveTrans(ths []common.Hash) []*SlotContext {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	slots := make([]*SlotContext, 0)
	for _, v := range vc.slots {
		if v != nil && v.QueueNumber != INVALID_QN {
			before, after := v.ReceTrans(ths)
			if !before && after { //该插槽已不再有缺失的交易
				slots = append(slots, v)
			}
		}
	}
	return slots
}

//判断该context是否可以删除
func (vc *VerifyContext) ShouldRemove(topHeight uint64) bool {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	//不在铸块或者已出最大块的, 可以删除
	if !vc.isCasting() || vc.maxQNCasted() {
		return true
	}
	allFinished := true
	//所有的槽都失败或者已验证的, 可以删除
	for _, slt := range vc.slots{
		if !slt.IsFailed() && !slt.IsVerified() {
			allFinished = false
			break
		}
	}
	if allFinished {
		return true
	}
	//铸过块, 且高度已经低于10块的, 可以删除
	if vc.signedMaxQN != INVALID_QN && vc.castHeight+10 < topHeight {
		return true
	}

	//未出过块, 但高度已经低于200块, 可以删除
	if vc.castHeight+200 < topHeight {
		return true
	}
	return false
}
