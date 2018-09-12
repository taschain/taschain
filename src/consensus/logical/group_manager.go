package logical

import (
	"consensus/groupsig"
	"log"
	"time"
	"core"
	"fmt"
	"consensus/model"
	"consensus/base"
	"strings"
)

/*
**  Creator: pxf
**  Date: 2018/6/23 下午4:07
**  Description: 组生命周期, 包括建组, 解散组
*/


type GroupManager struct {
	groupChain     *core.GroupChain
	mainChain      core.BlockChainI
	processor      *Processor
	creatingGroups *CreatingGroups
	checker 		*GroupCreateChecker
}

func NewGroupManager(processor *Processor) *GroupManager {
	return &GroupManager{
		processor:      processor,
		mainChain:      processor.MainChain,
		groupChain:     processor.GroupChain,
		creatingGroups: &CreatingGroups{},
		checker: 		newGroupCreateChecker(processor),
	}
}

func (gm *GroupManager) addCreatingGroup(group *CreatingGroup)  {
    gm.creatingGroups.addCreatingGroup(group)
}

func (gm *GroupManager) removeCreatingGroup(id groupsig.ID)  {
    gm.creatingGroups.removeGroup(id)
}


func (gm *GroupManager) CreateNextGroupRoutine() {
	top := gm.mainChain.QueryTopBlock()
	topHeight := top.Height
	blog := newBizLog("CreateNextGroupRoutine")

	create, group, king, theBH := gm.checker.checkCreateGroup(topHeight)
	//不是当前组铸
	if !create {
		return
	}
	//不是当期用户铸
	if !king.IsEqual(gm.processor.GetMinerID()) {
		return
	}

	topGroup := gm.groupChain.LastGroup()

	var gis model.ConsensusGroupInitSummary

	gis.ParentID = group.GroupID
	gis.PrevGroupID = *groupsig.DeserializeId(topGroup.Id)

	gn := fmt.Sprintf("%s-%v", group.GroupID.GetHexString(), theBH.Height)
	bi := base.Data2CommonHash([]byte(gn)).Big()
	gis.DummyID = *groupsig.NewIDFromBigInt(bi)

	if gm.groupChain.GetGroupById(gis.DummyID.Serialize()) != nil {
		blog.log("ingored, dummyId already onchain! dummyId=%v\n", GetIDPrefix(gis.DummyID))
		return
	}

	blog.log("group name=%v, group dummy id=%v.\n", gn, GetIDPrefix(gis.DummyID))
	gis.Authority = 777
	if len(gn) <= 64 {
		copy(gis.Name[:], gn[:])
	} else {
		copy(gis.Name[:], gn[:64])
	}
	gis.BeginTime = time.Now()
	gis.TopHeight = topHeight
	gis.GetReadyHeight = topHeight + model.Param.GroupGetReadyGap
	gis.BeginCastHeight = gis.GetReadyHeight + model.Param.GroupCastQualifyGap
	gis.DismissHeight = gis.BeginCastHeight + model.Param.GroupCastDuration

	if !gis.ParentID.IsValid() || !gis.DummyID.IsValid() {
		panic("create group init summary failed")
	}
	gis.Extends = "Dummy"

	enough, memPkis := gm.checker.selectCandidates(theBH, topHeight)
	if !enough {
		return
	}
	memIds := make([]groupsig.ID, len(memPkis))
	for idx, mem := range memPkis {
		memIds[idx] = mem.ID
	}
	gis.WithMemberIds(memIds)

	msg := &model.ConsensusCreateGroupRawMessage{
		GI: gis,
		IDs: memIds,
	}
	msg.GenSign(model.NewSecKeyInfo(gm.processor.GetMinerID(), gm.processor.getSignKey(group.GroupID)), msg)

	creatingGroup := newCreateGroup(&gis, memPkis, group)
	gm.addCreatingGroup(creatingGroup)

	log.Printf("proc(%v) start Create Group consensus, send network msg to members, dummyId=%v...\n", gm.processor.getPrefix(), GetIDPrefix(gis.DummyID))
	log.Printf("call network service SendCreateGroupRawMessage...\n")
	memIdStrs := make([]string, 0)
	for _, mem := range memIds {
		memIdStrs = append(memIdStrs, GetIDPrefix(mem))
	}
	logKeyword("CreateGroupRoutine", GetIDPrefix(gis.DummyID), gm.processor.getPrefix(), "parent %v, members %v", GetIDPrefix(gis.ParentID), strings.Join(memIdStrs, ","))

	gm.processor.NetServer.SendCreateGroupRawMessage(msg)
}

func (gm *GroupManager) OnMessageCreateGroupRaw(msg *model.ConsensusCreateGroupRawMessage) bool {
	blog := newBizLog("OMCGR")
	blog.log("dummyId=%v, sender=%v\n", GetIDPrefix(msg.GI.DummyID), GetIDPrefix(msg.SI.SignMember))
	gis := &msg.GI
	if gis.GenHash() != msg.SI.DataHash {
		blog.log("hash diff\n")
		return false
	}

	preGroup := gm.groupChain.GetGroupById(msg.GI.PrevGroupID.Serialize())
	if preGroup == nil {
		blog.log("preGroup is nil, preGroupId=%v\n", GetIDPrefix(msg.GI.PrevGroupID))
		return false
	}

	memHash := model.GenMemberHashByIds(msg.IDs)
	if memHash != gis.MemberHash {
		blog.log("memberHash diff\n")
		return false
	}
	bh := gm.mainChain.QueryBlockByHeight(gis.TopHeight)
	if bh == nil {
		blog.log("theBH is nil, height=%v", gis.TopHeight)
		return false
	}
	create, _, king, theBH := gm.checker.checkCreateGroup(gis.TopHeight)
	if !create {
		blog.log("current group is not the next CastGroup!")
		return false
	}
	if !king.IsEqual(msg.SI.SignMember) {
		blog.log("not the user for casting! expect user is %v, receive user is %v\n", GetIDPrefix(king), GetIDPrefix(msg.SI.SignMember))
		return false
	}

	enough, memPkis := gm.checker.selectCandidates(theBH, gis.TopHeight)
	if !enough {
		return false
	}
	memIds := make([]groupsig.ID, len(memPkis))
	for idx, mem := range memPkis {
		memIds[idx] = mem.ID
	}
	if len(memIds) != len(msg.IDs) {
		blog.log("member len not equal, expect len %v, receive len %v\n", len(memIds), len(msg.IDs))
		return  false
	}

	for idx, id := range memIds {
		if !id.IsEqual(msg.IDs[idx]) {
			blog.log("member diff [%v, %v]", GetIDPrefix(id), GetIDPrefix(msg.IDs[idx]))
			return  false
		}
	}
	return true

}

func (gm *GroupManager) OnMessageCreateGroupSign(msg *model.ConsensusCreateGroupSignMessage) bool {
	blog := newBizLog("OMCGS")
	blog.log("dummyId=%v, sender=%v\n", GetIDPrefix(msg.GI.DummyID), GetIDPrefix(msg.SI.SignMember))
	gis := &msg.GI
	if gis.GenHash() != msg.SI.DataHash {
		blog.log("hash diff\n")
		return false
	}

	creating := gm.creatingGroups.getCreatingGroup(gis.DummyID)
	if creating == nil {
		blog.log("get creating group nil!\n")
		return false
	}

	memHash := model.GenMemberHashByIds(creating.getIDs())
	if memHash != gis.MemberHash {
		blog.log("memberHash diff\n")
		return false
	}

	height := gm.processor.MainChain.QueryTopBlock().Height
	if gis.ReadyTimeout(height) {
		blog.log("gis expired!\n")
		return false
	}
	accept := gm.creatingGroups.acceptPiece(gis.DummyID, msg.SI.SignMember, msg.SI.DataSign)
	blog.log("accept result %v\n", accept)
	logKeyword("OMCGS", GetIDPrefix(msg.GI.DummyID), GetIDPrefix(msg.SI.SignMember), "OnMessageCreateGroupSign ret %v, %v", PIECE_RESULT(accept), creating.gSignGenerator.Brief())
	if accept == PIECE_THRESHOLD {
		sig := creating.gSignGenerator.GetGroupSign()
		msg.GI.Signature = sig
		return true
	}
	return false
}

func (gm *GroupManager) AddGroupOnChain(sgi *StaticGroupInfo, isDummy bool)  {
	group := ConvertStaticGroup2CoreGroup(sgi, isDummy)
	err := gm.groupChain.AddGroup(group, nil, nil)
	if err != nil {
		log.Printf("ERROR:add group fail! isDummy=%v, dummyId=%v, err=%v\n", isDummy, GetIDPrefix(sgi.GIS.DummyID), err.Error())
		return
	}
	if isDummy {
		log.Printf("AddGroupOnChain success, dummyId=%v, height=%v\n", GetIDPrefix(sgi.GIS.DummyID), gm.groupChain.Count())
	} else {
		mems := make([]groupsig.ID, 0)
		for _, mem := range sgi.Members {
			mems = append(mems, mem.ID)
		}
		gm.processor.NetServer.BuildGroupNet(sgi.GroupID, mems)
		log.Printf("AddGroupOnChain success, ID=%v, height=%v\n", GetIDPrefix(sgi.GroupID), gm.groupChain.Count())
	}
}