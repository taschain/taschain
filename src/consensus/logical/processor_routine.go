package logical

import (
	"time"
	"consensus/model"
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

	worker := p.getVrfWorker()

	if worker != nil && worker.workingOn(top, castHeight) {
		//blog.log("already working on that block height=%v, status=%v", castHeight, worker.getStatus())
		return false
	} else {
		blog.log("topHeight=%v, topHash=%v, topCurTime=%v, castHeight=%v, expireTime=%v", top.Height, top.Hash.ShortS(), top.CurTime, castHeight, expireTime)
		worker = newVRFWorker(p.GetSelfMinerDO(), top, castHeight, expireTime)
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
	//在当前高度解散的组不应立即从缓存删除，延缓一个建组周期删除。保证该组解散前夕建的块有效
	ids := p.globalGroups.DismissGroups(topHeight - model.Param.CreateGroupInterval)
	if len(ids) == 0 {
		return true
	}

	// 从内存中删除组创建高度对应的组信息，防止占用内存过大
	for k := range p.CreateHeightGroups {
		if k < topHeight - model.Param.CreateGroupInterval {
			delete(p.CreateHeightGroups, k)
		}
	}

	blog := newBizLog("releaseRoutine")
	blog.log("clean group %v\n", len(ids))
	p.globalGroups.RemoveGroups(ids)
	p.blockContexts.removeBlockContexts(ids)
	p.belongGroups.leaveGroups(ids)
	for _, gid := range ids {
		blog.debug("DissolveGroupNet staticGroup gid ", gid.ShortS())
		p.NetServer.ReleaseGroupNet(gid.GetHexString())
	}

	//释放超时未建成组的组网络和相应的dummy组
	p.joiningGroups.forEach(func(gc *GroupContext) bool {
		gis := &gc.gInfo.GI
		gHash := gis.GetHash()
		if gis.ReadyTimeout(topHeight) {
			blog.debug("DissolveGroupNet dummyGroup from joiningGroups gHash ", gHash.ShortS())
			p.NetServer.ReleaseGroupNet(gHash.Hex())
			p.joiningGroups.RemoveGroup(gHash)
		}
		return true
	})
	p.groupManager.creatingGroups.forEach(func(cg *CreatingGroup) bool {
		gis := &cg.gInfo.GI
		gHash := gis.GetHash()
		if gis.ReadyTimeout(topHeight) {
			blog.debug("DissolveGroupNet dummyGroup from creatingGroups gHash ", gHash.ShortS())
			p.NetServer.ReleaseGroupNet(gHash.Hex())
			p.groupManager.creatingGroups.removeGroup(gHash)
		}
		return true
	})
	return true
}