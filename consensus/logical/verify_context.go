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
	"sync"
	"sync/atomic"

	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/middleware/time"
	"github.com/taschain/taschain/middleware/types"
	"gopkg.in/fatih/set.v0"
)

const (
	// svWorking Casting blocks, waiting for shards
	svWorking = iota
	// svSuccess indicate block signature has been aggregated
	svSuccess
	// svNotified means block has notified the sponsor
	svNotified
	// svTimeout means Group cast block timeout
	svTimeout
)

const (
	// pieceNormal means received a shard and received normal
	pieceNormal = 1

	// pieceThreshold means received a shard and reached the threshold,
	// the group signature succeeded
	pieceThreshold = 2
	pieceFail      = -1
)

type VerifyContext struct {
	prevBH           *types.BlockHeader
	castHeight       uint64
	signedMaxWeight  atomic.Value // *types.BlockWeight
	signedBlockHashs set.Interface
	expireTime       time.TimeStamp // cast block timeout
	createTime       time.TimeStamp
	consensusStatus  int32 // cast state
	slots            map[common.Hash]*SlotContext
	proposers        map[string]common.Hash
	successSlot      *SlotContext
	group            *StaticGroupInfo
	signedNum        int32 // Numbers of signatures
	verifyNum        int32 // Numbers of Verify signatures
	aggrNum          int32 // Numbers of aggregate group signatures
	lock             sync.RWMutex
	ts               time.TimeService
}

func newVerifyContext(group *StaticGroupInfo, castHeight uint64, expire time.TimeStamp, preBH *types.BlockHeader) *VerifyContext {
	ctx := &VerifyContext{
		prevBH:           preBH,
		castHeight:       castHeight,
		group:            group,
		expireTime:       expire,
		consensusStatus:  svWorking,
		slots:            make(map[common.Hash]*SlotContext),
		ts:               time.TSInstance,
		createTime:       time.TSInstance.Now(),
		proposers:        make(map[string]common.Hash),
		signedBlockHashs: set.New(set.ThreadSafe),
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

// castExpire means whether the ingot has expired
func (vc *VerifyContext) castExpire() bool {
	return vc.ts.NowAfter(vc.expireTime)
}

// castRewardSignExpire means whether the reward transaction signature expires
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
	gid := groupsig.DeserializeID(bh.GroupID)
	if !vc.group.GroupID.IsEqual(gid) {
		return fmt.Errorf("groupId error:vc-%v, bh-%v", vc.group.GroupID.ShortS(), gid.ShortS())
	}

	// Only sign qn not less than the block of the highest block that has been signed
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
	vc.slots[bh.Hash] = sc
	return sc, nil

}

func (vc *VerifyContext) Clear() {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	vc.slots = nil
	vc.successSlot = nil
}

// shouldRemove determine whether the context can be deleted,
// mainly consider whether to send a dividend transaction
func (vc *VerifyContext) shouldRemove(topHeight uint64) bool {
	// Transaction signature timed out, can be deleted
	if vc.castRewardSignExpire() {
		return true
	}

	// Self-broadcast and already sent reward transaction, can be deleted
	if vc.successSlot != nil && vc.successSlot.IsRewardSent() {
		return true
	}

	// No block, but the height is already less than 200, can be deleted
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
