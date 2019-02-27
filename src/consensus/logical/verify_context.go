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
	"common"
	"consensus/model"
	"middleware/types"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"consensus/base"
	"fmt"
	"consensus/groupsig"
)

/*
**  Creator: pxf
**  Date: 2018/5/29 上午10:19
**  Description:
 */

const (
	CBCS_IDLE      int32 = iota //非当前组
	CBCS_CASTING                //正在铸块
	CBCS_BLOCKED                //组内已有铸块完成
	CBCS_BROADCAST              //块已广播
	CBCS_TIMEOUT                //组铸块超时
)

type CAST_BLOCK_MESSAGE_RESULT int8 //出块和验证消息处理结果枚举

const (
	CBMR_PIECE_NORMAL      CAST_BLOCK_MESSAGE_RESULT = iota //收到一个分片，接收正常
	CBMR_PIECE_LOSINGTRANS                                  //收到一个分片, 缺失交易
	CBMR_THRESHOLD_SUCCESS                                  //收到一个分片且达到阈值，组签名成功
	CBMR_THRESHOLD_FAILED                                   //收到一个分片且达到阈值，组签名失败
	CBMR_IGNORE_REPEAT                                      //丢弃：重复收到该消息
	CBMR_IGNORE_KING_ERROR                                  //丢弃：king错误
	CBMR_STATUS_FAIL                                        //已经失败的
	CBMR_ERROR_UNKNOWN                                      //异常：未知异常
	CBMR_CAST_SUCCESS                                       //铸块成功
	CBMR_BH_HASH_DIFF                                       //slot已经被替换过了
	CBMR_VERIFY_TIMEOUT                                     //已超时
	CBMR_SLOT_INIT_FAIL                                     //slot初始化失败
	CBMR_SLOT_REPLACE_FAIL                                  //slot初始化失败
	CBMR_SIGNED_MAX_QN										//签过更高的qn
	CBMR_SIGN_VERIFY_FAIL										//签名错误
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
	case CBMR_SLOT_INIT_FAIL:
		return "slot初始化失败"
	case CBMR_SLOT_REPLACE_FAIL:
		return "slot替换失败"
	case CBMR_SIGNED_MAX_QN:
		return "签过更高qn"

	}
	return strconv.FormatInt(int64(ret), 10)
}

const (
	TRANS_INVALID_SLOT          int8 = iota //无效验证槽
	TRANS_DENY                              //拒绝该交易
	TRANS_ACCEPT_NOT_FULL                   //接受交易, 但仍缺失交易
	TRANS_ACCEPT_FULL_THRESHOLD             //接受交易, 无缺失, 验证已达到门限
	TRANS_ACCEPT_FULL_PIECE                 //接受交易, 无缺失, 未达到门限
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

type VerifyContext struct {
	prevBH     *types.BlockHeader
	castHeight uint64
	signedMaxQN uint64
	createTime      time.Time
	expireTime      time.Time //铸块超时时间
	consensusStatus int32     //铸块状态
	slots           map[common.Hash]*SlotContext
	broadcastSlot 	*SlotContext
	//castedQNs []int64 //自己铸过的qn
	blockCtx *BlockContext
	signedNum 	int32
	lock     sync.RWMutex
}

func newVerifyContext(bc *BlockContext, castHeight uint64, expire time.Time, preBH *types.BlockHeader) *VerifyContext {
	ctx := &VerifyContext{
		prevBH:          preBH,
		castHeight:      castHeight,
		blockCtx:        bc,
		expireTime:      expire,
		createTime:      time.Now(),
		consensusStatus: CBCS_CASTING,
		signedMaxQN: 	0,
		slots:           make(map[common.Hash]*SlotContext),
		//castedQNs:       make([]int64, 0),
	}
	return ctx
}

func (vc *VerifyContext) increaseSignedNum()  {
    atomic.AddInt32(&vc.signedNum, 1)
}

func (vc *VerifyContext) isCasting() bool {
	status := atomic.LoadInt32(&vc.consensusStatus)
	return !(status == CBCS_IDLE || status == CBCS_TIMEOUT)
}

func (vc *VerifyContext) castSuccess() bool {
	return atomic.LoadInt32(&vc.consensusStatus) == CBCS_BLOCKED
}
func (vc *VerifyContext) broadCasted() bool {
	return atomic.LoadInt32(&vc.consensusStatus) == CBCS_BROADCAST
}

func (vc *VerifyContext) markTimeout() {
	if !vc.castSuccess() && !vc.broadCasted() {
		atomic.StoreInt32(&vc.consensusStatus, CBCS_TIMEOUT)
	}
}

func (vc *VerifyContext) markCastSuccess() {
	atomic.StoreInt32(&vc.consensusStatus, CBCS_BLOCKED)
}

func (vc *VerifyContext) markBroadcast() bool {
	return atomic.CompareAndSwapInt32(&vc.consensusStatus, CBCS_BLOCKED, CBCS_BROADCAST)
}

//铸块是否过期
func (vc *VerifyContext) castExpire() bool {
	return time.Now().After(vc.expireTime)
}

//分红交易签名是否过期
func (vc *VerifyContext) castRewardSignExpire() bool {
	return time.Now().After(vc.expireTime.Add(time.Duration(30*model.Param.MaxGroupCastTime)*time.Second))
}

func (vc *VerifyContext) findSlot(hash common.Hash) *SlotContext {
	if sc, ok := vc.slots[hash]; ok {
		return sc
	}
	return nil
}

func (vc *VerifyContext) getSignedMaxQN() uint64 {
 	return atomic.LoadUint64(&vc.signedMaxQN)
}

func (vc *VerifyContext) hasSignedBiggerQN(totalQN uint64) bool {
	return vc.getSignedMaxQN() > totalQN
}

func (vc *VerifyContext) updateSignedMaxQN(totalQN uint64) bool {
	if vc.getSignedMaxQN() < totalQN {
		atomic.StoreUint64(&vc.signedMaxQN, totalQN)
		return true
	}
	return false
}

func (vc *VerifyContext) baseCheck(bh *types.BlockHeader, sender groupsig.ID) (slot *SlotContext, err error) {
	//只签qn不小于已签出的最高块的块
	if vc.hasSignedBiggerQN(bh.TotalQN) {
		err = fmt.Errorf("已签过更高qn块%v,本块qn%v", vc.getSignedMaxQN(), bh.TotalQN)
		return
	}

	if vc.castSuccess() || vc.broadCasted() {
		err = fmt.Errorf("已出块")
		return
	}
	if vc.castExpire() {
		vc.markTimeout()
		err = fmt.Errorf("已超时" + vc.expireTime.String())
		return
	}
	slot = vc.GetSlotByHash(bh.Hash)
	if slot != nil {
		if slot.GetSlotStatus() >= SS_RECOVERD {
			err = fmt.Errorf("slot不接受piece，状态%v", slot.slotStatus)
			return
		}
		if _, ok := slot.gSignGenerator.GetWitness(sender); ok {
			err = fmt.Errorf("重复消息%v", sender.ShortS())
			return
		}
	}

	return
}

func (vc *VerifyContext) GetSlotByHash(hash common.Hash) *SlotContext {
	vc.lock.RLock()
	defer vc.lock.RUnlock()

	return vc.findSlot(hash)
}

func (vc *VerifyContext) prepareSlot(bh *types.BlockHeader, blog *bizLog) (*SlotContext, error) {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	if sc := vc.findSlot(bh.Hash); sc != nil {
		blog.log("prepareSlot find exist, status %v", sc.GetSlotStatus())
		return sc, nil
	} else {
		if vc.hasSignedBiggerQN(bh.TotalQN) {
			return nil, fmt.Errorf("hasSignedBiggerQN")
		}

		sc = createSlotContext(bh, vc.blockCtx.threshold())
		if len(vc.slots) >= model.Param.MaxSlotSize {
			minQN := uint64(10000)
			minQNHash := common.Hash{}
			for hash, slot := range vc.slots {
				if slot.BH.TotalQN < minQN {
					minQN = slot.BH.TotalQN
					minQNHash = hash
				}
			}
			delete(vc.slots, minQNHash)
			blog.log("prepreSlot replace slotHash %v, qn %v, commingQN %v, commingHash %v", minQNHash.ShortS(), minQN, bh.TotalQN, bh.Hash.ShortS())
		} else {
			blog.log("prepareSlot add slot")
		}
		//sc.init(bh)
		vc.slots[bh.Hash] = sc
		return sc, nil
	}
}

//收到某个验证人的验证完成消息（可能会比铸块完成消息先收到）
func (vc *VerifyContext) UserVerified(bh *types.BlockHeader, signData *model.SignData, pk groupsig.Pubkey, slog *slowLog) (ret CAST_BLOCK_MESSAGE_RESULT, err error) {
	blog := newBizLog("UserVerified")

	slog.addStage("prePareSlot")
	slot, err := vc.prepareSlot(bh, blog)
	if err != nil {
		blog.log("prepareSlot fail, err %v", err)
		return CBMR_ERROR_UNKNOWN, fmt.Errorf("prepareSlot fail, err %v", err)
	}
	slog.endStage()

	slog.addStage("initIfNeeded")
	slot.initIfNeeded()
	slog.endStage()

	//警惕并发
	if slot.IsFailed() {
		return CBMR_STATUS_FAIL, fmt.Errorf("slot fail")
	}
	if _, err2 := vc.baseCheck(bh, signData.GetID()); err2 != nil {
		err = err2
		return
	}
	isProposal := slot.castor.IsEqual(signData.GetID())

	if isProposal { //提案者
		slog.addStage("vCastorSign")
		b := signData.VerifySign(pk)
		slog.endStage()

		if !b {
			err = fmt.Errorf("verify castorsign fail, id %v, pk %v", signData.GetID().ShortS(), pk.ShortS())
			return
		}

	} else {
		slog.addStage("vMemSign")
		b := signData.VerifySign(pk)
		slog.endStage()

		if !b {
			err = fmt.Errorf("verify sign fail, id %v, pk %v, sig %v hash %v", signData.GetID().ShortS(), pk.GetHexString(), signData.DataSign.GetHexString(), signData.DataHash.Hex())
			return
		}
		sig := groupsig.DeserializeSign(bh.Random)
		if sig == nil || sig.IsNil() {
			err = fmt.Errorf("deserialize bh random fail, random %v", bh.Random)
			return
		}
		slog.addStage("vMemRandSign")
		b = groupsig.VerifySig(pk, vc.prevBH.Random, *sig)
		slog.endStage()

		if !b {
			err = fmt.Errorf("random sign verify fail")
			return
		}
	}
	//如果是提案者，因为提案者没有对块进行签名，则直接返回
	if isProposal {
		return CBMR_PIECE_NORMAL, nil
	}
	return slot.AcceptVerifyPiece(bh, signData)
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

func (vc *VerifyContext) Clear()  {
	vc.lock.Lock()
	defer vc.lock.Unlock()

    vc.slots = nil
    vc.broadcastSlot = nil
}

//判断该context是否可以删除，主要考虑是否发送了分红交易
func (vc *VerifyContext)shouldRemove(topHeight uint64) bool {
	//交易签名超时, 可以删除
	if vc.castRewardSignExpire() {
		return true
	}

	//自己广播的且已经发送过分红交易，可以删除
	if vc.broadcastSlot != nil && vc.broadcastSlot.IsRewardSent() {
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
	slots := make([]*SlotContext, 0)
	for _, slot := range vc.slots {
		slots = append(slots, slot)
	}
	return slots
}

func (vc *VerifyContext) checkBroadcast() (*SlotContext) {
	blog := newBizLog("checkBroadcast")
	if !vc.castSuccess() {
		//blog.log("not success st=%v", vc.consensusStatus)
		return nil
	}
	if time.Since(vc.createTime).Seconds() < float64(model.Param.MaxWaitBlockTime) {
		//blog.log("not the time, creatTime %v, now %v, since %v", vc.createTime, time.Now(), time.Since(vc.createTime).String())
		return nil
	}
	var maxQNSlot *SlotContext

	vc.lock.RLock()
	defer vc.lock.RUnlock()
	qns := make([]uint64, 0)

	for _, slot := range vc.slots {
		if !slot.IsRecovered() {
			continue
		}
		qns = append(qns, slot.BH.TotalQN)
		if maxQNSlot == nil {
			maxQNSlot = slot
		} else {
			if maxQNSlot.BH.TotalQN < slot.BH.TotalQN {
				maxQNSlot = slot
			} else if maxQNSlot.BH.TotalQN == slot.BH.TotalQN {
				v1 := base.VRF_proof2hash(maxQNSlot.BH.ProveValue.Bytes()).Big()
				v2 := base.VRF_proof2hash(slot.BH.ProveValue.Bytes()).Big()
				if v1.Cmp(v2) < 0 {
					maxQNSlot = slot
				}
			}
		}
	}
	if maxQNSlot != nil {
		blog.log("select max qn=%v, hash=%v, height=%v, hash=%v, all qn=%v", maxQNSlot.BH.TotalQN, maxQNSlot.BH.Hash.ShortS(), maxQNSlot.BH.Height, maxQNSlot.BH.Hash.ShortS(), qns)
	}
	return maxQNSlot
}
