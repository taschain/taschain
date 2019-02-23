package logical

import (
	"consensus/model"
	"fmt"
	"consensus/groupsig"
	"consensus/net"
	"time"
)

/*
**  Creator: pxf
**  Date: 2019/2/18 下午3:51
**  Description: 
*/

func (p *Processor) OnMessageCreateGroupPing(msg *model.CreateGroupPingMessage)  {
	blog := newBizLog("OMCGPing")
	var err error
	defer func() {
		if err != nil {
			blog.log("from %v, gid %v, pingId %v, won't pong, err=%v", msg.SI.GetID().ShortS(), msg.FromGroupID.ShortS(), msg.PingID, err)
		} else {
			blog.log("from %v, gid %v, pingId %v, pong!", msg.SI.GetID().ShortS(), msg.FromGroupID.ShortS(), msg.PingID)
		}
	}()
	pk := GetMinerPK(msg.SI.GetID())
	if pk == nil {
		return
	}
    if msg.VerifySign(*pk) {
    	pongMsg := &model.CreateGroupPongMessage{
    		PingID: msg.PingID,
    		Ts: time.Now(),
		}
		group := p.GetGroup(msg.FromGroupID)
		gb := &net.GroupBrief{
			Gid: msg.FromGroupID,
			MemIds: group.GetMembers(),
		}
		if pongMsg.GenSign(p.getDefaultSeckeyInfo(), pongMsg) {
			p.NetServer.SendGroupPongMessage(pongMsg, gb)
		} else {
			err = fmt.Errorf("gen sign fail")
		}
	} else {
		err = fmt.Errorf("verify sign fail")
	}
}

func (p *Processor) OnMessageCreateGroupPong(msg *model.CreateGroupPongMessage)  {
	blog := newBizLog("OMCGPong")
	var err error
	defer func() {
		blog.log("from %v, pingId %v, got pong, ret=%v", msg.SI.GetID().ShortS(), msg.PingID, err)
	}()

	ctx := p.groupManager.getContext()
	if ctx == nil {
		err = fmt.Errorf("creatingGroupCtx is nil")
		return
	}
	if ctx.pingID != msg.PingID {
		err = fmt.Errorf("pingId not equal, expect=%v, got=%v", p.groupManager.creatingGroupCtx.pingID, msg.PingID)
		return
	}
	pk := GetMinerPK(msg.SI.GetID())
	if pk == nil {
		return
	}

	if msg.VerifySign(*pk) {
		add, got := ctx.addPong(p.MainChain.Height(), msg.SI.GetID())
		err = fmt.Errorf("size %v", got)
		if add {
			p.groupManager.checkReqCreateGroupSign(p.MainChain.Height())
		}
	} else {
		err = fmt.Errorf("verify sign fail")
	}
}


func (p *Processor) OnMessageCreateGroupRaw(msg *model.ConsensusCreateGroupRawMessage) {
	blog := newBizLog("OMCGR")

	gh := msg.GInfo.GI.GHeader
	blog.log("Proc(%v) begin, gHash=%v sender=%v", p.getPrefix(), gh.Hash.ShortS(), msg.SI.SignMember.ShortS())

	if p.GetMinerID().IsEqual(msg.SI.SignMember) {
		return
	}
	parentGid := msg.GInfo.GI.ParentID()

	gpk, ok := p.GetMemberSignPubKey(model.NewGroupMinerID(parentGid, msg.SI.SignMember))
	if !ok {
		blog.log("GetMemberSignPubKey not ok, ask id %v", parentGid.ShortS())
		return
	}

	if !msg.VerifySign(gpk) {
		return
	}
	if gh.Hash != gh.GenHash() || gh.Hash != msg.SI.DataHash {
		blog.log("hash diff expect %v, receive %v", gh.GenHash().ShortS(), gh.Hash.ShortS())
		return
	}

	tlog := newHashTraceLog("OMCGR", gh.Hash, msg.SI.GetID())
	if ok, err := p.groupManager.OnMessageCreateGroupRaw(msg); ok {
		signMsg := &model.ConsensusCreateGroupSignMessage{
			Launcher: msg.SI.SignMember,
			GHash: gh.Hash,
		}
		ski := p.getInGroupSeckeyInfo(parentGid)
		if signMsg.GenSign(ski, signMsg) {
			tlog.log("SendCreateGroupSignMessage id=%v", p.getPrefix())
			blog.debug("OMCGR SendCreateGroupSignMessage... ")
			p.NetServer.SendCreateGroupSignMessage(signMsg, parentGid)
		} else {
			blog.debug("SendCreateGroupSignMessage sign fail, ski=%v, %v", ski.ID.ShortS(), ski.SK.ShortS(), p.IsMinerGroup(parentGid))
		}

	} else {
		tlog.log("groupManager.OnMessageCreateGroupRaw fail, err:%v", err.Error())
	}
}

func (p *Processor) OnMessageCreateGroupSign(msg *model.ConsensusCreateGroupSignMessage) {
	blog := newBizLog("OMCGS")

	blog.log("Proc(%v) begin, gHash=%v, sender=%v", p.getPrefix(), msg.GHash.ShortS(), msg.SI.SignMember.ShortS())
	if p.GetMinerID().IsEqual(msg.SI.SignMember) {
		return
	}

	if msg.GenHash() != msg.SI.DataHash {
		blog.log("hash diff")
		return
	}

	ctx := p.groupManager.getContext()
	if ctx == nil {
		blog.log("context is nil")
		return
	}
	mpk, ok := p.GetMemberSignPubKey(model.NewGroupMinerID(ctx.parentInfo.GroupID, msg.SI.SignMember))
	if !ok {
		blog.log("GetMemberSignPubKey not ok, ask id %v", ctx.parentInfo.GroupID.ShortS())
		return
	}
	if !msg.VerifySign(mpk) {
		return
	}
	if ok, err := p.groupManager.OnMessageCreateGroupSign(msg); ok {
		gpk := ctx.parentInfo.GroupPK
		if !groupsig.VerifySig(gpk, msg.SI.DataHash.Bytes(), ctx.gInfo.GI.Signature) {
			blog.log("Proc(%v) verify group sign fail", p.getPrefix())
			return
		}
		initMsg := &model.ConsensusGroupRawMessage{
			GInfo:   *ctx.gInfo,
		}

		blog.debug("Proc(%v) send group init Message", p.getPrefix())
		ski := p.getDefaultSeckeyInfo()
		if initMsg.GenSign(ski, initMsg) && ctx.getStatus() != sendInit {
			tlog := newHashTraceLog("OMCGS", msg.GHash, msg.SI.GetID())
			tlog.log("收齐分片，SendGroupInitMessage")
			p.NetServer.SendGroupInitMessage(initMsg)
			ctx.setStatus(sendInit)
			groupLogger.Infof("OMCGS send group init: info=%v, gHash=%v, costHeight=%v", ctx.logString(), ctx.gInfo.GroupHash().ShortS(), p.MainChain.Height()-ctx.createTopHeight)

		} else {
			blog.log("genSign fail, id=%v, sk=%v", ski.ID.ShortS(), ski.SK.ShortS())
		}

	} else {
		blog.log("fail, err=%v", err)
	}
}
///////////////////////////////////////////////////////////////////////////////
//组初始化相关消息
//组初始化的相关消息都用（和组无关的）矿工ID和公钥验签

func (p *Processor) OnMessageGroupInit(msg *model.ConsensusGroupRawMessage) {
	blog := newBizLog("OMGI")
	gHash := msg.GInfo.GroupHash()
	gis := &msg.GInfo.GI
	gh := gis.GHeader

	blog.log("proc(%v) begin, sender=%v, gHash=%v...", p.getPrefix(), msg.SI.GetID().ShortS(), gHash.ShortS())
	tlog := newHashTraceLog("OMGI", gHash, msg.SI.GetID())

	if msg.SI.DataHash != msg.GenHash() || gh.Hash != gh.GenHash() {
		panic("msg gis hash diff")
	}
	//非组内成员不走后续流程
	if !msg.MemberExist(p.GetMinerID()) {
		return
	}

	var desc string
	defer func() {
		if desc != "" {
			groupLogger.Infof("OMGI:gHash=%v,sender=%v, %v", msg.GInfo.GroupHash().ShortS(), msg.SI.GetID().ShortS(), desc)
		}
	}()

	groupContext := p.joiningGroups.GetGroup(gHash)
	if groupContext != nil && groupContext.GetGroupStatus() != GisInit {
		blog.debug("already handle, status=%v", groupContext.GetGroupStatus())
		return
	}

	topHeight := p.MainChain.QueryTopBlock().Height
	if gis.ReadyTimeout(topHeight) {
		desc = fmt.Sprintf("OMGI ready timeout, readyHeight=%v, now=%v", gh.ReadyHeight, topHeight)
		blog.debug(desc)
		return
	}

	var candidates []groupsig.ID
	if cands, ok, err := p.groupManager.checkGroupInfo(&msg.GInfo); !ok {
		blog.debug("group header illegal, err=%v", err)
		return
	} else {
		candidates = cands
	}

	//if p.globalGroups.AddInitingGroup(CreateInitingGroup(msg)) {
	//	//to do : 从链上检查消息发起人（父亲组成员）是否有权限发该消息（鸠兹）
	//	//dummy 组写入组链 add by 小熊
	//	//staticGroupInfo := NewDummySGIFromGroupRawMessage(msg)
	//	//p.groupManager.AddGroupOnChain(staticGroupInfo, true)
	//}

	tlog.logStart("%v", "")

	groupContext = p.joiningGroups.ConfirmGroupFromRaw(msg, candidates, p.mi)
	if groupContext == nil {
		panic("Processor::OMGI failed, ConfirmGroupFromRaw return nil.")
	}

	//提前建立组网络
	p.NetServer.BuildGroupNet(gHash.Hex(), msg.GInfo.Mems)

	gs := groupContext.GetGroupStatus()
	blog.debug("joining group(%v) status=%v.", gHash.ShortS(), gs)
	if groupContext.StatusTransfrom(GisInit, GisSendSharePiece) {
		//blog.log("begin GenSharePieces in OMGI...")
		desc = "send sharepiece"

		shares := groupContext.GenSharePieces() //生成秘密分享
		//blog.log("proc(%v) end GenSharePieces in OMGI, piece size=%v.", p.getPrefix(), len(shares))

		spm := &model.ConsensusSharePieceMessage{
			GHash: gHash,
		}
		ski := model.NewSecKeyInfo(p.GetMinerID(), p.mi.GetDefaultSecKey())
		spm.SI.SignMember = p.GetMinerID()
		spm.MemCnt = int32(msg.GInfo.MemberSize())

		for id, piece := range shares {
			if id != "0x0" && piece.IsValid() {
				spm.Dest.SetHexString(id)
				spm.Share = piece
				if spm.GenSign(ski, spm) {
					//blog.log("OMGI spm.GenSign result=%v.", sb)
					blog.debug("piece to ID(%v), gHash=%v, share=%v, pub=%v.", spm.Dest.ShortS(), gHash.ShortS(), spm.Share.Share.ShortS(), spm.Share.Pub.ShortS())
					tlog.log("sharepiece to %v", spm.Dest.ShortS())
					blog.debug("call network service SendKeySharePiece...")
					p.NetServer.SendKeySharePiece(spm)
				} else {
					blog.log("genSign fail, id=%v, sk=%v", ski.ID.ShortS(), ski.SK.ShortS())
				}

			} else {
				panic("GenSharePieces data not IsValid.")
			}
		}
		//blog.log("end GenSharePieces.")
	}

	//blog.log("proc(%v) end OMGI, sender=%v.", p.getPrefix(), GetIDPrefix(msg.SI.GetID()))
	return
}

//收到组内成员发给我的秘密分享片段消息
func (p *Processor) OnMessageSharePiece(spm *model.ConsensusSharePieceMessage) {
	blog := newBizLog("OMSP")
	gHash := spm.GHash

	blog.log("proc(%v)begin Processor::OMSP, sender=%v, gHash=%v...", p.getPrefix(), spm.SI.GetID().ShortS(), gHash.ShortS())
	tlog := newHashTraceLog("OMSP", gHash, spm.SI.GetID())

	if !spm.Dest.IsEqual(p.GetMinerID()) {
		return
	}

	gc := p.joiningGroups.GetGroup(gHash)
	if gc == nil {
		blog.debug("failed, receive SHAREPIECE msg but gc=nil.")
		return
	}
	if gc.gInfo.GroupHash() != spm.GHash {
		blog.debug("failed, gisHash diff.")
		return
	}

	pk := GetMinerPK(spm.SI.GetID())
	if pk == nil {
		blog.debug("miner pk is nil, id=%v", spm.SI.GetID().ShortS())
		return
	}
	if !spm.VerifySign(*pk) {
		blog.debug("miner sign verify fail")
		return
	}

	gh := gc.gInfo.GI.GHeader

	topHeight := p.MainChain.QueryTopBlock().Height
	if gc.gInfo.GI.ReadyTimeout(topHeight) {
		blog.debug("ready timeout, readyHeight=%v, now=%v", gh.ReadyHeight, topHeight)
		return
	}

	result := gc.PieceMessage(spm)
	waitPieceIds := make([]string, 0)
	for _, mem := range gc.gInfo.Mems {
		if !gc.node.hasPiece(mem) {
			waitPieceIds = append(waitPieceIds, mem.ShortS())
			if len(waitPieceIds) >= 10 {
				break
			}
		}
	}
	blog.log("proc(%v) after gc.PieceMessage, gc result=%v. lackof %v", p.getPrefix(), result, waitPieceIds)

	tlog.log("收到piece数 %v, 收齐分片%v, 缺少%v等", gc.node.groupInitPool.GetSize(), result == 1, waitPieceIds)

	if result == 1 { //已聚合出签名私钥,组公钥，组id
		groupLogger.Infof("OMSP收齐分片: gHash=%v, elapsed=%v.", spm.GHash.ShortS(), time.Since(gc.createTime).String())
		jg := gc.GetGroupInfo()
		p.joinGroup(jg)
		//这时还没有所有组成员的签名公钥
		if jg.GroupPK.IsValid() && jg.SignKey.IsValid() {
			ski := model.NewSecKeyInfo(p.mi.GetMinerID(), jg.SignKey)
			//1. 先发送自己的签名公钥
			if gc.StatusTransfrom(GisSendSharePiece, GisSendSignPk) {
				msg := &model.ConsensusSignPubKeyMessage{
					GroupID: jg.GroupID,
					SignPK:  *groupsig.NewPubkeyFromSeckey(jg.SignKey),
					GHash:   spm.GHash,
					MemCnt:  int32(gc.gInfo.MemberSize()),
				}
				if !msg.SignPK.IsValid() {
					panic("signPK is InValid")
				}
				if msg.GenSign(ski, msg) {
					blog.debug("send sign pub key to group members, spk=%v...", msg.SignPK.ShortS())
					tlog.log("SendSignPubKey %v", p.getPrefix())

					blog.debug("call network service SendSignPubKey...")
					p.NetServer.SendSignPubKey(msg)
				} else {
					blog.log("genSign fail, id=%v, sk=%v", ski.ID.ShortS(), ski.SK.ShortS())
				}
			}
			//2. 再发送初始化完成消息
			if gc.StatusTransfrom(GisSendSignPk, GisSendInited) {
				msg := &model.ConsensusGroupInitedMessage{
					GHash: spm.GHash,
					GroupPK: jg.GroupPK,
					GroupID: jg.GroupID,
					CreateHeight: gh.CreateHeight,
					ParentSign: gc.gInfo.GI.Signature,
					MemCnt:  int32(gc.gInfo.MemberSize()),
					MemMask: gc.generateMemberMask(),
				}
				ski := model.NewSecKeyInfo(p.mi.GetMinerID(), p.mi.GetDefaultSecKey())

				if msg.GenSign(ski, msg) {
					tlog.log("BroadcastGroupInfo %v", jg.GroupID.ShortS())

					blog.debug("call network service BroadcastGroupInfo...")
					p.NetServer.BroadcastGroupInfo(msg)
				} else {
					blog.log("genSign fail, id=%v, sk=%v", ski.ID.ShortS(), ski.SK.ShortS())
				}
			}
		} else {
			panic("Processor::OMSP failed, aggr key error.")
		}
	}

	//blog.log("prov(%v) end OMSP, sender=%v.", p.getPrefix(), GetIDPrefix(spm.SI.GetID()))
	return
}

//收到组内成员发给我的组成员签名公钥消息
func (p *Processor) OnMessageSignPK(spkm *model.ConsensusSignPubKeyMessage) {
	blog := newBizLog("OMSPK")
	tlog := newHashTraceLog("OMSPK", spkm.GHash, spkm.SI.GetID())

	blog.log("proc(%v) begin , sender=%v, gHash=%v, gid=%v...", p.getPrefix(), spkm.SI.GetID().ShortS(), spkm.GHash.ShortS(), spkm.GroupID.ShortS())

	//jg := p.belongGroups.getJoinedGroup(spkm.GroupID)
	//if jg == nil {
	//	blog.debug("failed, local node not found joinedGroup with group id=%v.", spkm.GroupID.ShortS())
	//	return
	//}
	if spkm.GenHash() != spkm.SI.DataHash {
		blog.log("spkm hash diff")
		return
	}

	if !spkm.VerifySign(spkm.SignPK) {
		blog.log("miner sign verify fail")
		return
	}

	removeSignPkRecord(spkm.SI.GetID())

	jg, ret := p.belongGroups.addMemSignPk(spkm.SI.GetID(), spkm.GroupID, spkm.SignPK)

	if jg != nil {
		blog.log("after SignPKMessage exist mem sign pks=%v, ret=%v", jg.memSignPKSize(), ret)
		tlog.log("收到签名公钥数 %v", jg.memSignPKSize())
		for mem, pk := range jg.getMemberMap() {
			blog.log("signPKS: %v, %v", mem, pk.GetHexString())
		}
	}

	//blog.log("proc(%v) end OMSPK, sender=%v, dummy gid=%v.", p.getPrefix(), GetIDPrefix(spkm.SI.GetID()), GetIDPrefix(spkm.DummyID))
	return
}

func (p *Processor) OnMessageSignPKReq(msg *model.ConsensusSignPubkeyReqMessage) {
	blog := newBizLog("OMSPKR")
	sender := msg.SI.GetID()
	var err error
	defer func() {
		blog.log("sender=%v, gid=%v, result=%v", sender.ShortS(), msg.GroupID.ShortS(), err.Error())
	}()

	jg := p.belongGroups.getJoinedGroup(msg.GroupID)
	if jg == nil {
		err = fmt.Errorf("failed, local node not found joinedGroup with group id=%v", msg.GroupID.ShortS())
		return
	}

	pk := GetMinerPK(sender)
	if pk == nil {
		err = fmt.Errorf("get minerPK is nil, id=%v", sender.ShortS())
		return
	}
	if !msg.VerifySign(*pk) {
		err = fmt.Errorf("verifySign fail, pk=%v, sign=%v", pk.GetHexString(), msg.SI.DataSign.GetHexString())
		return
	}
	if !jg.SignKey.IsValid() {
		err = fmt.Errorf("invalid sign secKey, id=%v, sk=%v", p.GetMinerID().ShortS(), jg.SignKey.ShortS())
		return
	}

	//signPk, ok := p.GetMemberSignPubKey(model.NewGroupMinerID(jg.GroupID, p.GetMinerID()))
	//if !ok {
	//	err = fmt.Errorf("current node dosen't has signPk in group %v", jg.GroupID.ShortS())
	//	return
	//}
	//if !signPk.IsValid() {
	//	err = fmt.Errorf("current node signPK is invalid, pk=%v", signPk.GetHexString())
	//	return
	//}
	resp := &model.ConsensusSignPubKeyMessage{
		GHash: jg.gHash,
		GroupID: msg.GroupID,
		SignPK: *groupsig.NewPubkeyFromSeckey(jg.SignKey),
	}
	ski := model.NewSecKeyInfo(p.GetMinerID(), jg.SignKey)
	if resp.GenSign(ski, resp) {
		blog.log("answer signPKReq Message, receiver %v, gid %v", sender.ShortS(), msg.GroupID.ShortS())
		p.NetServer.AnswerSignPkMessage(resp, sender)
	} else {
		err = fmt.Errorf("gen Sign fail, ski=%v,%v", ski.ID.ShortS(), ski.SK.GetHexString() )
	}
}

func (p *Processor) acceptGroup(staticGroup *StaticGroupInfo) {
	add := p.globalGroups.AddStaticGroup(staticGroup)
	blog := newBizLog("acceptGroup")
	blog.debug("Add to Global static groups, result=%v, groups=%v.", add, p.globalGroups.GetGroupSize())
	if staticGroup.MemExist(p.GetMinerID()) {
		p.prepareForCast(staticGroup)
		//jg := p.belongGroups.getJoinedGroup(staticGroup.GroupID)
		//if jg != nil {
		//} else {
		//	blog.log("[ERROR]cannot find joined group info, gid=%v", staticGroup.GroupID.ShortS())
		//}
	}
}

//全网节点收到某组已初始化完成消息（在一个时间窗口内收到该组51%成员的消息相同，才确认上链）
//最终版本修改为父亲节点进行验证（51%）和上链
//全网节点处理函数->to do : 调整为父亲组节点处理函数
func (p *Processor) OnMessageGroupInited(msg *model.ConsensusGroupInitedMessage) {
	blog := newBizLog("OMGIED")
	gHash := msg.GHash

	blog.log("proc(%v) begin, sender=%v, gHash=%v, gid=%v, gpk=%v...", p.getPrefix(),
		msg.SI.GetID().ShortS(), gHash.ShortS(), msg.GroupID.ShortS(), msg.GroupPK.ShortS())
	tlog := newHashTraceLog("OMGIED", gHash, msg.SI.GetID())

	if msg.SI.DataHash != msg.GenHash() {
		panic("grm gis hash diff")
	}

	//此时组通过同步已上链了，但是后面的逻辑还是要执行，否则组数据状态有问题
	//g := p.GroupChain.GetGroupById(msg.GroupID.Serialize())
	//if g != nil {
	//	blog.log("group already onchain")
	//	return
	//}

	pk := GetMinerPK(msg.SI.GetID())
	if !msg.VerifySign(*pk) {
		blog.log("verify sign fail, id=%v, pk=%v, sign=%v",  msg.SI.GetID().ShortS(), pk.GetHexString(), msg.SI.DataSign.GetHexString())
		return
	}

	initedGroup := p.globalGroups.GetInitedGroup(msg.GHash)
	if initedGroup == nil {
		gInfo, err := p.groupManager.recoverGroupInitInfo(msg.CreateHeight, msg.MemMask)
		if err != nil {
			blog.log("recover group info fail, err %v", err)
			return
		}
		if gInfo.GroupHash() != msg.GHash {
			blog.log("groupHeader hash error, expect %v, receive %v", gInfo.GroupHash(), msg.GHash.Hex())
			return
		}
		gInfo.GI.Signature = msg.ParentSign
		initedGroup = createInitedGroup(gInfo)
		//initedGroup = p.globalGroups.generator.addInitedGroup(ig)
		blog.log("add inited group")
	}
	if initedGroup.gInfo.GI.ReadyTimeout(p.MainChain.Height()) {
		blog.log("group ready timeout, gid=%v", msg.GroupID.ShortS())
		return
	}

	parentId := initedGroup.gInfo.GI.ParentID()
	parentGroup := p.GetGroup(parentId)

	gpk := parentGroup.GroupPK
	if !groupsig.VerifySig(gpk, msg.GHash.Bytes(), msg.ParentSign) {
		blog.log("verify parent groupsig fail! gHash=%v", gHash.ShortS())
		return
	}
	if !initedGroup.gInfo.GI.Signature.IsEqual(msg.ParentSign) {
		blog.log("signature differ, old %v, new %v", initedGroup.gInfo.GI.Signature.GetHexString(), msg.ParentSign.GetHexString())
		return
	}
	initedGroup = p.globalGroups.generator.addInitedGroup(initedGroup)

	result := initedGroup.receive(msg.SI.GetID(), msg.GroupPK)

	waitIds := make([]string, 0)
	for _, mem := range initedGroup.gInfo.Mems {
		if !initedGroup.hasRecived(mem) {
			waitIds = append(waitIds, mem.ShortS())
			if len(waitIds) >= 10 {
				break
			}
		}
	}

	tlog.log("ret:%v,收到消息数量 %v, 需要消息数 %v, 缺少%v等", result, initedGroup.receiveSize(), initedGroup.threshold, waitIds)

	switch result {
	case INIT_SUCCESS: //收到组内相同消息>=阈值，可上链
		staticGroup := NewSGIFromStaticGroupSummary(msg.GroupID, msg.GroupPK, initedGroup)
		gh := staticGroup.getGroupHeader()
		blog.debug("SUCCESS accept a new group, gHash=%v, gid=%v, workHeight=%v, dismissHeight=%v.", gHash.ShortS(), msg.GroupID.ShortS(), gh.WorkHeight, gh.DismissHeight)

		//p.acceptGroup(staticGroup)
		p.groupManager.AddGroupOnChain(staticGroup)
		p.globalGroups.removeInitedGroup(gHash)
		p.joiningGroups.Clean(gHash)

	case INIT_FAIL: //该组初始化异常，且无法恢复
		tlog.log("初始化失败")
		p.globalGroups.removeInitedGroup(gHash)

	case INITING:
		//继续等待下一包数据
	}
	//blog.log("proc(%v) end OMGIED, sender=%v...", p.getPrefix(), GetIDPrefix(msg.SI.GetID()))
	return
}
