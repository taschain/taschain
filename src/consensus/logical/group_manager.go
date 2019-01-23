package logical

import (
	"consensus/groupsig"
	"core"
	"fmt"
	"consensus/model"
	"strings"
	"common"
	"middleware/types"
	"bytes"
)

/*
**  Creator: pxf
**  Date: 2018/6/23 下午4:07
**  Description: 组生命周期, 包括建组, 解散组
*/

type GroupManager struct {
	groupChain     *core.GroupChain
	mainChain      core.BlockChain
	processor      *Processor
	creatingGroups *CreatingGroups
	checker        *GroupCreateChecker
}

func NewGroupManager(processor *Processor) *GroupManager {
	return &GroupManager{
		processor:      processor,
		mainChain:      processor.MainChain,
		groupChain:     processor.GroupChain,
		creatingGroups: &CreatingGroups{},
		checker:        newGroupCreateChecker(processor),
	}
}

func (gm *GroupManager) addCreatingGroup(group *CreatingGroup) {
	gm.creatingGroups.addCreatingGroup(group)
}

func (gm *GroupManager) removeCreatingGroup(hash common.Hash) {
	gm.creatingGroups.removeGroup(hash)
}

func (gm *GroupManager) CreateNextGroupRoutine() {
	top := gm.mainChain.QueryTopBlock()
	topHeight := top.Height
	blog := newBizLog("CreateNextGroupRoutine")

	gh, memIds, threshold, kings, err := gm.checker.generateGroupHeader(topHeight, top.CurTime, gm.groupChain.LastGroup())
	if gh == nil {
		return
	}
	if err != nil {
		blog.log(err.Error())
	}
	isKing := false
	for _, id := range kings {
		if id.IsEqual(gm.processor.GetMinerID()) {
			isKing = true
			break
		}
	}
	if !isKing {
		blog.log("current proc is not a king! topHeight=%v", topHeight)
		return
	}

	parentGroupId := groupsig.DeserializeId(gh.Parent)

	gInfo := model.ConsensusGroupInitInfo{
		GI:   model.ConsensusGroupInitSummary{GHeader: gh},
		Mems: memIds,
	}
	msg := &model.ConsensusCreateGroupRawMessage{
		GInfo: gInfo,
	}
	ski := model.NewSecKeyInfo(gm.processor.GetMinerID(), gm.processor.getSignKey(parentGroupId))
	if !msg.GenSign(ski, msg) {
		blog.debug("genSign fail, id=%v, sk=%v, belong=%v", ski.ID.ShortS(), ski.SK.ShortS(), gm.processor.IsMinerGroup(parentGroupId))
		return
	}

	creatingGroup := newCreateGroup(&gInfo, threshold)
	gm.addCreatingGroup(creatingGroup)

	blog.log("proc(%v) start Create Group consensus, send network msg to members, hash=%v...", gm.processor.getPrefix(), gh.Hash.ShortS())
	memIdStrs := make([]string, 0)
	for _, mem := range memIds {
		memIdStrs = append(memIdStrs, mem.ShortS())
	}
	newHashTraceLog("CreateGroupRoutine", gh.Hash, gm.processor.GetMinerID()).log("parent %v, members %v", parentGroupId.ShortS(), strings.Join(memIdStrs, ","))

	blog.log("ski %v %v, sign %v, msg %v", ski.ID.GetHexString(), ski.SK.GetHexString(), msg.SI.DataSign.GetHexString(), msg.SI.DataHash.Hex())

	gm.processor.NetServer.SendCreateGroupRawMessage(msg)
}

func (gm *GroupManager) isGroupHeaderLegal(gh *types.GroupHeader) (bool, error) {
	if gh.Hash != gh.GenHash() {
		return false, fmt.Errorf("gh hash error, hash=%v, genHash=%v", gh.Hash.ShortS(), gh.GenHash().ShortS())
	}

	//前一组，父亲组是否存在
	preGroup := gm.groupChain.GetGroupById(gh.PreGroup)
	if preGroup == nil {
		return false, fmt.Errorf("preGroup is nil, gid=%v", groupsig.DeserializeId(gh.PreGroup).ShortS())
	}
	parentGroup := gm.groupChain.GetGroupById(gh.Parent)
	if parentGroup == nil {
		return false, fmt.Errorf("parentGroup is nil, gid=%v", groupsig.DeserializeId(gh.Parent).ShortS())
	}
	lastGroup := gm.groupChain.LastGroup()
	if !bytes.Equal(preGroup.Id, lastGroup.Id) {
		return false, fmt.Errorf("preGroup not equal to lastGroup")
	}

	//建组时高度是否存在 由于分叉处理，建组时的高度可能不存在
	//bh := gm.mainChain.QueryBlockByHeight(gh.CreateHeight)
	//if bh == nil {
	//	core.Logger.Debugf("createBlock is nil, height=%v", gh.CreateHeight)
	//	return false, common.ErrCreateBlockNil
	//}

	//生成组头是否与收到的一致
	expectGH, _, _, _, err := gm.checker.generateGroupHeader(gh.CreateHeight, gh.BeginTime, lastGroup)
	if expectGH == nil || err != nil {
		return false, err
	}
	if expectGH.Hash != gh.Hash {
		core.Logger.Debugf("hhhhhhhhh expect=%+v, rec=%+v\n", expectGH, gh)
		return false, fmt.Errorf("expectGroup hash differ from receive hash, expect %v, receive %v", expectGH.Hash.ShortS(), gh.Hash.ShortS())
	}

	return true, nil
}

func (gm *GroupManager) OnMessageCreateGroupRaw(msg *model.ConsensusCreateGroupRawMessage) bool {
	blog := newBizLog("OMCGR")
	blog.log("gHash=%v, sender=%v", msg.GInfo.GI.GetHash().ShortS(), msg.SI.SignMember.ShortS())

	if gm.creatingGroups.hasSentHash(msg.GInfo.GroupHash()) {
		blog.log("has sent initMsg, gHash=%v", msg.GInfo.GroupHash().ShortS())
		return false
	}

	if ok, err := gm.isGroupHeaderLegal(msg.GInfo.GI.GHeader); !ok {
		blog.log(err.Error())
		return false
	}
	return true

}

func (gm *GroupManager) OnMessageCreateGroupSign(msg *model.ConsensusCreateGroupSignMessage, creating *CreatingGroup) bool {
	blog := newBizLog("OMCGS")
	blog.log("gHash=%v, sender=%v", msg.GHash.ShortS(), msg.SI.SignMember.ShortS())
	gis := &creating.gInfo.GI

	if msg.GHash != gis.GetHash() {
		blog.log("creating group hash diff, gHash=%v", msg.GHash.ShortS())
		return false
	}

	height := gm.processor.MainChain.QueryTopBlock().Height
	if gis.ReadyTimeout(height) {
		blog.log("gis expired!")
		return false
	}
	accept := gm.creatingGroups.acceptPiece(gis.GetHash(), msg.SI.SignMember, msg.SI.DataSign)
	blog.log("accept result %v", accept)
	newHashTraceLog("OMCGS", gis.GetHash(), msg.SI.SignMember).log("OnMessageCreateGroupSign ret %v, %v", PIECE_RESULT(accept), creating.gSignGenerator.Brief())
	if accept == PIECE_THRESHOLD {
		gis.Signature = creating.gSignGenerator.GetGroupSign()
		return true
	}
	return false
}

func (gm *GroupManager) AddGroupOnChain(sgi *StaticGroupInfo) {
	group := ConvertStaticGroup2CoreGroup(sgi)

	stdLogger.Infof("AddGroupOnChain height:%d,id:%s\n", group.GroupHeight, sgi.GroupID.ShortS())


	if gm.groupChain.GetGroupById(group.Id) != nil {
		stdLogger.Debugf("group already onchain, accept, id=%v\n", sgi.GroupID.ShortS())
		gm.processor.acceptGroup(sgi)
	} else {
		top := gm.processor.MainChain.Height()
		if !sgi.GetReadyTimeout(top) {
			err := gm.groupChain.AddGroup(group)
			if err != nil {
				stdLogger.Infof("ERROR:add group fail! hash=%v, gid=%v, err=%v\n", group.Header.Hash.ShortS(), sgi.GroupID.ShortS(), err.Error())
				return
			}
			gm.checker.addHeightCreated(group.Header.CreateHeight)
			stdLogger.Infof("AddGroupOnChain success, ID=%v, height=%v\n", sgi.GroupID.ShortS(), gm.groupChain.Count())
		} else {
			stdLogger.Infof("AddGroupOnChain group ready timeout, gid %v, timeout height %v, top %v\n", sgi.GroupID.ShortS(), sgi.GInfo.GI.GHeader.ReadyHeight, top)
		}
	}


}
