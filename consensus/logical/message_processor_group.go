//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package logical

import (
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/consensus/net"
	"time"
)

func (p *Processor) OnMessageCreateGroupPing(msg *model.CreateGroupPingMessage) {
	blog := newBizLog("OMCGPing")
	var err error
	defer func() {
		if err != nil {
			blog.log("from %v, gid %v, pingId %v, height=%v, won't pong, err=%v", msg.SI.GetID().ShortS(), msg.FromGroupID.ShortS(), msg.PingID, msg.BaseHeight, err)
		} else {
			blog.log("from %v, gid %v, pingId %v, height=%v, pong!", msg.SI.GetID().ShortS(), msg.FromGroupID.ShortS(), msg.PingID, msg.BaseHeight)
		}
	}()
	pk := GetMinerPK(msg.SI.GetID())
	if pk == nil {
		return
	}
	if msg.VerifySign(*pk) {
		top := p.MainChain.Height()
		if top <= msg.BaseHeight {
			err = fmt.Errorf("localheight is %v, not enough", top)
			return
		}
		pongMsg := &model.CreateGroupPongMessage{
			PingID: msg.PingID,
			Ts:     time.Now(),
		}
		group := p.GetGroup(msg.FromGroupID)
		gb := &net.GroupBrief{
			Gid:    msg.FromGroupID,
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

func (p *Processor) OnMessageCreateGroupPong(msg *model.CreateGroupPongMessage) {
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

	begin := time.Now()
	defer func() {
		if time.Since(begin).Seconds() > 0.5 {
			slowLogger.Warnf("handle slow:%v, sender=%v, gHash=%v, cost %v", "OMCGR", msg.SI.GetID().ShortS(), msg.GInfo.GroupHash().ShortS(), time.Since(begin).String())
		}
	}()

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
	if ok, err := p.groupManager.onMessageCreateGroupRaw(msg); ok {
		signMsg := &model.ConsensusCreateGroupSignMessage{
			Launcher: msg.SI.SignMember,
			GHash:    gh.Hash,
		}
		ski := p.getInGroupSeckeyInfo(parentGid)
		if signMsg.GenSign(ski, signMsg) {
			tlog.log("SendCreateGroupSignMessage id=%v", p.getPrefix())
			blog.debug("OMCGR SendCreateGroupSignMessage... ")
			p.NetServer.SendCreateGroupSignMessage(signMsg, parentGid)
		} else {
			blog.debug("SendCreateGroupSignMessage sign fail, ski=%v, %v", ski.ID.ShortS(), ski.SK.ShortS())
		}

	} else {
		tlog.log("groupManager.onMessageCreateGroupRaw fail, err:%v", err.Error())
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
	if ok, err := p.groupManager.onMessageCreateGroupSign(msg); ok {
		gpk := ctx.parentInfo.GroupPK
		if !groupsig.VerifySig(gpk, msg.SI.DataHash.Bytes(), ctx.gInfo.GI.Signature) {
			blog.log("Proc(%v) verify group sign fail", p.getPrefix())
			return
		}
		initMsg := &model.ConsensusGroupRawMessage{
			GInfo: *ctx.gInfo,
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

// OnMessageGroupInit is group initialization related message
// Group-initiated related messages are checked with the miner ID and public key (and group-independent)
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

	// Non-group members do not follow the follow-up process
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
	cands, ok, err := p.groupManager.checkGroupInfo(&msg.GInfo)
	if !ok {
		blog.debug("group header illegal, err=%v", err)
		return
	}
	candidates = cands

	tlog.logStart("%v", "")

	groupContext = p.joiningGroups.ConfirmGroupFromRaw(msg, candidates, p.mi)
	if groupContext == nil {
		panic("Processor::OMGI failed, ConfirmGroupFromRaw return nil.")
	}

	// Establish a group network in advance
	p.NetServer.BuildGroupNet(gHash.Hex(), msg.GInfo.Mems)

	gs := groupContext.GetGroupStatus()
	blog.debug("joining group(%v) status=%v.", gHash.ShortS(), gs)
	if groupContext.StatusTransfrom(GisInit, GisSendSharePiece) {
		desc = "send sharepiece"

		// Generate secret sharing
		shares := groupContext.GenSharePieces()

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
	}

	return
}

func (p *Processor) handleSharePieceMessage(blog *bizLog, gHash common.Hash, share *model.SharePiece, si *model.SignData, response bool) (recover bool, err error) {
	blog.log("gHash=%v, sender=%v, response=%v", gHash.ShortS(), si.GetID().ShortS(), response)
	defer func() {
		blog.log("recover %v, err %v", recover, err)
	}()

	gc := p.joiningGroups.GetGroup(gHash)
	if gc == nil {
		err = fmt.Errorf("failed, receive SHAREPIECE msg but gc=nil.gHash=%v", gHash.Hex())
		return
	}
	if gc.gInfo.GroupHash() != gHash {
		err = fmt.Errorf("failed, gisHash diff")
		return
	}

	pk := GetMinerPK(si.GetID())
	if pk == nil {
		err = fmt.Errorf("miner pk is nil, id=%v", si.GetID().ShortS())
		return
	}
	if !si.VerifySign(*pk) {
		err = fmt.Errorf("miner sign verify fail")
		return
	}

	gh := gc.gInfo.GI.GHeader

	topHeight := p.MainChain.QueryTopBlock().Height

	if !response && gc.gInfo.GI.ReadyTimeout(topHeight) {
		err = fmt.Errorf("ready timeout, readyHeight=%v, now=%v", gh.ReadyHeight, topHeight)
		return
	}

	result := gc.PieceMessage(si.GetID(), share)
	waitPieceIds := make([]string, 0)
	for _, mem := range gc.gInfo.Mems {
		if !gc.node.hasPiece(mem) {
			waitPieceIds = append(waitPieceIds, mem.ShortS())
			if len(waitPieceIds) >= 10 {
				break
			}
		}
	}

	mtype := "OMSP"
	if response {
		mtype = "OMSPResponse"
	}
	tlog := newHashTraceLog(mtype, gHash, si.GetID())
	tlog.log("收到piece数 %v, 收齐分片%v, 缺少%v等", gc.node.groupInitPool.GetSize(), result == 1, waitPieceIds)

	// Already signed private key, group public key, group id
	if result == 1 {
		recover = true
		groupLogger.Infof("OMSP收齐分片: gHash=%v, elapsed=%v.", gHash.ShortS(), time.Since(gc.createTime).String())
		jg := gc.GetGroupInfo()
		p.joinGroup(jg)
		// At this time, there is no signature public key of all group members.
		if jg.GroupPK.IsValid() && jg.SignKey.IsValid() {
			ski := model.NewSecKeyInfo(p.mi.GetMinerID(), jg.SignKey)
			// 1. Send your own signature public key first
			if gc.StatusTransfrom(GisSendSharePiece, GisSendSignPk) {
				msg := &model.ConsensusSignPubKeyMessage{
					GroupID: jg.GroupID,
					SignPK:  *groupsig.NewPubkeyFromSeckey(jg.SignKey),
					GHash:   gHash,
					MemCnt:  int32(gc.gInfo.MemberSize()),
				}
				if !msg.SignPK.IsValid() {
					panic("signPK is InValid")
				}
				if msg.GenSign(ski, msg) {
					tlog.log("SendSignPubKey %v", p.getPrefix())
					p.NetServer.SendSignPubKey(msg)
				} else {
					err = fmt.Errorf("genSign fail, id=%v, sk=%v", ski.ID.ShortS(), ski.SK.ShortS())
					return
				}
			}
			// 2. Resend the initialization complete message
			// The response to the sharepiece request does not need to send a GroupInitDone message
			// because the group has already been initialized.
			if !response && gc.StatusTransfrom(GisSendSignPk, GisSendInited) {
				msg := &model.ConsensusGroupInitedMessage{
					GHash:        gHash,
					GroupPK:      jg.GroupPK,
					GroupID:      jg.GroupID,
					CreateHeight: gh.CreateHeight,
					ParentSign:   gc.gInfo.GI.Signature,
					MemCnt:       int32(gc.gInfo.MemberSize()),
					MemMask:      gc.generateMemberMask(),
				}
				ski := model.NewSecKeyInfo(p.mi.GetMinerID(), p.mi.GetDefaultSecKey())

				if msg.GenSign(ski, msg) {
					tlog.log("BroadcastGroupInfo %v", jg.GroupID.ShortS())
					p.NetServer.BroadcastGroupInfo(msg)
				} else {
					err = fmt.Errorf("genSign fail, id=%v, sk=%v", ski.ID.ShortS(), ski.SK.ShortS())
					return
				}
			}
		} else {
			err = fmt.Errorf("Processor::%v failed, aggr key error", mtype)
			return
		}
	}
	return
}

// OnMessageSharePiece received a secret sharing clip message sent to me by members of the group
func (p *Processor) OnMessageSharePiece(spm *model.ConsensusSharePieceMessage) {
	blog := newBizLog("OMSP")

	p.handleSharePieceMessage(blog, spm.GHash, &spm.Share, &spm.SI, false)
	return
}

// OnMessageSignPK received the signature of the group member sent to me by the member of the group
func (p *Processor) OnMessageSignPK(spkm *model.ConsensusSignPubKeyMessage) {
	blog := newBizLog("OMSPK")
	tlog := newHashTraceLog("OMSPK", spkm.GHash, spkm.SI.GetID())

	blog.log("proc(%v) begin , sender=%v, gHash=%v, gid=%v...", p.getPrefix(), spkm.SI.GetID().ShortS(), spkm.GHash.ShortS(), spkm.GroupID.ShortS())

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

	return
}

func (p *Processor) OnMessageSignPKReq(msg *model.ConsensusSignPubkeyReqMessage) {
	blog := newBizLog("OMSPKR")
	sender := msg.SI.GetID()
	var err error
	defer func() {
		blog.log("sender=%v, gid=%v, result=%v", sender.ShortS(), msg.GroupID.ShortS(), err)
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

	resp := &model.ConsensusSignPubKeyMessage{
		GHash:   jg.gHash,
		GroupID: msg.GroupID,
		SignPK:  *groupsig.NewPubkeyFromSeckey(jg.SignKey),
	}
	ski := model.NewSecKeyInfo(p.GetMinerID(), jg.SignKey)
	if resp.GenSign(ski, resp) {
		blog.log("answer signPKReq Message, receiver %v, gid %v", sender.ShortS(), msg.GroupID.ShortS())
		p.NetServer.AnswerSignPkMessage(resp, sender)
	} else {
		err = fmt.Errorf("gen Sign fail, ski=%v,%v", ski.ID.ShortS(), ski.SK.GetHexString())
	}
}

func (p *Processor) acceptGroup(staticGroup *StaticGroupInfo) {
	add := p.globalGroups.AddStaticGroup(staticGroup)
	blog := newBizLog("acceptGroup")
	blog.debug("Add to Global static groups, result=%v, groups=%v.", add, p.globalGroups.GetGroupSize())
	if staticGroup.MemExist(p.GetMinerID()) {
		p.prepareForCast(staticGroup)
	}
}

// OnMessageGroupInited is a network-wide node processing function(Adjust to parent group
// node handler),The entire network node receives a group of initialized completion messages
// (the same message is received when 51% of the members of the group are received in a time window)
//
// The final version was modified to the parent node for verification (51%) and the winding
func (p *Processor) OnMessageGroupInited(msg *model.ConsensusGroupInitedMessage) {
	blog := newBizLog("OMGIED")
	gHash := msg.GHash

	blog.log("proc(%v) begin, sender=%v, gHash=%v, gid=%v, gpk=%v...", p.getPrefix(),
		msg.SI.GetID().ShortS(), gHash.ShortS(), msg.GroupID.ShortS(), msg.GroupPK.ShortS())
	tlog := newHashTraceLog("OMGIED", gHash, msg.SI.GetID())

	if msg.SI.DataHash != msg.GenHash() {
		panic("grm gis hash diff")
	}

	// At this point, the group is already connected by synchronization, but the logic behind it still
	// needs to be executed, otherwise the status of the group data is faulty.
	g := p.GroupChain.GetGroupByID(msg.GroupID.Serialize())
	if g != nil {
		blog.log("group already onchain")
		p.globalGroups.removeInitedGroup(gHash)
		p.joiningGroups.Clean(gHash)
		return
	}

	pk := GetMinerPK(msg.SI.GetID())
	if !msg.VerifySign(*pk) {
		blog.log("verify sign fail, id=%v, pk=%v, sign=%v", msg.SI.GetID().ShortS(), pk.GetHexString(), msg.SI.DataSign.GetHexString())
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
			blog.log("groupHeader hash error, expect %v, receive %v", gInfo.GroupHash().Hex(), msg.GHash.Hex())
			return
		}
		gInfo.GI.Signature = msg.ParentSign
		initedGroup = createInitedGroup(gInfo)
		blog.log("add inited group")
	}
	if initedGroup.gInfo.GI.ReadyTimeout(p.MainChain.Height()) {
		blog.log("group ready timeout, gid=%v", msg.GroupID.ShortS())
		return
	}

	parentID := initedGroup.gInfo.GI.ParentID()
	parentGroup := p.GetGroup(parentID)

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
		if !initedGroup.hasReceived(mem) {
			waitIds = append(waitIds, mem.ShortS())
			if len(waitIds) >= 10 {
				break
			}
		}
	}

	tlog.log("ret:%v,收到消息数量 %v, 需要消息数 %v, 缺少%v等", result, initedGroup.receiveSize(), initedGroup.threshold, waitIds)

	switch result {
	case InitSuccess: // Receive the same message in the group >= threshold, can add on chain
		staticGroup := newSGIFromStaticGroupSummary(msg.GroupID, msg.GroupPK, initedGroup)
		gh := staticGroup.getGroupHeader()
		blog.debug("SUCCESS accept a new group, gHash=%v, gid=%v, workHeight=%v, dismissHeight=%v.", gHash.ShortS(), msg.GroupID.ShortS(), gh.WorkHeight, gh.DismissHeight)

		p.groupManager.addGroupOnChain(staticGroup)
		p.globalGroups.removeInitedGroup(gHash)
		p.joiningGroups.Clean(gHash)

	case InitFail: // The group is initialized abnormally and cannot be recovered
		tlog.log("初始化失败")
		p.globalGroups.removeInitedGroup(gHash)
	}
	return
}

func (p *Processor) OnMessageSharePieceReq(msg *model.ReqSharePieceMessage) {
	blog := newBizLog("OMSPR")
	blog.log("gHash=%v, sender=%v", msg.GHash.ShortS(), msg.SI.GetID().ShortS())

	pk := GetMinerPK(msg.SI.GetID())
	if pk == nil || !msg.VerifySign(*pk) {
		blog.log("verify sign fail")
		return
	}
	gc := p.joiningGroups.GetGroup(msg.GHash)
	if gc == nil {
		blog.log("gc is nil")
		return
	}
	if gc.sharePieceMap == nil {
		blog.log("sharePiece map is nil")
		return
	}
	piece := gc.sharePieceMap[msg.SI.GetID().GetHexString()]

	pieceMsg := &model.ResponseSharePieceMessage{
		GHash: msg.GHash,
		Share: piece,
	}
	if pieceMsg.GenSign(p.getDefaultSeckeyInfo(), pieceMsg) {
		blog.log("response share piece to %v, gHash=%v, share=%v", msg.SI.GetID().ShortS(), msg.GHash.ShortS(), piece.Share.ShortS())
		p.NetServer.ResponseSharePiece(pieceMsg, msg.SI.GetID())
	}
}

func (p *Processor) OnMessageSharePieceResponse(msg *model.ResponseSharePieceMessage) {
	blog := newBizLog("OMSPRP")

	p.handleSharePieceMessage(blog, msg.GHash, &msg.Share, &msg.SI, true)
	return
}
