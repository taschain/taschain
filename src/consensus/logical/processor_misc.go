package logical

import (
	"consensus/groupsig"
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
    rets := p.GroupChain.GetAllGroupID()
	if len(rets) == 0 {
		return
	}
	topHeight := p.MainChain.QueryTopBlock().Height

	storeFile := p.genBelongGroupStoreFile()

	belongs := NewBelongGroups(storeFile)
	if !belongs.load() {

	}

	log.Printf("prepareMiner get groups from groupchain, len=%v, belongGroup len=%v\n", len(rets), belongs.groupSize())
	for _, gidBytes := range rets {
		coreGroup := p.GroupChain.GetGroupById(gidBytes)
		if coreGroup == nil {
			panic("buildGlobalGroups getGroupById failed! gid=" + string(gidBytes))
		}
		if coreGroup.Id == nil || len(coreGroup.Id) == 0 {
			continue
		}
		sgi := NewSGIFromCoreGroup(coreGroup)
		log.Printf("load group=%v\n", GetIDPrefix(sgi.GroupID))
		if !sgi.CastQualified(topHeight) {
			continue
		}
		for _, mem := range coreGroup.Members {
			pkInfo := &PubKeyInfo{ID: *groupsig.DeserializeId(mem.Id), PK: *groupsig.DeserializePubkeyBytes(mem.PubKey)}
			sgi.addMember(pkInfo)
		}
		if !p.globalGroups.AddStaticGroup(sgi) {
			continue
		}
		if sgi.MemExist(p.GetMinerID()) {
			gid := sgi.GroupID
			jg := belongs.getJoinedGroup(gid)
			if jg == nil {
				log.Println("cannot find joinedgroup infos! gid=" + GetIDPrefix(gid))
				continue
			}
			p.joinGroup(jg, false)
			p.prepareForCast(sgi)
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
	ids := p.globalGroups.DismissGroups(topHeight)
	log.Printf("releaseRoutine: clean group %v\n", len(ids))
	p.globalGroups.RemoveGroups(ids)
	p.blockContexts.removeContexts(ids)
	p.belongGroups.leaveGroups(ids)
	for _, gid := range ids {
		log.Println("releaseRoutine DissolveGroupNet staticGroup gid ", GetIDPrefix(gid))
		p.Server.DissolveGroupNet(gid.GetString())
	}

    //释放超时未建成组的组网络和相应的dummy组
	p.joiningGroups.forEach(func(gc *GroupContext) bool {
		if gc.gis.IsExpired() || gc.gis.ReadyTimeout(topHeight) {
			log.Println("releaseRoutine DissolveGroupNet dummyGroup from joiningGroups gid ", GetIDPrefix(gc.gis.DummyID))
			p.Server.DissolveGroupNet(gc.gis.DummyID.GetString())
			p.joiningGroups.RemoveGroup(gc.gis.DummyID)
		}
		return true
	})
	p.groupManager.creatingGroups.forEach(func(cg *CreatingGroup) bool {
		if cg.gis.IsExpired() || cg.gis.ReadyTimeout(topHeight) {
			log.Println("releaseRoutine DissolveGroupNet dummyGroup from creatingGroups gid ", GetIDPrefix(cg.gis.DummyID))
			p.Server.DissolveGroupNet(cg.gis.DummyID.GetString())
			p.groupManager.creatingGroups.removeGroup(cg.gis.DummyID)
		}
		return true
	})
	return true
}