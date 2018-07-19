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
		storeFile = "joined_group.config." + common.GlobalConf.GetString("chain", "database", "")
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
	belongs.load()

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
			p.prepareForCast(gid)
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