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
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/middleware/time"
	"github.com/taschain/taschain/middleware/types"
	"gopkg.in/fatih/set.v0"
	"sync"
	"sync/atomic"
)

/*
**  Creator: pxf
**  Date: 2018/5/29 上午10:19
**  Description:
 */

const (
	svWorking  = iota //正在铸块,等待分片
	svSuccess         //块签名已聚合
	svNotified        //块已通知提案者
	svTimeout         //组铸块超时
)

const (
	pieceNormal    = 1 //收到一个分片，接收正常
	pieceThreshold = 2 //收到一个分片且达到阈值，组签名成功
	pieceFail      = -1
)

type VerifyContext struct {
	prevBH           *types.BlockHeader
	castHeight       uint64
	signedMaxWeight  atomic.Value //**types.BlockWeight
	signedBlockHashs set.Interface
	expireTime       time.TimeStamp //铸块超时时间
	createTime       time.TimeStamp
	consensusStatus  int32 //铸块状态
	slots            map[common.Hash]*SlotContext
	proposers        map[string]common.Hash
	successSlot      *SlotContext
	//castedQNs []int64 //自己铸过的qn
	group     *StaticGroupInfo
	signedNum int32 //签名数量
	verifyNum int32 //验证签名数量
	aggrNum   int32 //聚合出组签名数量
	lock      sync.RWMutex
	ts        time.TimeService
}

func newVerifyContext(group *StaticGroupInfo, castHeight uint64, expire time.TimeStamp, preBH *types.BlockHeader) *VerifyContext {
	ctx := &VerifyContext{
		prevBH:          preBH,
		castHeight:      castHeight,
		group:           group,
		expireTime:      expire,
		consensusStatus: svWorking,
		//signedMaxWeight: 	*types.NewBlockWeight(),
		slots:            make(map[common.Hash]*SlotContext),
		ts:               time.TSInstance,
		createTime:       time.TSInstance.Now(),
		proposers:        make(map[string]common.Hash),
		signedBlockHashs: set.New(set.ThreadSafe),
		//castedQNs:       make([]int64, 0),
	}
	return ctx
}

func (vc *VerifyContext) isWorking() bool {
	status := atomic.LoadInt32(&vc.consensusStatus)
	return status != svTimeout
}

func (vc *VerifyContext) castSuccess() bool {
	return atomic.LoadInt32(&vc.consensusStatus) == svSuccess
}

//func (vc *VerifyContext) broadCasted() bool {
//	return atomic.LoadInt32(&vc.consensusStatus) == svSuccess
//}

func (vc *VerifyContext) isNotified() bool {
	return atomic.LoadInt32(&vc.consensusStatus) == svNotified
}

func (vc *VerifyContext) markTimeout() {
	if !vc.castSuccess() {
		atomic.StoreInt32(&vc.consensusStatus, svTimeout)
	}
}

func (vc *VerifyContext) markCastSuccess() {
	atomic.StoreInt32(&vc.consensusStatus, svSuccess)
}

func (vc *VerifyContext) markNotified() {
	atomic.StoreInt32(&vc.consensusStatus, svNotified)
}

//func (vc *VerifyContext) markBroadcast() bool {
//	return atomic.CompareAndSwapInt32(&vc.consensusStatus, svBlocked, svSuccess)
//}

//铸块是否过期
func (vc *VerifyContext) castExpire() bool {
	return vc.ts.NowAfter(vc.expireTime)
}

//分红交易签名是否过期
func (vc *VerifyContext) castRewardSignExpire() bool {
	return vc.ts.NowAfter(vc.expireTime.Add(int64(30 * model.Param.MaxGroupCastTime)))
}

func (vc *VerifyContext) findSlot(hash common.Hash) *SlotContext {
	if sc, ok := vc.slots[hash]; ok {
		return sc
	}
	return nil
}

func (vc *VerifyContext) getSignedMaxWeight() *types.BlockWeight {
	v := vc.signedMaxWeight.Load()
	if v == nil {
		return nil
	}
	return v.(*types.BlockWeight)
}

func (vc *VerifyContext) hasSignedMoreWeightThan(bh *types.BlockHeader) bool {
	bw := vc.getSignedMaxWeight()
	if bw == nil {
		return false
	}
	bw2 := types.NewBlockWeight(bh)
	return bw.MoreWeight(bw2)
}

func (vc *VerifyContext) updateSignedMaxWeightBlock(bh *types.BlockHeader) bool {
	bw := vc.getSignedMaxWeight()
	bw2 := types.NewBlockWeight(bh)
	if bw != nil && bw.MoreWeight(bw2) {
		return false
	}
	vc.signedMaxWeight.Store(bw2)
	return true
}

func (vc *VerifyContext) baseCheck(bh *types.BlockHeader, sender groupsig.ID) (err error) {
	if bh.Elapsed <= 0 {
		err = fmt.Errorf("elapsed error %v", bh.Elapsed)
		return
	}
	if vc.ts.Since(bh.CurTime) < -1 {
		return fmt.Errorf("block too early: now %v, curtime %v", vc.ts.Now(), bh.CurTime)
	}
	begin := vc.expireTime.Add(-int64(model.Param.MaxGroupCastTime + 1))
	if bh.Height > 1 && !vc.ts.NowAfter(begin) {
		return fmt.Errorf("block too early: begin %v, now %v", begin, vc.ts.Now())
	}
	gid := groupsig.DeserializeId(bh.GroupID)
	if !vc.group.GroupID.IsEqual(gid) {
		return fmt.Errorf("groupId error:vc-%v, bh-%v", vc.group.GroupID.ShortS(), gid.ShortS())
	}

	//只签qn不小于已签出的最高块的块
	if vc.hasSignedMoreWeightThan(bh) {
		err = fmt.Errorf("已签过更高qn块%v,本块qn%v", vc.getSignedMaxWeight().String(), bh.TotalQN)
		return
	}

	if vc.castSuccess() {
		err = fmt.Errorf("已出块")
		return
	}
	if vc.castExpire() {
		vc.markTimeout()
		err = fmt.Errorf("已超时" + vc.expireTime.String())
		return
	}
	slot := vc.GetSlotByHash(bh.Hash)
	if slot != nil {
		if slot.GetSlotStatus() >= slRecoverd {
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

func (vc *VerifyContext) PrepareSlot(bh *types.BlockHeader) (*SlotContext, error) {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	if vc.slots == nil {
		return nil, fmt.Errorf("slots is nil")
	}

	if vc.hasSignedMoreWeightThan(bh) {
		return nil, fmt.Errorf("hasSignedMoreWeightThan")
	}
	sc := createSlotContext(bh, model.Param.GetGroupK(vc.group.GetMemberCount()))
	if v, ok := vc.proposers[sc.castor.GetHexString()]; ok {
		if v != bh.Hash {
			return nil, fmt.Errorf("too many proposals: castor %v", sc.castor.GetHexString())
		}
	} else {
		vc.proposers[sc.castor.GetHexString()] = bh.Hash
	}
	if len(vc.slots) >= model.Param.MaxSlotSize {
		var (
			minWeight     *types.BlockWeight
			minWeightHash common.Hash
		)
		for hash, slot := range vc.slots {
			bw := types.NewBlockWeight(slot.BH)
			if minWeight == nil || minWeight.MoreWeight(bw) {
				minWeight = bw
				minWeightHash = hash
			}
		}
		currBw := *types.NewBlockWeight(bh)
		if currBw.MoreWeight(minWeight) {
			delete(vc.slots, minWeightHash)
		} else {
			return nil, fmt.Errorf("comming block weight less than min block weight")
		}
	}
	//sc.init(bh)
	vc.slots[bh.Hash] = sc
	return sc, nil

}

//（网络接收）新到交易集通知
//返回不再缺失交易的QN槽列表
//func (vc *VerifyContext) AcceptTrans(slot *SlotContext, ths []common.Hash) int8 {
//
//	if !slot.IsValid() {
//		return TRANS_INVALID_SLOT
//	}
//	accept := slot.AcceptTrans(ths)
//	if !accept {
//		return TRANS_DENY
//	}
//	if slot.HasTransLost() {
//		return TRANS_ACCEPT_NOT_FULL
//	}
//	st := slot.GetSlotStatus()
//
//	if st == slRecoverd || st == slVerified {
//		return TRANS_ACCEPT_FULL_THRESHOLD
//	} else {
//		return TRANS_ACCEPT_FULL_PIECE
//	}
//}

func (vc *VerifyContext) Clear() {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	vc.slots = nil
	vc.successSlot = nil
}

//判断该context是否可以删除，主要考虑是否发送了分红交易
func (vc *VerifyContext) shouldRemove(topHeight uint64) bool {
	//交易签名超时, 可以删除
	if vc.castRewardSignExpire() {
		return true
	}

	//自己广播的且已经发送过分红交易，可以删除
	if vc.successSlot != nil && vc.successSlot.IsRewardSent() {
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

func (vc *VerifyContext) checkNotify() *SlotContext {
	blog := newBizLog("checkNotify")
	if !vc.castSuccess() || vc.isNotified() {
		return nil
	}
	if vc.ts.Since(vc.createTime) < int64(model.Param.MaxWaitBlockTime) {
		return nil
	}
	var (
		maxBwSlot *SlotContext
		maxBw     *types.BlockWeight
	)

	vc.lock.RLock()
	defer vc.lock.RUnlock()
	qns := make([]uint64, 0)

	for _, slot := range vc.slots {
		if !slot.IsRecovered() {
			continue
		}
		qns = append(qns, slot.BH.TotalQN)
		bw := types.NewBlockWeight(slot.BH)

		if maxBw == nil || bw.MoreWeight(maxBw) {
			maxBwSlot = slot
			maxBw = bw
		}
	}
	if maxBwSlot != nil {
		blog.log("select max qn=%v, hash=%v, height=%v, hash=%v, all qn=%v", maxBwSlot.BH.TotalQN, maxBwSlot.BH.Hash.ShortS(), maxBwSlot.BH.Height, maxBwSlot.BH.Hash.ShortS(), qns)
	}
	return maxBwSlot
}

func (vc *VerifyContext) increaseVerifyNum() {
	atomic.AddInt32(&vc.verifyNum, 1)
}

func (vc *VerifyContext) increaseAggrNum() {
	atomic.AddInt32(&vc.aggrNum, 1)
}

func (vc *VerifyContext) markSignedBlock(bh *types.BlockHeader) {
	vc.signedBlockHashs.Add(bh.Hash)
	atomic.AddInt32(&vc.signedNum, 1)
	vc.updateSignedMaxWeightBlock(bh)
}

func (vc *VerifyContext) blockSigned(hash common.Hash) bool {
	return vc.signedBlockHashs.Has(hash)
}
