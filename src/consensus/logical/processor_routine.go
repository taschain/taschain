package logical

import (
	"time"
	"consensus/model"
	"common"
	"consensus/groupsig"
	"logservice"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2018/11/28 下午2:03
**  Description: 
*/

func (p *Processor) getCastCheckRoutineName() string {
	return "self_cast_check_" + p.getPrefix()
}

func (p *Processor) getBroadcastRoutineName() string {
	return "broadcast_" + p.getPrefix()
}

func (p *Processor) getReleaseRoutineName() string {
	return "release_routine_" + p.getPrefix()
}


//检查是否当前组铸块
func (p *Processor) checkSelfCastRoutine() bool {
	if !p.Ready() {
		return false
	}

	blog := newBizLog("checkSelfCastRoutine")

	if p.MainChain.IsAdujsting() {
		blog.log("isAdjusting, return...")
		p.triggerCastCheck()
		return false
	}

	top := p.MainChain.QueryTopBlock()
	//if time.Since(top.CurTime).Seconds() < 1.0 {
	//	blog.log("too quick, slow down. preTime %v, now %v", top.CurTime.String(), time.Now().String())
	//	return false
	//}

	var (
		expireTime  time.Time
		castHeight  uint64
		deltaHeight uint64
	)
	d := time.Since(top.CurTime)
	if d < 0 {
		return false
	}

	deltaHeight = uint64(d.Seconds())/uint64(model.Param.MaxGroupCastTime) + 1
	if top.Height > 0 {
		castHeight = top.Height + deltaHeight
	} else {
		castHeight = uint64(1)
	}
	expireTime = GetCastExpireTime(top.CurTime, deltaHeight, castHeight)

	if !p.canProposalAt(castHeight) {
		return false
	}

	worker := p.GetVrfWorker()

	if worker != nil && worker.workingOn(top, castHeight) {
		//blog.log("already working on that block height=%v, status=%v", castHeight, worker.getStatus())
		return false
	} else {
		blog.log("topHeight=%v, topHash=%v, topCurTime=%v, castHeight=%v, expireTime=%v", top.Height, top.Hash.ShortS(), top.CurTime, castHeight, expireTime)
		worker = NewVRFWorker(p.GetSelfMinerDO(), top, castHeight, expireTime)
		p.setVrfWorker(worker)
		p.blockProposal()
	}
	return true
}

func (p *Processor) broadcastRoutine() bool {
	p.blockContexts.forEachReservedVctx(func(vctx *VerifyContext) bool {
		p.tryBroadcastBlock(vctx)
		return true
	})
	return true
}

func (p *Processor) releaseRoutine() bool {
	topHeight := p.MainChain.QueryTopBlock().Height
	if topHeight <= model.Param.CreateGroupInterval {
		return true
	}

	//删除verifyContext
	p.cleanVerifyContext(topHeight)

	//在当前高度解散的组不应立即从缓存删除，延缓一个建组周期删除。保证该组解散前夕建的块有效
	ids := p.globalGroups.DismissGroups(topHeight - model.Param.CreateGroupInterval)
	blog := newBizLog("releaseRoutine")

	if len(ids) > 0 {
		blog.log("clean group %v\n", len(ids))
		p.globalGroups.RemoveGroups(ids)
		p.blockContexts.removeBlockContexts(ids)
		p.belongGroups.leaveGroups(ids)
		for _, gid := range ids {
			blog.debug("DissolveGroupNet staticGroup gid %v ", gid.ShortS())
			p.NetServer.ReleaseGroupNet(gid.GetHexString())
		}
	}


	//释放超时未建成组的组网络和相应的dummy组
	p.joiningGroups.forEach(func(gc *GroupContext) bool {
		if gc.gInfo == nil || gc.is == GisGroupInitDone {
			return true
		}
		gis := &gc.gInfo.GI
		gHash := gis.GetHash()
		if gis.ReadyTimeout(topHeight) {
			blog.debug("DissolveGroupNet dummyGroup from joiningGroups gHash %v", gHash.ShortS())
			p.NetServer.ReleaseGroupNet(gHash.Hex())
			p.joiningGroups.RemoveGroup(gHash)

			initedGroup := p.globalGroups.GetInitedGroup(gHash)
			omgied := "nil"
			if initedGroup != nil {
				omgied = fmt.Sprintf("OMGIED:%v(%v)", initedGroup.receiveSize(), initedGroup.threshold)
			}

			waitPieceIds := make([]string, 0)
			for _, mem := range gc.gInfo.Mems {
				if !gc.node.hasPiece(mem) {
					waitPieceIds = append(waitPieceIds, mem.ShortS())
					if len(waitPieceIds) >= 5 {
						break
					}
				}
			}
			//发送日志
			le := &logservice.LogEntry{
				LogType: logservice.LogTypeInitGroupRevPieceTimeout,
				Height: p.GroupChain.Count(),
				Hash: gHash.Hex(),
				Proposer: "00",
				Ext: fmt.Sprintf("MemCnt:%v,Pieces:%v,wait:%v,%v", gc.gInfo.MemberSize(),gc.node.groupInitPool.GetSize(),waitPieceIds,omgied),
			}
			if logservice.Instance.IsFirstNInternalNodesInGroup(gc.gInfo.Mems, 50) {
				logservice.Instance.AddLog(le)
			}
		}
		return true
	})
	gctx := p.groupManager.getContext()
	if gctx != nil && gctx.readyTimeout(topHeight) {
		groupLogger.Infof("releaseRoutine:info=%v, elapsed %v. ready timeout.", gctx.logString(), time.Since(gctx.createTime))

		if gctx.isKing() {
			gHash := "0000"
			if gctx != nil && gctx.gInfo != nil {
				gHash = gctx.gInfo.GroupHash().Hex()
			}
			//发送日志
			le := &logservice.LogEntry{
				LogType: logservice.LogTypeCreateGroupSignTimeout,
				Height: p.GroupChain.Count(),
				Hash: gHash,
				Proposer: p.GetMinerID().GetHexString(),
				Ext: fmt.Sprintf("%v", gctx.gSignGenerator.Brief()),
			}
			if logservice.Instance.IsFirstNInternalNodesInGroup(gctx.kings, 50) {
				logservice.Instance.AddLog(le)
			}
		}
		p.groupManager.removeContext()
	}
	//p.groupManager.creatingGroups.forEach(func(cg *CreatingGroupContext) bool {
	//	gis := &cg.gInfo.GI
	//	gHash := gis.GetHash()
	//	if gis.ReadyTimeout(topHeight) {
	//		blog.debug("DissolveGroupNet dummyGroup from creatingGroups gHash %v", gHash.ShortS())
	//		p.NetServer.ReleaseGroupNet(gHash.Hex())
	//		p.groupManager.creatingGroups.removeGroup(gHash)
	//	}
	//	return true
	//})
	p.globalGroups.generator.forEach(func(ig *InitedGroup) bool {
		hash := ig.gInfo.GroupHash()
		if ig.gInfo.GI.ReadyTimeout(topHeight) {
			blog.debug("remove InitedGroup, gHash %v", hash.ShortS())
			p.NetServer.ReleaseGroupNet(hash.Hex())
			p.globalGroups.removeInitedGroup(hash)
		}
		return true
	})

	//释放futureVerifyMsg
	p.futureVerifyMsgs.forEach(func(key common.Hash, arr []interface{}) bool {
		for _, msg := range arr {
			b := msg.(*model.ConsensusCastMessage)
			if b.BH.Height+200 < topHeight {
				blog.debug("remove future verify msg, hash=%v", key.String())
				p.removeFutureVerifyMsgs(key)
				break
			}
		}
		return true
	})
	//释放futureRewardMsg
	p.futureRewardReqs.forEach(func(key common.Hash, arr []interface{}) bool {
		for _, msg := range arr {
			b := msg.(*model.CastRewardTransSignReqMessage)
			if time.Now().After(b.ReceiveTime.Add(400*time.Second)) {//400s不能处理的，都删除
				p.futureRewardReqs.remove(key)
				blog.debug("remove future reward msg, hash=%v", key.String())
				break
			}
		}
		return true
	})

	//清理超时的签名公钥请求
	cleanSignPkReqRecord()

	for _, h := range p.verifyMsgCaches.Keys() {
		hash := h.(common.Hash)
		cache := p.getVerifyMsgCache(hash)
		if cache != nil && cache.expired() {
			blog.debug("remove verify cache msg, hash=%v", hash.ShortS())
			p.removeVerifyMsgCache(hash)
		}
	}

	return true
}

func (p *Processor) getUpdateGlobalGroupsRoutineName() string {
    return "update_global_groups_routine_" + p.getPrefix()
}

func (p *Processor) updateGlobalGroups() bool {
    top := p.MainChain.Height()
    chainGroups := p.globalGroups.getCastQualifiedGroupFromChains(top)
	for _, g := range chainGroups {
		gid := groupsig.DeserializeId(g.Id)
		if g, _ := p.globalGroups.getGroupFromCache(gid); g != nil {
			continue
		}
		sgi := NewSGIFromCoreGroup(g)
		stdLogger.Debugf("updateGlobalGroups:gid=%v, workHeight=%v, topHeight=%v", gid.ShortS(), g.Header.WorkHeight, top)
		p.acceptGroup(sgi)
	}
	return true
}