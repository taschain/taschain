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
	"sync"
	"middleware/types"
	"strconv"
	"consensus/model"
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
	CBMR_IGNORE_KING_ERROR                                     //丢弃：king错误
	CBMR_STATUS_FAIL                                           //已经失败的
	CBMR_ERROR_UNKNOWN                                         //异常：未知异常
	CBMR_CAST_SUCCESS											//铸块成功
	CBMR_BH_HASH_DIFF											//slot已经被替换过了
	CBMR_VERIFY_TIMEOUT											//已超时
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
	case CBMR_VERIFY_TIMEOUT:
		return "验证超时"
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
	//castedQNs []int64 //自己铸过的qn
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
		//castedQNs:       make([]int64, 0),
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


func (vc *VerifyContext) markTimeout() {
	atomic.StoreInt32(&vc.consensusStatus, CBCS_TIMEOUT)
}

func (vc *VerifyContext) markCastSuccess() {
	atomic.StoreInt32(&vc.consensusStatus, CBCS_BLOCKED)
}

func (vc *VerifyContext) castExpire() bool {
    return time.Now().After(vc.expireTime)
}

func (vc *VerifyContext) findSlot(hash common.Hash) int {
	for idx, slot := range vc.slots {
		if slot.BH.Hash == hash {
			return idx
		}
	}
	return -1
}

//根据QN优先级规则，尝试找到有效的插槽
func (vc *VerifyContext) consensusFindSlot(bh *types.BlockHeader) (idx int, ret QN_QUERY_SLOT_RESULT) {
	vc.lock.RLock()
	defer vc.lock.RUnlock()

	idx = vc.findSlot(bh.Hash)
	if idx >= 0 {
		return idx, QQSR_EXIST_SLOT
	}

	for idx, slot := range vc.slots {
		if !slot.IsValid() {
			return idx, QQSR_EMPTY_SLOT
		}
	}
	for idx, slot := range vc.slots {
		if slot.IsFailed() {
			return idx, QQSR_REPLACE_SLOT
		}
	}
	var (
		maxV uint64 = 0
		index int = -1
	)

	for idx, slot := range vc.slots {
		if slot.vrfValue > maxV {
			maxV = slot.vrfValue
			index = idx
		}
	}
	return index, QQSR_REPLACE_SLOT
}

func (vc *VerifyContext) GetSlotByHash(hash common.Hash) *SlotContext {
	vc.lock.RLock()
	defer vc.lock.RUnlock()

	if i := vc.findSlot(hash); i >= 0 {
		return vc.slots[i]
	}
	return nil
}

func (vc *VerifyContext) replaceSlot(idx int, bh *types.BlockHeader, threshold int)  {
    vc.lock.Lock()
    defer vc.lock.Unlock()
    slot := initSlotContext(bh, threshold)
    vc.slots[idx] = slot
}

func (vc *VerifyContext) getSlot(idx int) *SlotContext {
    vc.lock.RLock()
    defer vc.lock.RUnlock()
    return vc.slots[idx]
}


//收到某个验证人的验证完成消息（可能会比铸块完成消息先收到）
func (vc *VerifyContext) UserVerified(bh *types.BlockHeader, signData *model.SignData, summary *model.CastGroupSummary) CAST_BLOCK_MESSAGE_RESULT {
	if bh.GenHash() != signData.DataHash {
		panic("acceptCV arg failed, hash not samed 1.")
	}
	if bh.Hash != signData.DataHash {
		panic("acceptCV arg failed, hash not samed 2")
	}

	idPrefix := vc.blockCtx.Proc.getPrefix()

	i, info := vc.consensusFindSlot(bh)
	newBizLog("UserVerified").log("proc(%v) consensusFindSlot, qn=%v, i=%v, info=%v.\n", idPrefix, bh.ProveValue, i, info)

	//找到有效的插槽
	if info == QQSR_EMPTY_SLOT || info == QQSR_REPLACE_SLOT {
		vc.replaceSlot(i, bh, vc.blockCtx.threshold())
	}
	//警惕并发
	slot := vc.getSlot(i)
	if slot.IsFailed() {
		return CBMR_STATUS_FAIL
	}
	result := slot.AcceptVerifyPiece(bh, signData)
	return result
}

//（网络接收）新到交易集通知
//返回不再缺失交易的QN槽列表
func (vc *VerifyContext) AcceptTrans(slot *SlotContext, ths []common.Hash) int8 {

	if !slot.IsValid() {
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

func (vc *VerifyContext) GetSlots() []*SlotContext {
	vc.lock.RLock()
	defer vc.lock.RUnlock()
	slots := make([]*SlotContext, len(vc.slots))
	copy(slots, vc.slots[:])
	return slots
}
