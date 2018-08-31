package logical

import (
		"log"
	"strings"
	"common"
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
	for coreGroup := iterator.Current(); iterator.Current() != nil;coreGroup = iterator.MovePre(){
		if coreGroup == nil {
			panic("buildGlobalGroups getGroupById failed!")
		}
		if coreGroup.Id == nil || len(coreGroup.Id) == 0 {
			continue
		}
		sgi := NewSGIFromCoreGroup(coreGroup)
		log.Printf("load group=%v, topHeight=%v\n", GetIDPrefix(sgi.GroupID), topHeight)
		if sgi.Dismissed(topHeight) {
			break
		}
		if sgi.MemExist(p.GetMinerID()) {
			jg := belongs.getJoinedGroup(sgi.GroupID)
			if jg == nil {
				log.Printf("prepareMiner get join group fail, gid=%v\n", sgi.GroupID)
			} else {
				p.joinGroup(jg, true)
			}
		}
		p.acceptGroup(sgi)
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
	ids := p.globalGroups.DismissGroups(topHeight)
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