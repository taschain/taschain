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
	"github.com/taschain/taschain/consensus/base"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/middleware/types"
)

/*
**  Creator: pxf
**  Date: 2018/6/12 下午6:12
**  Description:
 */

//后续如有全局定时器，从这个函数启动
func (p *Processor) Start() bool {
	p.Ticker.RegisterPeriodicRoutine(p.getCastCheckRoutineName(), p.checkSelfCastRoutine, 1)
	p.Ticker.RegisterPeriodicRoutine(p.getReleaseRoutineName(), p.releaseRoutine, 2)
	p.Ticker.RegisterPeriodicRoutine(p.getBroadcastRoutineName(), p.broadcastRoutine, 1)
	p.Ticker.StartTickerRoutine(p.getReleaseRoutineName(), false)
	p.Ticker.StartTickerRoutine(p.getBroadcastRoutineName(), false)

	p.Ticker.RegisterPeriodicRoutine(p.getUpdateGlobalGroupsRoutineName(), p.updateGlobalGroups, 60)
	p.Ticker.StartTickerRoutine(p.getUpdateGlobalGroupsRoutineName(), false)

	p.Ticker.RegisterPeriodicRoutine(p.getUpdateMonitorNodeInfoRoutine(), p.updateMonitorInfo, 3)
	p.Ticker.StartTickerRoutine(p.getUpdateMonitorNodeInfoRoutine(), false)

	p.triggerCastCheck()
	p.prepareMiner()
	p.ready = true
	return true
}

//预留接口
func (p *Processor) Stop() {
	return
}

func (p *Processor) prepareMiner() {

	topHeight := p.MainChain.QueryTopBlock().Height

	stdLogger.Infof("prepareMiner get groups from groupchain")
	iterator := p.GroupChain.NewIterator()
	groups := make([]*StaticGroupInfo, 0)
	for coreGroup := iterator.Current(); coreGroup != nil; coreGroup = iterator.MovePre() {
		stdLogger.Infof("get group from core, id=%+v", coreGroup.Header)
		if coreGroup.ID == nil || len(coreGroup.ID) == 0 {
			continue
		}
		needBreak := false
		sgi := NewSGIFromCoreGroup(coreGroup)
		if sgi.Dismissed(topHeight) && len(groups) > 100 {
			needBreak = true
			genesis := p.GroupChain.GetGroupByHeight(0)
			if genesis == nil {
				panic("get genesis group nil")
			}
			sgi = NewSGIFromCoreGroup(genesis)

		}
		groups = append(groups, sgi)
		stdLogger.Infof("load group=%v, beginHeight=%v, topHeight=%v\n", sgi.GroupID.ShortS(), sgi.getGroupHeader().WorkHeight, topHeight)
		if sgi.MemExist(p.GetMinerID()) {
			jg := p.belongGroups.getJoinedGroup(sgi.GroupID)
			if jg == nil {
				stdLogger.Infof("prepareMiner get join group fail, gid=%v\n", sgi.GroupID.ShortS())
			} else {
				p.joinGroup(jg)
			}
			if sgi.GInfo.GI.CreateHeight() == 0 {
				stdLogger.Infof("genesis member start...id %v", p.GetMinerID().GetHexString())
				p.genesisMember = true
			}
		}
		if needBreak {
			break
		}
	}
	for i := len(groups) - 1; i >= 0; i-- {
		p.acceptGroup(groups[i])
	}
	stdLogger.Infof("prepare finished")
}

func (p *Processor) Ready() bool {
	return p.ready
}

func (p *Processor) GetCastQualifiedGroups(height uint64) []*StaticGroupInfo {
	return p.globalGroups.GetCastQualifiedGroups(height)
}

func (p *Processor) Finalize() {
	if p.belongGroups != nil {
		p.belongGroups.close()
	}
}

func (p *Processor) getVrfWorker() *vrfWorker {
	if v := p.vrf.Load(); v != nil {
		return v.(*vrfWorker)
	}
	return nil
}

func (p *Processor) setVrfWorker(vrf *vrfWorker) {
	p.vrf.Store(vrf)
}

func (p *Processor) GetSelfMinerDO() *model.SelfMinerDO {
	md := p.minerReader.getProposeMiner(p.GetMinerID())
	if md != nil {
		p.mi.MinerDO = *md
	}
	return p.mi
}

func (p *Processor) canProposalAt(h uint64) bool {
	miner := p.minerReader.getProposeMiner(p.GetMinerID())
	if miner == nil {
		return false
	}
	return miner.CanCastAt(h)
}

func (p *Processor) GetJoinedWorkGroupNums() (work, avail int) {
	h := p.MainChain.QueryTopBlock().Height
	groups := p.globalGroups.GetAvailableGroups(h)
	for _, g := range groups {
		if !g.MemExist(p.GetMinerID()) {
			continue
		}
		if g.CastQualified(h) {
			work++
		}
		avail++
	}
	return
}

func (p *Processor) CalcBlockHeaderQN(bh *types.BlockHeader) uint64 {
	pi := base.VRFProve(bh.ProveValue)
	castor := groupsig.DeserializeId(bh.Castor)
	miner := p.minerReader.getProposeMiner(castor)
	if miner == nil {
		stdLogger.Infof("CalcBHQN getMiner nil id=%v, bh=%v", castor.ShortS(), bh.Hash.ShortS())
		return 0
	}
	pre := p.MainChain.QueryBlockHeaderByHash(bh.PreHash)
	if pre == nil {
		return 0
	}
	totalStake := p.minerReader.getTotalStake(pre.Height, false)
	_, qn := vrfSatisfy(pi, miner.Stake, totalStake)
	return qn
}

func (p *Processor) GetVrfThreshold(stake uint64) float64 {
	totalStake := p.minerReader.getTotalStake(p.MainChain.Height(), true)
	if totalStake == 0 {
		return 0
	}
	vs := vrfThreshold(stake, totalStake)
	f, _ := vs.Float64()
	return f
}

//func (p *Processor) BlockContextSummary() string {
//
//	type slotSummary struct {
//		Hash string `json:"hash"`
//		GSigSize int `json:"g_sig_size"`
//		RSigSize int `json:"r_sig_size"`
//		TxSigSize int `json:"tx_sig_size"`
//		LostTxSize int `json:"lost_tx_size"`
//		Status int32 `json:"status"`
//	}
//	type vctxSummary struct {
//		CastHeight uint64 `json:"cast_height"`
//		Status int32 `json:"status"`
//		Slots  []*slotSummary `json:"slots"`
//		NumSlots int `json:"num_slots"`
//		Expire time.Time `json:"expire"`
//		ShouldRemove bool `json:"should_remove"`
//	}
//	type bctxSummary struct {
//    	Gid string `json:"gid"`
//    	NumRvh int `json:"num_rvh"`
//    	NumVctx int `json:"num_vctx"`
//    	Vctxs []*vctxSummary `json:"vctxs"`
//	}
//	type contextSummary struct {
//		NumBctxs int `json:"num_bctxs"`
//		Bctxs []*bctxSummary `json:"bctxs"`
//		NumReserVctx int `json:"num_reser_vctx"`
//		ReservVctxs []*vctxSummary `json:"reserv_vctxs"`
//		NumFutureVerifyMsg int `json:"num_future_verify_msg"`
//		NumFutureRewardMsg int `json:"num_future_reward_msg"`
//		NumVerifyCache int `json:"num_verify_cache"`
//	}
//	bctxs := make([]*bctxSummary, 0)
//	p.blockContexts.forEachBlockContext(func(bc *BlockContext) bool {
//		vs := make([]*vctxSummary, 0)
//		for _, vctx := range bc.SafeGetVerifyContexts() {
//			ss := make([]*slotSummary, 0)
//			for _, slot := range vctx.GetSlots() {
//				s := &slotSummary{
//					Hash: slot.BH.Hash.String(),
//					GSigSize: slot.gSignGenerator.WitnessSize(),
//					RSigSize: slot.rSignGenerator.WitnessSize(),
//					Status: slot.GetSlotStatus(),
//				}
//				if slot.rewardGSignGen != nil {
//					s.TxSigSize = slot.rewardGSignGen.WitnessSize()
//				}
//				ss = append(ss, s)
//			}
//			v := &vctxSummary{
//				CastHeight: vctx.castHeight,
//				Status: vctx.consensusStatus,
//				NumSlots: len(vctx.slots),
//				Expire: vctx.expireTime.Local(),
//				ShouldRemove: vctx.castRewardSignExpire() || (vctx.successSlot != nil && vctx.successSlot.IsRewardSent()),
//				Slots:ss,
//			}
//			vs = append(vs, v)
//		}
//		b := &bctxSummary{
//			Gid: bc.MinerID.Gid.GetHexString(),
//			NumRvh: len(bc.recentCasted),
//			NumVctx: len(vs),
//			Vctxs: vs,
//		}
//		bctxs = append(bctxs, b)
//		return true
//	})
//	reservVctxs := make([]*vctxSummary, 0)
//	p.blockContexts.forEachReservedVctx(func(vctx *VerifyContext) bool {
//		v := &vctxSummary{
//			CastHeight: vctx.castHeight,
//			Status: vctx.consensusStatus,
//			NumSlots: len(vctx.slots),
//			Expire: vctx.expireTime.Local(),
//			ShouldRemove: vctx.castRewardSignExpire() || (vctx.successSlot != nil && vctx.successSlot.IsRewardSent()),
//		}
//		reservVctxs = append(reservVctxs, v)
//		return true
//	})
//	cs := &contextSummary{
//		Bctxs: bctxs,
//		ReservVctxs: reservVctxs,
//		NumBctxs:len(bctxs),
//		NumReserVctx: len(reservVctxs),
//		NumFutureVerifyMsg: p.futureVerifyMsgs.size(),
//		NumFutureRewardMsg: p.futureRewardReqs.size(),
//		NumVerifyCache: p.verifyMsgCaches.Len(),
//	}
//	b, _ := json.MarshalIndent(cs, "", "\t")
//	fmt.Printf("%v\n", string(b))
//	fmt.Println("============================================================")
//	return string(b)
//}

func (p *Processor) GetJoinGroupInfo(gid string) *JoinedGroup {
	var id groupsig.ID
	id.SetHexString(gid)
	jg := p.belongGroups.getJoinedGroup(id)
	return jg
}

func (p *Processor) GetAllMinerDOs() []*model.MinerDO {
	h := p.MainChain.Height()
	dos := make([]*model.MinerDO, 0)
	miners := p.minerReader.getAllMinerDOByType(types.MinerTypeHeavy, h)
	dos = append(dos, miners...)

	miners = p.minerReader.getAllMinerDOByType(types.MinerTypeLight, h)
	dos = append(dos, miners...)
	return dos
}

func (p *Processor) GetCastQualifiedGroupsFromChain(height uint64) []*types.Group {
	return p.globalGroups.getCastQualifiedGroupFromChains(height)
}

func (p *Processor) CheckProveRoot(bh *types.BlockHeader) (bool, error) {
	//exist, ok, err := p.proveChecker.getPRootResult(bh.Hash)
	//if exist {
	//	return ok, err
	//}
	//slog := taslog.NewSlowLog("checkProveRoot-" + bh.Hash.ShortS(), 0.6)
	//defer func() {
	//	slog.Log("hash=%v, height=%v", bh.Hash.String(), bh.Height)
	//}()
	//slog.AddStage("queryBlockHeader")
	//preBH := p.MainChain.QueryBlockHeaderByHash(bh.PreHash)
	//slog.EndStage()
	//if preBH == nil {
	//	return false, errors.New(fmt.Sprintf("preBlock is nil,hash %v", bh.PreHash.ShortS()))
	//}
	//gid := groupsig.DeserializeId(bh.GroupID)
	//
	//slog.AddStage("getGroup")
	//group := p.GetGroup(gid)
	//slog.EndStage()
	//if !group.GroupID.IsValid() {
	//	return false, errors.New(fmt.Sprintf("group is invalid, gid %v", gid))
	//}

	////这个还是很耗时
	//slog.AddStage("genProveHash")
	//if _, root := p.proveChecker.genProveHashs(bh.Height, preBH.Random, group.GetMembers()); root == bh.ProveRoot {
	//	slog.EndStage()
	//	p.proveChecker.addPRootResult(bh.Hash, true, nil)
	//	return true, nil
	//} else {
	//	//TODO: 2019-04-08:bug 导致部分分红交易的source存进了db，全量账本校验失败，删库重启后再放开
	//	//panic(fmt.Errorf("check prove fail, hash=%v, height=%v", bh.Hash.String(), bh.Height))
	//	//return false, errors.New(fmt.Sprintf("proveRoot expect %v, receive %v", bh.ProveRoot.String(), root.String()))
	//}
	return true, nil
}

func (p *Processor) DebugPrintCheckProves(preBH *types.BlockHeader, height uint64, gid groupsig.ID) []string {

	group := p.GetGroup(gid)
	ss := make([]string, 0)
	for _, id := range group.GetMembers() {
		h := p.proveChecker.sampleBlockHeight(height, preBH.Random, id)
		bs := p.MainChain.QueryBlockBytesFloor(h)
		block := p.MainChain.QueryBlockFloor(h)
		hash := p.proveChecker.genVerifyHash(bs, id)

		var s string
		if block == nil {
			s = fmt.Sprintf("id %v, height %v, bytes %v, prove hash %v block nil", id.GetHexString(), h, bs, hash.Hex())
			stdLogger.Debugf("id %v, height %v, bytes %v, prove hash %v block nil", id.GetHexString(), h, bs, hash.Hex())
		} else {
			s = fmt.Sprintf("id %v, height %v, bytes %v, prove hash %v blockheader %+v, body %+v", id.GetHexString(), h, bs, hash.Hex(), block.Header, block.Transactions)
			stdLogger.Debugf("id %v, height %v, bytes %v, prove hash %v blockheader %+v, body %+v", id.GetHexString(), h, bs, hash.Hex(), block.Header, block.Transactions)
		}
		ss = append(ss, s)
	}
	return ss
}
