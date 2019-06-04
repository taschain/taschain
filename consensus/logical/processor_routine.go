package logical

import (
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/middleware/time"
	"github.com/taschain/taschain/monitor"
	time2 "time"
)

func (p *Processor) getCastCheckRoutineName() string {
	return "self_cast_check_" + p.getPrefix()
}

func (p *Processor) getBroadcastRoutineName() string {
	return "broadcast_" + p.getPrefix()
}

func (p *Processor) getReleaseRoutineName() string {
	return "release_routine_" + p.getPrefix()
}

// checkSelfCastRoutine check if the current group cast block
func (p *Processor) checkSelfCastRoutine() bool {
	if !p.Ready() {
		return false
	}

	blog := newBizLog("checkSelfCastRoutine")

	if p.MainChain.IsAdjusting() {
		blog.log("isAdjusting, return...")
		p.triggerCastCheck()
		return false
	}

	top := p.MainChain.QueryTopBlock()

	var (
		expireTime  time.TimeStamp
		castHeight  uint64
		deltaHeight uint64
	)
	d := p.ts.Since(top.CurTime)
	if d < 0 {
		return false
	}

	deltaHeight = uint64(d)/uint64(model.Param.MaxGroupCastTime) + 1
	if top.Height > 0 {
		castHeight = top.Height + deltaHeight
	} else {
		castHeight = uint64(1)
	}
	expireTime = GetCastExpireTime(top.CurTime, deltaHeight, castHeight)

	if !p.canProposalAt(castHeight) {
		return false
	}

	worker := p.getVrfWorker()

	if worker != nil && worker.workingOn(top, castHeight) {
		return false
	}
	blog.log("topHeight=%v, topHash=%v, topCurTime=%v, castHeight=%v, expireTime=%v", top.Height, top.Hash.ShortS(), top.CurTime, castHeight, expireTime)
	worker = newVRFWorker(p.GetSelfMinerDO(), top, castHeight, expireTime, p.ts)
	p.setVrfWorker(worker)
	p.blockProposal()
	return true
}

func (p *Processor) broadcastRoutine() bool {
	p.blockContexts.forEachReservedVctx(func(vctx *VerifyContext) bool {
		p.tryNotify(vctx)
		return true
	})
	return true
}

func (p *Processor) releaseRoutine() bool {
	topHeight := p.MainChain.QueryTopBlock().Height
	if topHeight <= model.Param.CreateGroupInterval {
		return true
	}
	// Groups that are currently highly disbanded should not be deleted from
	// the cache immediately, delaying the deletion of a build cycle. Ensure that
	// the block built on the eve of the dissolution of the group is valid
	groups := p.globalGroups.DismissGroups(topHeight - model.Param.CreateGroupInterval)
	ids := make([]groupsig.ID, 0)
	for _, g := range groups {
		ids = append(ids, g.GroupID)
	}

	p.blockContexts.cleanVerifyContext(topHeight)

	blog := newBizLog("releaseRoutine")

	if len(ids) > 0 {
		blog.log("clean group %v\n", len(ids))
		p.globalGroups.RemoveGroups(ids)
		p.belongGroups.leaveGroups(ids)
		for _, g := range groups {
			gid := g.GroupID
			blog.debug("DissolveGroupNet staticGroup gid %v ", gid.ShortS())
			p.NetServer.ReleaseGroupNet(gid.GetHexString())
			p.joiningGroups.RemoveGroup(g.GInfo.GroupHash())
		}
	}

	// Release the group network of the uncompleted group and the corresponding dummy group
	p.joiningGroups.forEach(func(gc *GroupContext) bool {
		if gc.gInfo == nil || gc.is == GisGroupInitDone {
			return true
		}
		gis := &gc.gInfo.GI
		gHash := gis.GetHash()
		if gis.ReadyTimeout(topHeight) {
			blog.debug("DissolveGroupNet dummyGroup from joiningGroups gHash %v", gHash.ShortS())
			p.NetServer.ReleaseGroupNet(gHash.Hex())

			initedGroup := p.globalGroups.GetInitedGroup(gHash)
			omgied := "nil"
			if initedGroup != nil {
				omgied = fmt.Sprintf("OMGIED:%v(%v)", initedGroup.receiveSize(), initedGroup.threshold)
			}

			waitPieceIds := make([]string, 0)
			waitIds := make([]groupsig.ID, 0)
			for _, mem := range gc.gInfo.Mems {
				if !gc.node.hasPiece(mem) {
					waitPieceIds = append(waitPieceIds, mem.ShortS())
					waitIds = append(waitIds, mem)
				}
			}
			// Send log
			le := &monitor.LogEntry{
				LogType:  monitor.LogTypeInitGroupRevPieceTimeout,
				Height:   p.GroupChain.Height(),
				Hash:     gHash.Hex(),
				Proposer: "00",
				Ext:      fmt.Sprintf("MemCnt:%v,Pieces:%v,wait:%v,%v", gc.gInfo.MemberSize(), gc.node.groupInitPool.GetSize(), waitPieceIds, omgied),
			}
			if !gc.sendLog && monitor.Instance.IsFirstNInternalNodesInGroup(gc.gInfo.Mems, 50) {
				monitor.Instance.AddLog(le)
				gc.sendLog = true
			}

			msg := &model.ReqSharePieceMessage{
				GHash: gc.gInfo.GroupHash(),
			}
			stdLogger.Infof("reqSharePieceRoutine:req size %v, ghash=%v", len(waitIds), gc.gInfo.GroupHash().ShortS())
			if msg.GenSign(p.getDefaultSeckeyInfo(), msg) {
				for _, receiver := range waitIds {
					stdLogger.Infof("reqSharePieceRoutine:req share piece msg from %v, ghash=%v", receiver, gc.gInfo.GroupHash().ShortS())
					p.NetServer.ReqSharePiece(msg, receiver)
				}
			} else {
				ski := p.getDefaultSeckeyInfo()
				stdLogger.Infof("gen req sharepiece sign fail, ski=%v %v", ski.ID.ShortS(), ski.SK.ShortS())
			}

		}
		return true
	})
	gctx := p.groupManager.getContext()
	if gctx != nil && gctx.readyTimeout(topHeight) {
		groupLogger.Infof("releaseRoutine:info=%v, elapsed %v. ready timeout.", gctx.logString(), time2.Since(gctx.createTime))

		if gctx.isKing() {
			gHash := "0000"
			if gctx != nil && gctx.gInfo != nil {
				gHash = gctx.gInfo.GroupHash().Hex()
			}
			// Send log
			le := &monitor.LogEntry{
				LogType:  monitor.LogTypeCreateGroupSignTimeout,
				Height:   p.GroupChain.Height(),
				Hash:     gHash,
				Proposer: p.GetMinerID().GetHexString(),
				Ext:      fmt.Sprintf("%v", gctx.gSignGenerator.Brief()),
			}
			if monitor.Instance.IsFirstNInternalNodesInGroup(gctx.kings, 50) {
				monitor.Instance.AddLog(le)
			}
		}
		p.groupManager.removeContext()
	}

	p.globalGroups.generator.forEach(func(ig *InitedGroup) bool {
		hash := ig.gInfo.GroupHash()
		if ig.gInfo.GI.ReadyTimeout(topHeight) {
			blog.debug("remove InitedGroup, gHash %v", hash.ShortS())
			p.NetServer.ReleaseGroupNet(hash.Hex())
			p.globalGroups.removeInitedGroup(hash)
		}
		return true
	})

	// Release futureVerifyMsg
	p.futureVerifyMsgs.forEach(func(key common.Hash, arr []interface{}) bool {
		for _, msg := range arr {
			b := msg.(*model.ConsensusCastMessage)
			if b.BH.Height+200 < topHeight {
				blog.debug("remove future verify msg, hash=%v", key.Hex())
				p.removeFutureVerifyMsgs(key)
				break
			}
		}
		return true
	})
	// Release futureRewardMsg
	p.futureRewardReqs.forEach(func(key common.Hash, arr []interface{}) bool {
		for _, msg := range arr {
			b := msg.(*model.CastRewardTransSignReqMessage)
			if time2.Now().After(b.ReceiveTime.Add(400 * time2.Second)) { //400s不能处理的，都删除
				p.futureRewardReqs.remove(key)
				blog.debug("remove future reward msg, hash=%v", key.Hex())
				break
			}
		}
		return true
	})

	// Clean up the timeout signature public key request
	cleanSignPkReqRecord()

	for _, h := range p.blockContexts.verifyMsgCaches.Keys() {
		hash := h.(common.Hash)
		cache := p.blockContexts.getVerifyMsgCache(hash)
		if cache != nil && cache.expired() {
			blog.debug("remove verify cache msg, hash=%v", hash.ShortS())
			p.blockContexts.removeVerifyMsgCache(hash)
		}
	}

	return true
}

func (p *Processor) getUpdateGlobalGroupsRoutineName() string {
	return "update_global_groups_routine_" + p.getPrefix()
}

func (p *Processor) updateGlobalGroups() bool {
	top := p.MainChain.Height()
	iter := p.GroupChain.NewIterator()
	for g := iter.Current(); g != nil && !IsGroupDissmisedAt(g.Header, top); g = iter.MovePre() {
		gid := groupsig.DeserializeID(g.ID)
		if g, _ := p.globalGroups.getGroupFromCache(gid); g != nil {
			continue
		}
		sgi := NewSGIFromCoreGroup(g)
		stdLogger.Debugf("updateGlobalGroups:gid=%v, workHeight=%v, topHeight=%v", gid.ShortS(), g.Header.WorkHeight, top)
		p.acceptGroup(sgi)
	}
	return true
}

func (p *Processor) getUpdateMonitorNodeInfoRoutine() string {
	return "update_monitor_node_routine_" + p.getPrefix()
}

func (p *Processor) updateMonitorInfo() bool {
	if !monitor.Instance.MonitorEnable() {
		return false
	}
	top := p.MainChain.Height()

	ni := &monitor.NodeInfo{
		BlockHeight: top,
		GroupHeight: p.GroupChain.Height(),
		TxPoolCount: int(p.MainChain.GetTransactionPool().TxNum()),
	}
	proposer := p.minerReader.getProposeMiner(p.GetMinerID())
	if proposer != nil {
		ni.Type |= monitor.NtypeProposal
		ni.PStake = proposer.Stake
		ni.VrfThreshold = p.GetVrfThreshold(ni.PStake)
	}
	verifier := p.minerReader.getLightMiner(p.GetMinerID())
	if verifier != nil {
		ni.Type |= monitor.NtypeVerifier
	}

	monitor.Instance.UpdateNodeInfo(ni)
	return true
}
