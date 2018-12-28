package logical

import (
	"consensus/groupsig"
	"core"
	"fmt"
	"consensus/model"
	"strings"
	"common"
	"middleware/types"
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

	gh, memIds, threshold := gm.checker.generateGroupHeader(topHeight, top.CurTime, gm.groupChain.LastGroup())
	if gh == nil {
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
	msg.GenSign(model.NewSecKeyInfo(gm.processor.GetMinerID(), gm.processor.getSignKey(parentGroupId)), msg)

	creatingGroup := newCreateGroup(&gInfo, threshold)
	gm.addCreatingGroup(creatingGroup)

	blog.log("proc(%v) start Create Group consensus, send network msg to members, hash=%v...\n", gm.processor.getPrefix(), gh.Hash.ShortS())
	blog.log("call network service SendCreateGroupRawMessage...\n")
	memIdStrs := make([]string, 0)
	for _, mem := range memIds {
		memIdStrs = append(memIdStrs, mem.ShortS())
	}
	newHashTraceLog("CreateGroupRoutine", gh.Hash, gm.processor.GetMinerID()).log("parent %v, members %v", parentGroupId.ShortS(), strings.Join(memIdStrs, ","))

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

	//建组时高度是否存在 由于分叉处理，建组时的高度可能不存在
	//bh := gm.mainChain.QueryBlockByHeight(gh.CreateHeight)
	//if bh == nil {
	//	core.Logger.Debugf("createBlock is nil, height=%v", gh.CreateHeight)
	//	return false, common.ErrCreateBlockNil
	//}

	//生成组头是否与收到的一致
	expectGH, _, _ := gm.checker.generateGroupHeader(gh.CreateHeight, gh.BeginTime, gm.groupChain.LastGroup())
	if expectGH == nil {
		return false, fmt.Errorf("expect GroupHeader is nil")
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

	gm.processor.CreateHeightGroupsMutex.Lock()
	defer gm.processor.CreateHeightGroupsMutex.Unlock()

	stdLogger.Infof("AddGroupOnChain height:%d,id:%s\n", group.GroupHeight, common.BytesToAddress(group.Id).GetHexString())

	if _, ok := gm.processor.CreateHeightGroups[group.Header.CreateHeight]; !ok {
		err := gm.groupChain.AddGroup(group)
		if err != nil {
			stdLogger.Infof("ERROR:add group fail! hash=%v, err=%v\n", group.Header.Hash.ShortS(), err.Error())
			return
		} else {
			gm.processor.CreateHeightGroups[group.Header.CreateHeight] = group.Header.Hash.ShortS()
		}
	}

	stdLogger.Infof("AddGroupOnChain success, ID=%v, height=%v\n", sgi.GroupID.ShortS(), gm.groupChain.Count())

}
