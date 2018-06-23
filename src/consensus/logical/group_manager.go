package logical

import (
	"consensus/groupsig"
	"log"
	"time"
	"core"
	"consensus/rand"
)

/*
**  Creator: pxf
**  Date: 2018/6/23 下午4:07
**  Description: 组生命周期, 包括建组, 解散组
*/

type GroupManager struct {
	groupChain core.GroupChain
}

func newGroupManager() *GroupManager {
	return &GroupManager{}
}

//创建一个新建组。由（且有创建组权限的）父亲组节点发起。
//miners：待成组的矿工信息。ID，（和组无关的）矿工公钥。
//gn：组名。
func (gm *GroupManager) createNextDummyGroup(miners []PubKeyInfo, parent *StaticGroupInfo) int {
	if len(miners) != GROUP_MAX_MEMBERS {
		log.Printf("create group error, group max members=%v, real=%v.\n", GROUP_MAX_MEMBERS, len(miners))
		return -1
	}
	var gis ConsensusGroupInitSummary

	gis.ParentID = parent.GroupID

	gn := rand.Data2CommonHash([]byte(gis.ParentID.GetHexString() + time.Now().String())).String()
	gis.DummyID = *groupsig.NewIDFromString(gn)

	if gm.groupChain.GetGroupById(gis.DummyID.Serialize()) != nil {
		log.Printf("CreateDummyGroup ingored, dummyId already onchain! dummyId=%v\n", GetIDPrefix(gis.DummyID))
		return 0
	}

	log.Printf("create group, group name=%v, group dummy id=%v.\n", gn, GetIDPrefix(gis.DummyID))
	gis.Authority = 777
	if len(gn) <= 64 {
		copy(gis.Name[:], gn[:])
	} else {
		copy(gis.Name[:], gn[:64])
	}
	gis.BeginTime = time.Now()
	if !gis.ParentID.IsValid() || !gis.DummyID.IsValid() {
		panic("create group init summary failed")
	}
	gis.Members = uint64(GROUP_MAX_MEMBERS)
	gis.Extends = "Dummy"
	var grm ConsensusGroupRawMessage
	grm.MEMS = make([]PubKeyInfo, len(miners))
	copy(grm.MEMS[:], miners[:])
	grm.GI = gis
	grm.SI = GenSignData(grm.GI.GenHash(), gm.GetMinerID(), gm.getMinerInfo().GetDefaultSecKey())
	log.Printf("proc(%v) Create New Group, send network msg to members...\n", gm.getPrefix())
	log.Printf("call network service SendGroupInitMessage...\n")

	SendGroupInitMessage(grm)
	return 0
}

func (gm *GroupManager) CreateNextGroup()  {

}
