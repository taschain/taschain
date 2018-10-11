package logical

import (
		"log"
	"strings"
	"common"
	"consensus/model"
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

func (p *Processor) prepareMiner()  {

	topHeight := p.MainChain.QueryTopBlock().Height

	storeFile := p.genBelongGroupStoreFile()

	belongs := NewBelongGroups(storeFile)
	if !belongs.load() {

	}

	log.Printf("prepareMiner get groups from groupchain, belongGroup len=%v\n",  belongs.groupSize())
	iterator := p.GroupChain.NewIterator()
        needBreak := false
	for coreGroup := iterator.Current(); coreGroup != nil; coreGroup = iterator.MovePre(){
		if coreGroup.Id == nil || len(coreGroup.Id) == 0 {
			continue
		}
		sgi := NewSGIFromCoreGroup(coreGroup)
		log.Printf("load group=%v, topHeight=%v\n", GetIDPrefix(sgi.GroupID), topHeight)
		if sgi.Dismissed(topHeight) {
                        group,_ := p.GroupChain.GetGroupsByHeight(0)
                        sgi = NewSGIFromCoreGroup(group[0])
                        needBreak = true
                        log.Printf("iterator break for Dismissed Group, Get GenesisGroup")
                        needBreak = true
		}
		if sgi.MemExist(p.GetMinerID()) {
			jg := belongs.getJoinedGroup(sgi.GroupID)
			if jg == nil {
				log.Printf("prepareMiner get join group fail, gid=%v\n", GetIDPrefix(sgi.GroupID))
			} else {
				p.joinGroup(jg, true)
			}
		}
		p.acceptGroup(sgi)
                if needBreak == true {
                    break
                }
	}
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
		log.Println("releaseRoutine DissolveGroupNet staticGroup gid ", GetIDPrefix(gid))
		p.NetServer.ReleaseGroupNet(gid)
	}

    //释放超时未建成组的组网络和相应的dummy组
	p.joiningGroups.forEach(func(gc *GroupContext) bool {
		if gc.gis.ReadyTimeout(topHeight) {
			log.Println("releaseRoutine DissolveGroupNet dummyGroup from joutils.GetngGroups gid ", GetIDPrefix(gc.gis.DummyID))
			p.NetServer.ReleaseGroupNet(gc.gis.DummyID)
			p.joiningGroups.RemoveGroup(gc.gis.DummyID)
		}
		return true
	})
	p.groupManager.creatingGroups.forEach(func(cg *CreatingGroup) bool {
		if cg.gis.ReadyTimeout(topHeight) {
			log.Println("releaseRoutine DissolveGroupNet dummyGroup from creatingGroups gid ", GetIDPrefix(cg.gis.DummyID))
			p.NetServer.ReleaseGroupNet(cg.gis.DummyID)
			p.groupManager.creatingGroups.removeGroup(cg.gis.DummyID)
		}
		return true
	})
	return true
}
