package logical

import (
	"consensus/groupsig"
	"log"
	"time"
	"core"
	"consensus/rand"
	"middleware/types"
	"encoding/binary"
	"math/big"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2018/6/23 下午4:07
**  Description: 组生命周期, 包括建组, 解散组
*/

const (
	CHECK_CREATE_GROUP_HEIGHT_AFTER uint64 = 20	//启动建组的块高度差
)

type GroupManager struct {
	groupChain core.GroupChain
	mainChain	core.BlockChain
	processor *Processor
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
	
	return 0
}

//检查当前用户是否是属于建组的组, 返回组id
func (gm *GroupManager) checkCreateGroup() (bool, *StaticGroupInfo, *types.BlockHeader) {
	topHeight := gm.mainChain.QueryTopBlock().Height
	//todo 应该使用链接口, 取链上倒数第n块
	var theBH *types.BlockHeader
	h := topHeight - CHECK_CREATE_GROUP_HEIGHT_AFTER
	for theBH == nil {
		theBH = gm.mainChain.QueryBlockByHeight(h)
		h--
	}

	castGID := groupsig.DeserializeId(theBH.GroupId)
	if gm.processor.IsMinerGroup(*castGID) {
		sgi := gm.processor.getGroup(*castGID)
		if sgi.CastQualified(topHeight) && gm.checkKing(theBH, &sgi) {
			log.Printf("checkCreateNextGroup, topHeight=%v, theBH height=%v, king=%v\n", topHeight, theBH.Height, gm.processor.getPrefix())
			return true, &sgi, theBH
		}
	}

	return false, nil, nil

}


//检查当前用户是否是建组发起人
func (gm *GroupManager) checkKing(bh *types.BlockHeader, group *StaticGroupInfo) bool {
	data := gm.processor.getGroupSecret(group.GroupID).secretSign
	data = append(data, bh.Signature...)
	hash := rand.Data2CommonHash(data)
	biHash := hash.Big()

	var index int32 = -1
	mem := len(group.Members)
	if biHash.BitLen() > 0 {
		index = int32(biHash.Mod(biHash, big.NewInt(int64(mem))).Int64())
	}
	log.Printf("checkCreateNextGroup king index=%v, id=%v\n", index, GetIDPrefix(group.GetCastor(int(index))))
	if index < 0 {
		return false
	}
	return int32(group.GetMinerPos(gm.processor.GetMinerID())) == index
}

//todo 从链上获取所有候选者
func (this *GroupManager) getAllCandidates() []groupsig.ID {
    
}

func (this *GroupManager) selectCandidates() []groupsig.ID {

}

func (gm *GroupManager) CreateNextGroupRoutine()  {
    create, group, bh := gm.checkCreateGroup()
	if !create {
		return
	}

	var gis ConsensusGroupInitSummary

	gis.ParentID = group.GroupID

	gn := rand.Data2CommonHash([]byte(fmt.Sprintf("%s-%v", group.GroupID.GetHexString(), bh.Height))).String()
	gis.DummyID = *groupsig.NewIDFromString(gn)

	if gm.groupChain.GetGroupById(gis.DummyID.Serialize()) != nil {
		log.Printf("CreateNextGroupRoutine ingored, dummyId already onchain! dummyId=%v\n", GetIDPrefix(gis.DummyID))
		return
	}

	log.Printf("CreateNextGroupRoutine, group name=%v, group dummy id=%v.\n", gn, GetIDPrefix(gis.DummyID))
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

	memIds := gm.selectCandidates()
	gis.MemberHash = genMemberHashByIds(memIds)

	msg := ConsensusCreateGroupRawMessage{
		GI: gis,
		IDs: memIds,
	}
	msg.GenSign(SecKeyInfo{ID: gm.processor.GetMinerID(), SK: gm.processor.getMinerSignKey(group.GroupID)})


	log.Printf("proc(%v) start Create Group consensus, send network msg to members...\n", gm.processor.getPrefix())
	log.Printf("call network service SendCreateGroupRawMessage...\n")

	SendCreateGroupRawMessage(&msg)
}