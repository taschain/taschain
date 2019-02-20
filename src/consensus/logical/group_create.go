package logical

import (
	"middleware/types"
	"fmt"
	"consensus/base"
	"consensus/model"
	"strings"
	"logservice"
	"consensus/groupsig"
	"common"
)

/*
**  Creator: pxf
**  Date: 2019/2/18 下午2:18
**  Description: 
*/

func (gm *GroupManager) selectParentGroup(baseBH *types.BlockHeader, preGroupId []byte) (*StaticGroupInfo, error) {
	rand := baseBH.Random
	rand = append(rand, preGroupId...)
    gid, err := gm.processor.globalGroups.SelectNextGroupFromChain(base.Data2CommonHash(rand), baseBH.Height)
	if err != nil {
		return nil, err
	}
	return gm.processor.GetGroup(gid), nil
}


func (gm *GroupManager) generateCreateGroupContext(baseHeight uint64) (*createGroupBaseContext, error) {
	lastGroup := gm.groupChain.LastGroup()
	baseBH := gm.mainChain.QueryBlockByHeight(baseHeight)
	if baseBH == nil {
		return nil, fmt.Errorf("base block is nil, height=%v", baseHeight)
	}
	if !checkCreate(baseBH) {
		return nil, fmt.Errorf("cannot create group at the height")
	}
	sgi, err := gm.selectParentGroup(baseBH, lastGroup.Id)
	if err != nil {
		return nil, fmt.Errorf("select parent group err %v", err)
	}
	enough, candidates := gm.checker.selectCandidates(baseBH)
	if !enough {
		return nil, fmt.Errorf("not enough candidates")
	}
	return newCreateGroupBaseContext(sgi, baseBH, lastGroup, candidates), nil
}

func (gm *GroupManager) checkCreateGroupRoutine(baseHeight uint64)  {
    blog := newBizLog("checkCreateGroupRoutine")
    create := false
    var err error

    defer func() {
    	ret := ""
		if err != nil {
			ret = err.Error()
		}
		blog.log("baseBH height=%v, create=%v, ret=%v", baseHeight, create, ret)
	}()

	// 指定高度已经在组链上出现过
	if gm.checker.heightCreated(baseHeight) {
		err = fmt.Errorf("topHeight already created")
		return
	}

	baseCtx, err2 := gm.generateCreateGroupContext(baseHeight)
	if err2 != nil {
		err = err2
		return
	}

	if !gm.processor.IsMinerGroup(baseCtx.parentInfo.GroupID) {
		err = fmt.Errorf("next select group id %v, not belong to the group", baseCtx.parentInfo.GroupID.GetHexString())
		return
	}

	kings, isKing := gm.checker.selectKing(baseCtx.baseBH, baseCtx.parentInfo)

	gm.setCreatingGroupContext(baseCtx, kings, isKing)
	groupLogger.Infof("createGroupContext info=%v", gm.getContext().logString())

	gm.pingNodes()
	create = true

}


func (gm *GroupManager) pingNodes()  {
	ctx := gm.creatingGroupCtx
	if ctx == nil || !ctx.isKing() {
		return
	}
	msg := &model.CreateGroupPingMessage{
		FromGroupID: ctx.parentInfo.GroupID,
		PingID: ctx.pingID,
	}
	blog := newBizLog("pingNodes")
	if msg.GenSign(gm.processor.getDefaultSeckeyInfo(), msg) {
		for _, id := range ctx.candidates {
			blog.debug("baseHeight=%v, pingID=%v, id=%v", ctx.baseBH.Height, msg.PingID, id.ShortS())
			gm.processor.NetServer.SendGroupPingMessage(msg, id)
		}
	}
}


func (gm *GroupManager) checkReqCreateGroupSign(topHeight uint64) bool {
	blog := newBizLog("checkReqCreateGroupSign")
	ctx := gm.creatingGroupCtx
	if ctx == nil {
		return false
	}

	var desc string
	defer func() {
		if desc != "" {
			groupLogger.Infof("checkReqCreateGroupSign info=%v, %v", ctx.logString(), desc)
		}
	}()


	if ctx.readyTimeout(topHeight) {
		//blog.log("ctx readytimeout, baseHeight=%v", ctx.baseBH.Height)
		//desc = "ready timeout."
		//gm.removeContext()
		return false
	}

	pongsize := ctx.pongSize()
	deadline := ctx.pongDeadline(topHeight)
	//未收齐pong，且未到pong deadline
	if pongsize < len(ctx.candidates) && !deadline {
		blog.log("pongsize %v, deadline %v", pongsize, deadline)
		return false
	}

	//暂未收到足够的pong
	if pongsize < model.Param.GroupMemberMin {
		blog.log("got not enough pongs!, got %v", pongsize)
		desc = "not enough pongs."
		return false
	}

	if ctx.getStatus() != waitingPong  {
		return false
	}
	ctx.setStatus(waitingSign)

	if !ctx.generateGroupInitInfo() {
		return false
	}
	gInfo := ctx.gInfo
	gh := gInfo.GI.GHeader

	desc = fmt.Sprintf("generateGroupInitInfo gHash=%v, memsize=%v, wait sign", gh.Hash.ShortS(), gInfo.MemberSize())

	if !ctx.isKing() {
		return false
	}

	msg := &model.ConsensusCreateGroupRawMessage{
		GInfo: *gInfo,
	}
	ski := gm.processor.getInGroupSeckeyInfo(ctx.parentInfo.GroupID)
	if !msg.GenSign(ski, msg) {
		blog.debug("genSign fail, id=%v, sk=%v", ski.ID.ShortS(), ski.SK.ShortS())
		return false
	}

	memIdStrs := make([]string, 0)
	for _, mem := range gInfo.Mems {
		memIdStrs = append(memIdStrs, mem.ShortS())
	}
	newHashTraceLog("checkReqCreateGroupSign", gh.Hash, gm.processor.GetMinerID()).log("parent %v, members %v", ctx.parentInfo.GroupID.ShortS(), strings.Join(memIdStrs, ","))

	//发送日志
	le := &logservice.LogEntry{
		LogType: logservice.LogTypeCreateGroup,
		Height: gm.groupChain.Count(),
		Hash: gh.Hash.Hex(),
		Proposer: gm.processor.GetMinerID().GetHexString(),
	}
	if logservice.Instance.IsFirstNInternalNodesInGroup(ctx.kings, 3) {
		logservice.Instance.AddLog(le)
	}

	gm.processor.NetServer.SendCreateGroupRawMessage(msg)
	desc += "req sign"
	return true
}

func (gm *GroupManager) checkGroupInfo(gInfo *model.ConsensusGroupInitInfo) ([]groupsig.ID, bool, error) {
	gh := gInfo.GI.GHeader
	if gh.Hash != gh.GenHash() {
		return nil, false, fmt.Errorf("gh hash error, hash=%v, genHash=%v", gh.Hash.ShortS(), gh.GenHash().ShortS())
	}
	if !model.Param.IsGroupMemberCountLegal(len(gInfo.Mems)) {
		return nil, false, fmt.Errorf("group member size error %v(%v-%v)", len(gInfo.Mems), model.Param.GroupMemberMin, model.Param.GroupMemberMax)
	}
	baseBH := gm.mainChain.QueryBlockByHeight(gh.CreateHeight)
	if baseBH == nil {
		return nil, false, common.ErrCreateBlockNil
	}
	if !checkCreate(baseBH) {
		return nil, false, fmt.Errorf("cannot create at the height %v", baseBH.Height)
	}
	//前一组，父亲组是否存在
	preGroup := gm.groupChain.GetGroupById(gh.PreGroup)
	if preGroup == nil {
		return nil, false, fmt.Errorf("preGroup is nil, gid=%v", groupsig.DeserializeId(gh.PreGroup).ShortS())
	}
	parentGroup := gm.groupChain.GetGroupById(gh.Parent)
	if parentGroup == nil {
		return nil, false, fmt.Errorf("parentGroup is nil, gid=%v", groupsig.DeserializeId(gh.Parent).ShortS())
	}

	sgi, err := gm.selectParentGroup(baseBH, gh.PreGroup)
	if err != nil {
		return nil, false, fmt.Errorf("select parent group err %v", err)
	}
	pid := groupsig.DeserializeId(parentGroup.Id)
	if !sgi.GroupID.IsEqual(pid) {
		return nil, false, fmt.Errorf("select parent group not equal, expect %v, recieve %v", sgi.GroupID.ShortS(), pid.ShortS())
	}

	if !groupsig.VerifySig(sgi.GroupPK, gh.Hash.Bytes(), gInfo.GI.Signature) {
		return nil, false, fmt.Errorf("verify parent sign fail")
	}

	enough, candidates := gm.checker.selectCandidates(baseBH)
	if !enough {
		return nil, false, fmt.Errorf("not enough candidates")
	}
	//所选成员是否在指定候选人中
	for _, mem := range gInfo.Mems {
		find := false
		for _, cand := range candidates {
			if mem.IsEqual(cand) {
				find = true
				break
			}
		}
		if !find {
			return nil, false, fmt.Errorf("mem error: %v is not a legal candidate", mem.ShortS())
		}
	}

	return candidates, true, nil
}

func (gm *GroupManager) recoverGroupInitInfo(baseHeight uint64, mask []byte) (*model.ConsensusGroupInitInfo, error) {
    ctx, err := gm.generateCreateGroupContext(baseHeight)
	if err != nil {
		return nil, err
	}
	return ctx.createGroupInitInfo(mask), nil
}