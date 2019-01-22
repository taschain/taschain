package logical

import (
	"time"
	"consensus/model"
	"common"
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
		gis := &gc.gInfo.GI
		gHash := gis.GetHash()
		if gis.ReadyTimeout(topHeight) {
			blog.debug("DissolveGroupNet dummyGroup from joiningGroups gHash %v", gHash.ShortS())
			p.NetServer.ReleaseGroupNet(gHash.Hex())
			p.joiningGroups.RemoveGroup(gHash)
		}
		return true
	})
	p.groupManager.creatingGroups.forEach(func(cg *CreatingGroup) bool {
		gis := &cg.gInfo.GI
		gHash := gis.GetHash()
		if gis.ReadyTimeout(topHeight) {
			blog.debug("DissolveGroupNet dummyGroup from creatingGroups gHash %v", gHash.ShortS())
			p.NetServer.ReleaseGroupNet(gHash.Hex())
			p.groupManager.creatingGroups.removeGroup(gHash)
		}
		return true
	})

	//释放futureVerifyMsg
	p.futureVerifyMsgs.forEach(func(key common.Hash, arr []interface{}) bool {
		for _, msg := range arr {
			b := msg.(*model.ConsensusBlockMessageBase)
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
	return true
}