//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package logical

import (
	"time"
	"common"
	"log"
	"sync"
	"math/big"
	"middleware/types"
	"encoding/binary"
	"strconv"
	"consensus/model"
	"consensus/base"
	"sync/atomic"
)

/*
**  Creator: pxf
**  Date: 2018/5/29 上午10:19
**  Description: 
*/


const (
	CBCS_IDLE    int32 = iota //非当前组
	CBCS_CASTING                                    //正在铸块
	CBCS_BLOCKED                                    //组内已有铸块完成（已通知到组外）
	CBCS_TIMEOUT                                    //组铸块超时
)


type CAST_BLOCK_MESSAGE_RESULT int8 //出块和验证消息处理结果枚举

const (
	CBMR_PIECE_NORMAL         CAST_BLOCK_MESSAGE_RESULT = iota //收到一个分片，接收正常
	CBMR_PIECE_LOSINGTRANS                                     //收到一个分片, 缺失交易
	CBMR_THRESHOLD_SUCCESS                                     //收到一个分片且达到阈值，组签名成功
	CBMR_THRESHOLD_FAILED                                      //收到一个分片且达到阈值，组签名失败
	CBMR_IGNORE_REPEAT                                         //丢弃：重复收到该消息
	CBMR_IGNORE_QN_BIG_QN                                      //丢弃：QN太大
	CBMR_IGNORE_QN_ERROR                                       //丢弃：qn错误
	CBMR_IGNORE_KING_ERROR                                     //丢弃：king错误
	CBMR_STATUS_FAIL                                           //已经失败的
	CBMR_ERROR_UNKNOWN                                         //异常：未知异常
	CBMR_CAST_SUCCESS											//铸块成功
	CBMR_BH_HASH_DIFF											//slot已经被替换过了
	CBMR_SLOT_INIT_FAIL											//slot初始化失败
	CBMR_SLOT_REPLACE_FAIL											//slot初始化失败
)

func CBMR_RESULT_DESC(ret CAST_BLOCK_MESSAGE_RESULT) string {
	switch ret {
	case CBMR_PIECE_NORMAL:
		return "正常分片"
	case CBMR_PIECE_LOSINGTRANS:
		return "交易缺失"
	case CBMR_THRESHOLD_SUCCESS:
		return "达到门限值组签名成功"
	case CBMR_THRESHOLD_FAILED:
		return "达到门限值但组签名失败"
	case CBMR_IGNORE_QN_BIG_QN, CBMR_IGNORE_QN_ERROR:
		return "qn错误"
	case CBMR_IGNORE_KING_ERROR:
		return "king错误"
	case CBMR_STATUS_FAIL:
		return "失败状态"
	case CBMR_IGNORE_REPEAT:
		return "重复消息"
	case CBMR_CAST_SUCCESS:
		return "已铸块成功"
	case CBMR_BH_HASH_DIFF:
		return "hash不一致，slot已无效"
	case CBMR_SLOT_INIT_FAIL:
		return "slot初始化失败"
	case CBMR_SLOT_REPLACE_FAIL:
		return "slot替换失败"
	}
	return strconv.FormatInt(int64(ret), 10)
}

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


type QN_QUERY_SLOT_RESULT int //根据QN查找插槽结果枚举

const (
	QQSR_EMPTY_SLOT   QN_QUERY_SLOT_RESULT = iota //找到一个空槽
	QQSR_REPLACE_SLOT                             //找到一个能替换（QN值更低）的槽
	QQSR_EXIST_SLOT                               //该QN对应的插槽已存在
)

type VerifyContext struct {
	prevBH 		*types.BlockHeader
	castHeight  uint64
	//signedMaxQN int64
	expireTime	time.Time			//铸块超时时间
	consensusStatus int32 //铸块状态
	slots [model.MAX_CAST_SLOT]*SlotContext
	castedQNs []int64 //自己铸过的qn
	blockCtx *BlockContext
	lock sync.RWMutex
}

func newVerifyContext(bc *BlockContext, castHeight uint64, expire time.Time, preBH *types.BlockHeader) *VerifyContext {
	ctx := &VerifyContext{
		prevBH:          preBH,
		castHeight:      castHeight,
		blockCtx:        bc,
		expireTime:      expire,
		consensusStatus: CBCS_CASTING,
		castedQNs:       make([]int64, 0),
	}
	for i := 0; i < model.MAX_CAST_SLOT; i++ {
		sc := createSlotContext(ctx.blockCtx.threshold())
		ctx.slots[i] = sc
	}
	return ctx
}


func (vc *VerifyContext) isCasting() bool {
	status := atomic.LoadInt32(&vc.consensusStatus)
	return !(status == CBCS_IDLE || status == CBCS_TIMEOUT)
}

func (vc *VerifyContext) castSuccess() bool {
	return atomic.LoadInt32(&vc.consensusStatus) == CBCS_BLOCKED
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

func (vc *VerifyContext) markTimeout() {
	atomic.StoreInt32(&vc.consensusStatus, CBCS_TIMEOUT)
}

func (vc *VerifyContext) markCastSuccess() {
	atomic.StoreInt32(&vc.consensusStatus, CBCS_BLOCKED)
}

func (vc *VerifyContext) castExpire() bool {
    return time.Now().After(vc.expireTime)
}


//计算QN
func (vc *VerifyContext) calcQN(timeEnd time.Time) int64 {
	diff := timeEnd.Sub(vc.prevBH.CurTime).Seconds() //从上个铸块完成到现在的时间（秒）
	return vc.qnOfDiff(diff)
}

func (vc *VerifyContext) qnOfDiff(diff float64) int64 {
	max := int64(vc.expireTime.Sub(vc.prevBH.CurTime).Seconds())
	if max < 0 {
		return -1
	}
	d := int64(diff) + int64(model.Param.MaxGroupCastTime) - max
	qn := int64(model.Param.MaxQN) - d / int64(model.Param.MaxUserCastTime)
	if qn < 0 {
		log.Printf("maxQN %v, d %v, max %v, diff %v, expire %v, (%v, %v)\n", model.Param.MaxQN, d, max, diff, vc.expireTime, model.Param.MaxGroupCastTime, model.Param.MaxUserCastTime)
	}
	return qn
}

func (vc *VerifyContext) findSlot(qn int64) int {
	for idx, slot := range vc.slots {
		if slot.QueueNumber == qn {
			return idx
		}
	}
	return -1
}

//根据QN优先级规则，尝试找到有效的插槽
func (vc *VerifyContext) consensusFindSlot(qn int64) (*SlotContext, QN_QUERY_SLOT_RESULT, int) {
	vc.lock.RLock()
	defer vc.lock.RUnlock()

	idx := vc.findSlot(qn)
	if idx >= 0 {
		return vc.slots[idx], QQSR_EXIST_SLOT, idx
	}

	for idx, slot := range vc.slots {
		if !slot.IsValid() {
			return vc.slots[idx], QQSR_EMPTY_SLOT, idx
		}
	}
	for idx, slot := range vc.slots {
		if slot.IsFailed() {
			return vc.slots[idx], QQSR_REPLACE_SLOT, idx
		}
	}
	var (
		minQN int64 = common.MaxInt64
	)

	for i, slot := range vc.slots {
		if slot.QueueNumber < minQN {
			minQN = slot.QueueNumber
			idx = i
		}
	}
	return vc.slots[idx], QQSR_REPLACE_SLOT, idx
}

func (vc *VerifyContext) GetSlotByQN(qn int64) *SlotContext {
	vc.lock.RLock()
	defer vc.lock.RUnlock()

	if i := vc.findSlot(qn); i >= 0 {
		return vc.slots[i]
	}
	return nil
}

func (vc *VerifyContext) replaceSlot(idx int, old *SlotContext, bh *types.BlockHeader) *SlotContext {
	if old.BH.Hash == bh.Hash {
		return old
	}
	slot := createSlotContext(vc.blockCtx.threshold())
	slot.init(bh)
	vc.lock.Lock()
	defer vc.lock.Unlock()
	if vc.slots[idx].BH.Hash != bh.Hash {
		vc.slots[idx] = slot
		return slot
	}
	return nil
}

func (vc *VerifyContext) getSlot(idx int) *SlotContext {
    vc.lock.RLock()
    defer vc.lock.RUnlock()
    return vc.slots[idx]
}


//收到某个验证人的验证完成消息（可能会比铸块完成消息先收到）
func (vc *VerifyContext) UserVerified(bh *types.BlockHeader, signData *model.SignData, summary *model.CastGroupSummary) CAST_BLOCK_MESSAGE_RESULT {
	if vc.castSuccess() {
		return CBMR_CAST_SUCCESS
	}
	if bh.GenHash() != signData.DataHash {
		panic("acceptCV arg failed, hash not samed 1.")
	}
	if bh.Hash != signData.DataHash {
		panic("acceptCV arg failed, hash not samed 2")
	}

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

	slot, info, idx := vc.consensusFindSlot(int64(bh.QueueNumber))
	log.Printf("proc(%v) consensusFindSlot, height=%v, qn=%v, slotStatus=%v, info=%v, idx=%v.\n", idPrefix, bh.Height, bh.QueueNumber, slot.GetSlotStatus(), info, idx)

	//找到有效的插槽
	if info == QQSR_EMPTY_SLOT {
		if !slot.init(bh) {
			log.Printf("initSlotContext fail, status=%v", slot.GetSlotStatus())
			return CBMR_SLOT_INIT_FAIL
		}
	} else if info == QQSR_REPLACE_SLOT {
		slot = vc.replaceSlot(idx, slot, bh)
		if slot == nil {
			log.Printf("replaceSlot fail")
			return CBMR_SLOT_REPLACE_FAIL
		}
	}
	//警惕并发
	if slot.IsFailed() {
		return CBMR_STATUS_FAIL
	}
	result := slot.AcceptPiece(bh, signData)
	return result
}

//（网络接收）新到交易集通知
//返回不再缺失交易的QN槽列表
func (vc *VerifyContext) AcceptTrans(slot *SlotContext, ths []common.Hash) int8 {

	if slot.QueueNumber == int64(model.INVALID_QN) {
		return TRANS_INVALID_SLOT
	}
	accept := slot.AcceptTrans(ths)
	if !accept {
		return TRANS_DENY
	}
	if slot.HasTransLost() {
		return TRANS_ACCEPT_NOT_FULL
	}
	st := slot.GetSlotStatus()

	if st == SS_RECOVERD || st == SS_VERIFIED {
		return TRANS_ACCEPT_FULL_THRESHOLD
	} else {
		return TRANS_ACCEPT_FULL_PIECE
	}
}

//判断该context是否可以删除
func (vc *VerifyContext) shouldRemove(topHeight uint64) bool {
	//不在铸块的, 可以删除
	if !vc.isCasting() {
		return true
	}

	//铸过块, 且已经超时的， 可以删除
	if vc.castSuccess() && vc.castExpire() {
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
	secret := vc.blockCtx.getGroupSecret()
	if secret == nil {
		 return -1
	}
	data := secret.SecretSign
	data = append(data, vc.prevBH.Random...)
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

func (vc *VerifyContext) GetSlots() []*SlotContext {
	vc.lock.RLock()
	defer vc.lock.RUnlock()
	slots := make([]*SlotContext, len(vc.slots))
	copy(slots, vc.slots[:])
	return slots
}
