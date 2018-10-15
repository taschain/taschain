package logical

import (
		"log"
	"strings"
	"common"
	"consensus/model"
	"consensus/groupsig"
)

/*
**  Creator: pxf
**  Date: 2018/6/12 下午6:12
**  Description: 
*/

func (p *Processor) genBelongGroupStoreFile() string {
	storeFile := consensusConfManager.GetString("joined_group_store", "")
	if strings.TrimSpace(storeFile) == "" {
		storeFile = "joined_group.config." + common.GlobalConf.GetString("instance", "index", "")
	}
	return storeFile
}

//后续如有全局定时器，从这个函数启动
func (p *Processor) Start() bool {
	p.Ticker.RegisterRoutine(p.getCastCheckRoutineName(), p.checkSelfCastRoutine, 2)
	p.Ticker.RegisterRoutine(p.getReleaseRoutineName(), p.releaseRoutine, 2)
	p.Ticker.StartTickerRoutine(p.getReleaseRoutineName(), false)
	p.triggerCastCheck()
	p.prepareMiner()
	p.ready = true
	return true
}

//预留接口
func (p *Processor) Stop() {
	return
}

func (p *Processor) prepareMiner()  {

	topHeight := p.MainChain.QueryTopBlock().Height

	storeFile := p.genBelongGroupStoreFile()

	belongs := NewBelongGroups(storeFile)
	if !belongs.load() {

	}

	log.Printf("prepareMiner get groups from groupchain, belongGroup len=%v\n",  belongs.groupSize())
	iterator := p.GroupChain.NewIterator()
	for coreGroup := iterator.Current(); coreGroup != nil; coreGroup = iterator.MovePre(){
		log.Printf("get group from core, id=%v", coreGroup.Id)
		if coreGroup.Id == nil || len(coreGroup.Id) == 0 {
			continue
		}
		needBreak := false
		sgi := NewSGIFromCoreGroup(coreGroup)
		if sgi.Dismissed(topHeight) {
			needBreak = true
			genesis := p.GroupChain.GetGroupByHeight(0)
			if coreGroup == nil {
				panic("get genesis group nil")
			}
			sgi = NewSGIFromCoreGroup(genesis)
		}
		log.Printf("load group=%v, beginHeight=%v, topHeight=%v\n", sgi.GroupID.ShortS(), sgi.BeginHeight, topHeight)
		if sgi.MemExist(p.GetMinerID()) {
			jg := belongs.getJoinedGroup(sgi.GroupID)
			if jg == nil {
				log.Printf("prepareMiner get join group fail, gid=%v\n", sgi.GroupID.ShortS())
			} else {
				p.joinGroup(jg, true)
			}
		}
		p.acceptGroup(sgi)
		if needBreak {
			break
		}
	}
	log.Printf("prepare finished")
}

func (p *Processor) Ready() bool {
    return p.ready
}


func (p *Processor) GetAvailableGroupsAt(height uint64) []*StaticGroupInfo {
    return p.globalGroups.GetAvailableGroups(height)
}


func (p *Processor) GetCastQualifiedGroups(height uint64) []*StaticGroupInfo {
	return p.globalGroups.GetCastQualifiedGroups(height)
}

func (p *Processor) Finalize() {
	if p.belongGroups != nil {
		p.belongGroups.commit()
	}
}

func (p *Processor) releaseRoutine() bool {
	topHeight := p.MainChain.QueryTopBlock().Height
	if topHeight <= model.Param.CreateGroupInterval {
		return true
	}
	//在当前高度解散的组不应立即从缓存删除，延缓一个建组周期删除。保证改组解散前夕建的块有效
	ids := p.globalGroups.DismissGroups(topHeight - model.Param.CreateGroupInterval)
	if len(ids) == 0 {
		return true
	}
	log.Printf("releaseRoutine: clean group %v\n", len(ids))
	p.globalGroups.RemoveGroups(ids)
	p.blockContexts.removeContexts(ids)
	p.belongGroups.leaveGroups(ids)
	for _, gid := range ids {
		log.Println("releaseRoutine DissolveGroupNet staticGroup gid ", gid.ShortS())
		p.NetServer.ReleaseGroupNet(gid)
	}

    //释放超时未建成组的组网络和相应的dummy组
	p.joiningGroups.forEach(func(gc *GroupContext) bool {
		if gc.gis.ReadyTimeout(topHeight) {
			log.Println("releaseRoutine DissolveGroupNet dummyGroup from joutils.GetngGroups gid ", gc.gis.DummyID.ShortS())
			p.NetServer.ReleaseGroupNet(gc.gis.DummyID)
			p.joiningGroups.RemoveGroup(gc.gis.DummyID)
		}
		return true
	})
	p.groupManager.creatingGroups.forEach(func(cg *CreatingGroup) bool {
		if cg.gis.ReadyTimeout(topHeight) {
			log.Println("releaseRoutine DissolveGroupNet dummyGroup from creatingGroups gid ", cg.gis.DummyID.ShortS())
			p.NetServer.ReleaseGroupNet(cg.gis.DummyID)
			p.groupManager.creatingGroups.removeGroup(cg.gis.DummyID)
		}
		return true
	})
	return true
}

func (p *Processor) getVrfWorker() *vrfWorker {
    if v := p.vrf.Load(); v != nil {
    	return v.(*vrfWorker)
	}
	return nil
}

func (p *Processor) setVrfWorker(vrf *vrfWorker)  {
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
   	return p.minerCanProposalAt(p.GetMinerID(), h)
}

func (p *Processor) minerCanProposalAt(id groupsig.ID, h uint64) bool {
	miner := p.minerReader.getProposeMiner(id)
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