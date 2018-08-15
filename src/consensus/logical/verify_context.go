package logical

import (
	"time"
	"common"
	"log"
	"math"
	"sync"
	"math/big"
	"middleware/types"
	"encoding/binary"
	"strconv"
	"consensus/model"
	"consensus/base"
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

const (
	TRANS_INVALID_SLOT	int8	= iota	//无效验证槽
	TRANS_DENY					//拒绝该交易
	TRANS_ACCEPT_NOT_FULL		//接受交易, 但仍缺失交易
	TRANS_ACCEPT_FULL_THRESHOLD	//接受交易, 无缺失, 验证已达到门限
	TRANS_ACCEPT_FULL_PIECE		//接受交易, 无缺失, 未达到门限
)

func TRANS_ACCEPT_RESULT_DESC(ret int8) string {
	switch ret {
	case TRANS_INVALID_SLOT:
		return "验证槽无效"
	case TRANS_DENY:
		return "不接收该批交易"
	case TRANS_ACCEPT_NOT_FULL:
		return "接收交易,但仍缺失"
	case TRANS_ACCEPT_FULL_PIECE:
		return "交易收齐,等待分片"
	case TRANS_ACCEPT_FULL_THRESHOLD:
		return "交易收齐,分片已到门限"
	}
	return strconv.FormatInt(int64(ret), 10)
}

type VerifyContext struct {
	prevTime    time.Time
	prevHash    common.Hash
	prevRandSig	[]byte
	prevSign	[]byte
	castHeight  uint64
	signedMaxQN int64
	expireTime	time.Time			//铸块超时时间

	consensusStatus CAST_BLOCK_CONSENSUS_STATUS //铸块状态

	slots [model.MAX_CAST_SLOT]*SlotContext

	castedQNs []int64 //自己铸过的qn

	blockCtx *BlockContext

	lock sync.Mutex
}

func newVerifyContext(bc *BlockContext, castHeight uint64, expire time.Time, preBH *types.BlockHeader) *VerifyContext {
	ctx := &VerifyContext{}
	ctx.rebase(bc, castHeight, expire, preBH)
	return ctx
}

func (vc *VerifyContext) resetSlotContext() {
	for i := 0; i < model.MAX_CAST_SLOT; i++ {
		sc := new(SlotContext)
		sc.reset(vc.blockCtx.threshold())
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
	return vc.consensusStatus == CBCS_MAX_QN_BLOCKED || vc.signedMaxQN >= int64(model.Param.MaxQN)
}

func (vc *VerifyContext) isQNCasted(qn int64) bool {
	for _, _qn := range vc.castedQNs {
		if _qn == qn {
			return true
		}
	}
	return false
}

func (vc *VerifyContext) addCastedQN(qn int64) {
	vc.castedQNs = append(vc.castedQNs, qn)
}

func (vc *VerifyContext) rebase(bc *BlockContext, castHeight uint64, expire time.Time, preBH *types.BlockHeader)  {
    vc.prevTime = preBH.CurTime
    vc.prevHash = preBH.Hash
    vc.castHeight = castHeight
    vc.prevSign = preBH.Signature
    vc.prevRandSig = preBH.RandSig
    vc.signedMaxQN =  model.INVALID_QN
    vc.blockCtx = bc
	vc.expireTime = expire
	vc.consensusStatus = CBCS_CURRENT
	vc.castedQNs = make([]int64, 0)
	vc.resetSlotContext()
}

func (vc *VerifyContext) setTimeout() {
	vc.consensusStatus = CBCS_TIMEOUT
}

//func (vc *VerifyContext) baseOnGeneisBlock() bool {
//	return vc.castHeight == 1
//}

func (vc *VerifyContext) castExpire() bool {
    return time.Now().After(vc.expireTime)
}

//func (vc *VerifyContext) getMaxCastTime() int64 {
//	var max int64
//	defer func() {
//		log.Printf("getMaxCastTime calc max time = %v sec\n", max)
//	}()
//
//	//if vc.baseOnGeneisBlock() {
//	//	max = math.MaxInt64
//	//} else {
//	//	preBH := vc.blockCtx.Proc.getBlockHeaderByHash(vc.prevHash)
//	//	if preBH == nil {//TODO: handle preblock is nil. 有可能分叉处理, 把pre块删掉了
//	//		log.Printf("[ERROR]getMaxCastTime: query pre blockheader fail! vctx.castHeight=%v, vctx.prevHash=%v\n", vc.castHeight, GetHashPrefix(vc.prevHash))
//	//		//panic("[ERROR]getMaxCastTime: query pre blockheader nil!!!")
//	//		max = -1
//	//	} else {
//	//		max = int64(vc.castHeight - preBH.Height) * int64(MAX_GROUP_BLOCK_TIME)
//	//	}
//	//
//	//}
//	//
//	max = int64(vc.expireTime.Sub(vc.prevTime).Seconds())
//
//	return max
//}

//计算QN
func (vc *VerifyContext) calcQN(timeEnd time.Time) int64 {
	diff := timeEnd.Sub(vc.prevTime).Seconds() //从上个铸块完成到现在的时间（秒）
	return vc.qnOfDiff(diff)
}

func (vc *VerifyContext) qnOfDiff(diff float64) int64 {
	max := int64(vc.expireTime.Sub(vc.prevTime).Seconds())
	if max < 0 {
		return -1
	}
	d := int64(diff) + int64(model.Param.MaxGroupCastTime) - max
	qn := int64(model.Param.MaxQN) - d / int64(model.Param.MaxUserCastTime)

	//log.Printf("qnOfDiff diff %v, pre %v, d %v, qn=%v\n", int(diff), vc.prevTime, d, qn)
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
		if v.QueueNumber ==  model.INVALID_QN {
			return int32(i)
		}
	}
	return -1
}

//检查是否要处理某个铸块槽
//返回true需要处理，返回false可以丢弃。
func (vc *VerifyContext) needHandleQN(qn int64) bool {
	if vc.signedMaxQN == model.INVALID_QN { //当前该组还没有铸出过块
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
		return i, QQSR_EXIST_SLOT
	} else {
		i = vc.findEmptySlot()
		if i >= 0 { //找到空槽
			return i, QQSR_EMPTY_SLOT
		} else {
			i, minQN = vc.findMinQNSlot() //取得最小槽
			if qn > minQN { //最小槽的QN比新的QN小, 替换之
				return i, QQSR_REPLACE_SLOT
			}
		}
	}
	return -1, QQSR_NO_UNKNOWN
}

//func (vc *VerifyContext) Rebase(bc *BlockContext, castHeight uint64, preTime time.Time, preHash common.Hash)  {
//	vc.lock.Lock()
//	defer vc.lock.Unlock()
//	vc.rebase(bc, castHeight, preTime, preHash)
//}

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
//pow模式下， 此方法废弃
func (vc *VerifyContext) acceptCV2(bh *types.BlockHeader, si *model.SignData, summary *model.CastGroupSummary) CAST_BLOCK_MESSAGE_RESULT {
	idPrefix := vc.blockCtx.Proc.getPrefix()
	calcQN := vc.calcQN(bh.CurTime)
	if calcQN < 0 || uint64(calcQN) != bh.QueueNumber { //计算的qn错误
		log.Printf("calcQN %v, receiveQN %v\n", calcQN, bh.QueueNumber)
		return CBMR_IGNORE_QN_ERROR
	}

	calcKingPos := vc.getCastorPosByQN(calcQN)
	receiveKingPos := summary.CastorPos
	if calcKingPos != receiveKingPos { //该qn对应的king错误
		return CBMR_IGNORE_KING_ERROR
	}

	if !vc.needHandleQN(int64(bh.QueueNumber)) { //该组已经铸出过QN值更大的块
		return CBMR_IGNORE_MAX_QN_SIGNED
	}

	i, info := vc.consensusFindSlot(int64(bh.QueueNumber))
	log.Printf("proc(%v) consensusFindSlot, qn=%v, i=%v, info=%v.\n", idPrefix, bh.QueueNumber, i, info)
	if i < 0 { //没有找到有效的插槽
		return CBMR_IGNORE_QN_BIG_QN
	}
	//找到有效的插槽
	if info == QQSR_EMPTY_SLOT || info == QQSR_REPLACE_SLOT {
		vc.slots[i] = newSlotContext(bh, si, vc.blockCtx.threshold())
		if vc.slots[i].IsFailed() {
			return CBMR_STATUS_FAIL
		}
		return CBMR_PIECE_NORMAL
		//if vc.slots[i].transFulled {
		//	return CBMR_PIECE_NORMAL
		//} else {
		//	return CBMR_PIECE_LOSINGTRANS
		//}

	} else { //该QN值对应的插槽已存在
		if vc.slots[i].IsFailed() {
			return CBMR_STATUS_FAIL
		}
		result := vc.slots[i].AcceptPiece(*bh, *si)
		return result
	}
	return CBMR_ERROR_UNKNOWN
}

func (vc *VerifyContext) acceptCV(bh *types.BlockHeader, si *model.SignData, summary *model.CastGroupSummary) CAST_BLOCK_MESSAGE_RESULT {
	idPrefix := vc.blockCtx.Proc.getPrefix()



	i, info := vc.consensusFindSlot(int64(bh.QueueNumber))
	log.Printf("proc(%v) consensusFindSlot, qn=%v, i=%v, info=%v.\n", idPrefix, bh.QueueNumber, i, info)
	if i < 0 { //没有找到有效的插槽
		return CBMR_IGNORE_QN_BIG_QN
	}
	//找到有效的插槽
	if info == QQSR_EMPTY_SLOT || info == QQSR_REPLACE_SLOT {
		vc.slots[i] = newSlotContext(bh, si, vc.blockCtx.threshold())
		if vc.slots[i].IsFailed() {
			return CBMR_STATUS_FAIL
		}
		return CBMR_PIECE_NORMAL
		//if vc.slots[i].transFulled {
		//	return CBMR_PIoECE_NORMAL
		//} else {
		//	return CBMR_PIECE_LOSINGTRANS
		//}

	} else { //该QN值对应的插槽已存在
		if vc.slots[i].IsFailed() {
			return CBMR_STATUS_FAIL
		}
		result := vc.slots[i].AcceptPiece(*bh, *si)
		return result
	}
	return CBMR_ERROR_UNKNOWN
}

//完成某个铸块槽的铸块（上链，组外广播）后，更新组的当前高度铸块状态
func (vc *VerifyContext) CastedUpdateStatus(qn int64) bool {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	vc.signedUpdateMinQN(qn)

	switch vc.consensusStatus {
	case CBCS_IDLE, CBCS_TIMEOUT, CBCS_MAX_QN_BLOCKED: //不在铸块周期或已经铸出最大块
		return false
	case CBCS_CASTING, CBCS_CURRENT, CBCS_BLOCKED:
		if qn >= int64(model.Param.MaxQN) {
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
func (vc *VerifyContext) UserVerified(bh *types.BlockHeader, sd *model.SignData, summary *model.CastGroupSummary) CAST_BLOCK_MESSAGE_RESULT {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	result := vc.acceptCV(bh, sd, summary) //>=0为消息正确接收
	return result
}

//（网络接收）新到交易集通知
//返回不再缺失交易的QN槽列表
func (vc *VerifyContext) AcceptTrans(slot *SlotContext, ths []common.Hash) int8 {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	if slot.QueueNumber == int64(model.INVALID_QN) {
		return TRANS_INVALID_SLOT
	}
	accept := slot.AcceptTrans(ths)
	if !accept {
		return TRANS_DENY
	}
	if !slot.IsTransFull() {
		return TRANS_ACCEPT_NOT_FULL
	}
	if slot.thresholdWitnessGot() {
		return TRANS_ACCEPT_FULL_THRESHOLD
	} else {
		return TRANS_ACCEPT_FULL_PIECE
	}
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
	for _, slt := range vc.slots {
		if !slt.IsFailed() && !slt.IsVerified() {
			allFinished = false
			break
		}
	}
	if allFinished {
		return true
	}
	//铸过块, 且高度已经低于10块的, 可以删除
	if vc.signedMaxQN != model.INVALID_QN && vc.castHeight+10 < topHeight {
		return true
	}

	//未出过块, 但高度已经低于200块, 可以删除
	if vc.castHeight+200 < topHeight {
		return true
	}
	return false
}

//计算当前铸块人位置和QN
func (vc *VerifyContext) calcCastor() (int32, int64) {
	//if secs < max { //在组铸块共识时间窗口内
	qn := vc.calcQN(time.Now())
	if qn < 0 {
		return -1, qn
	}
	index := vc.getCastorPosByQN(qn)

	return index, qn
}

func (vc *VerifyContext) getCastorPosByQN(qn int64) int32 {
	//firstKing := vc.getFirstCastor(vc.prevHash) //取得第一个铸块人位置
	////log.Printf("mem_count=%v, first King pos=%v, qn=%v, cur King pos=%v.\n", bc.GroupMembers, firstKing, qn, int64(firstKing)+qn)
	//mem := vc.blockCtx.GroupMembers
	//if firstKing >= 0 {
	//	index := int32((qn + int64(firstKing)) % int64(mem))
	//	log.Printf("real King pos(MOD mem_count)=%v.\n", index)
	//	return index
	//} else {
	//	return -1
	//}
	secret := vc.blockCtx.getGroupSecret()
	if secret == nil {
		 return -1
	}
	data := secret.SecretSign
	data = append(data, vc.prevSign...)
	qnBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(qnBytes, uint64(qn))
	data = append(data, qnBytes...)
	hash := base.Data2CommonHash(data)
	biHash := hash.Big()

	var index int32 = -1
	mem := vc.blockCtx.GroupMembers
	if biHash.BitLen() > 0 {
		index = int32(biHash.Mod(biHash, big.NewInt(int64(mem))).Int64())
	}
	return index
}

//取得第一个铸块人在组内的位置
//deprecated
func (vc *VerifyContext) getFirstCastor(prevHash common.Hash) int32 {
	var index int32 = -1
	biHash := prevHash.Big()
	mem := vc.blockCtx.GroupMembers
	if biHash.BitLen() > 0 {
		index = int32(biHash.Mod(biHash, big.NewInt(int64(mem))).Int64())
	}
	return index
}
