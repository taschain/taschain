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
	"common"
	"consensus/groupsig"

	"fmt"
	"time"
	"middleware/types"
	"consensus/model"
	"middleware/statistics"
	"bytes"
)

func (p *Processor) genCastGroupSummary(bh *types.BlockHeader) *model.CastGroupSummary {
	var gid groupsig.ID
	if err := gid.Deserialize(bh.GroupId); err != nil {
		return nil
	}
	var castor groupsig.ID
	if err := castor.Deserialize(bh.Castor); err != nil {
		return nil
	}
	cgs := &model.CastGroupSummary{
		PreHash:     bh.Hash,
		PreTime:     bh.PreTime,
		BlockHeight: bh.Height,
		GroupID:     gid,
		Castor:      castor,
	}
	cgs.CastorPos = p.getMinerPos(cgs.GroupID, cgs.Castor)
	return cgs
}

func (p *Processor) thresholdPieceVerify(mtype string, sender string, gid groupsig.ID, vctx *VerifyContext, slot *SlotContext, traceLog *msgTraceLog)  {
	blog := newBizLog("thresholdPieceVerify")
	bh := &slot.BH
	if vctx.castSuccess() {
		blog.log("already cast success, height=%v", bh.Height)
		return
	}
	gpk := p.getGroupPubKey(gid)

	if len(bh.Signature) == 0 {
		blog.log("bh sign is empty! hash=%v", bh.Hash.ShortS())
	}

	if !slot.VerifyGroupSigns(gpk, vctx.prevBH.Random) { //组签名验证通过
		blog.log("%v group pub key local check failed, gpk=%v, hash in slot=%v, hash in bh=%v status=%v.", mtype,
			gpk.ShortS(), slot.BH.Hash.ShortS(), bh.Hash.ShortS(), slot.GetSlotStatus())
		return
	}

	if slot.IsVerified() {
		p.SuccessNewBlock(bh, vctx, slot, gid) //上链和组外广播
		//blog.log("%v remove verifycontext from bccontext! remain size=%v", mtype, len(bc.verifyContexts))
	}

}

func (p *Processor) normalPieceVerify(mtype string, sender string, gid groupsig.ID, vctx *VerifyContext, slot *SlotContext, traceLog *msgTraceLog)  {
	bh := &slot.BH
	castor := groupsig.DeserializeId(bh.Castor)
	if slot.StatusTransform(SS_WAITING, SS_SIGNED) && !castor.IsEqual(p.GetMinerID()) {
		skey := p.getSignKey(gid)
		var cvm model.ConsensusVerifyMessage
		cvm.BH = *bh
		//cvm.GroupID = gId
		cvm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), skey), &cvm)
		cvm.GenRandomSign(skey, vctx.prevBH.Random)
		newBizLog("normalPieceVerify").log("call network service SendVerifiedCast...")
		traceLog.log("SendVerifiedCast")
		p.NetServer.SendVerifiedCast(&cvm)
	}
}

func (p *Processor) doVerify(mtype string, msg *model.ConsensusBlockMessageBase, cgs *model.CastGroupSummary, traceLog *msgTraceLog, blog *bizLog) {
	bh := &msg.BH
	si := &msg.SI

	sender := si.SignMember.ShortS()
	result := ""
	defer func() {
		traceLog.log("height=%v, preHash=%v, doVerify result %v", bh.Height, bh.PreHash.ShortS(), result)
	}()

	if cgs == nil {
		cgs = p.genCastGroupSummary(bh)
		if cgs == nil {
			return
		}
	}

	gid := cgs.GroupID

	preBH := p.getBlockHeaderByHash(bh.PreHash)
	if preBH == nil {
		p.addFutureVerifyMsg(msg)
		result = "父块未到达"
		return
	}

	if !p.verifyCastSign(cgs, msg, preBH) {
		blog.log("verify failed!")
		return
	}

	if !p.isCastLegal(bh, preBH) {
		result = "非法的铸块组"
		blog.log("not the casting group!bh=%v, preBH=%v", p.blockPreview(bh), p.blockPreview(preBH))
		return
	}

	bc := p.GetBlockContext(gid)
	if bc == nil {
		result = "未获取到blockcontext"
		blog.log("[ERROR]blockcontext is nil!, gid=" + gid.ShortS())
		return
	}
	if bc.IsHeightCasted(bh.Height) {
		result = "该高度已铸过"
		blog.log("%v, height=%v", result, bh.Height)
		return
	}

	vctx := bc.GetOrNewVerifyContext(bh, preBH)
	if vctx == nil {
		result = "获取vctx为空，可能preBH已经被删除"
		blog.log("%v, preBH=%v", result, p.blockPreview(preBH))
		return
	}

	if vctx.castSuccess() {
		result = "已出块"
		blog.log(result)
		return
	}
	if vctx.castExpire() {
		vctx.markTimeout()
		result = "已超时" + vctx.expireTime.String()
		blog.log(result)
		return
	}

	blog.log("%v start UserVerified, height=%v, hash=%v", mtype, bh.Height, bh.Hash.ShortS())
	verifyResult := vctx.UserVerified(bh, si, cgs)
	blog.log("proc(%v) UserVerified height=%v, hash=%v, result=%v.", p.getPrefix(), bh.Height, bh.Hash.ShortS(), CBMR_RESULT_DESC(verifyResult))
	slot := vctx.GetSlotByHash(bh.Hash)
	if slot == nil {
		result = "找不到合适的验证槽, 放弃验证"
		return
	}

	result = fmt.Sprintf("%v, %v, %v", CBMR_RESULT_DESC(verifyResult), slot.gSignGenerator.Brief(), slot.TransBrief())

	switch verifyResult {
	case CBMR_THRESHOLD_SUCCESS:
		if !slot.HasTransLost() {
			p.thresholdPieceVerify(mtype, sender, gid, vctx, slot, traceLog)
		}

	case CBMR_PIECE_NORMAL:
		p.normalPieceVerify(mtype, sender, gid, vctx, slot, traceLog)

	case CBMR_PIECE_LOSINGTRANS: //交易缺失
	}
}

func (p *Processor) verifyCastMessage(mtype string, msg *model.ConsensusBlockMessageBase) {
	bh := &msg.BH
	si := &msg.SI

	blog := newBizLog(mtype)
	traceLog := newBlockTraceLog(mtype, bh.Hash, si.GetID())
	traceLog.logStart("height=%v", bh.Height)

	defer func() {
		traceLog.logEnd("height=%v", bh.Height)
	}()

	cgs := p.genCastGroupSummary(bh)
	if cgs == nil {
		blog.log("[ERROR]gen castGroupSummary fail!")
		return
	}
	if !p.IsMinerGroup(cgs.GroupID) { //检测当前节点是否在该铸块组
		return
	}
	blog.log("proc(%v) begin group=%v, sender=%v, height=%v, hash=%v, castor=%v...", p.getPrefix(), cgs.GroupID.ShortS(), si.GetID().ShortS(), bh.Height, bh.Hash.ShortS(), cgs.Castor.ShortS())

	//如果是自己发的, 不处理
	if p.GetMinerID().IsEqual(si.SignMember) {
		return
	}

	if p.blockOnChain(bh) {
		//blog.log("%v receive block already onchain! , height = %v", mtype, bh.Height)
		return
	}

	outputBlockHeaderAndSign(mtype, bh, si)

	if !p.minerCanProposalAt(si.SignMember, bh.Height) {
		blog.log("miner cannot propose at the height, id=%v, height=%v", si.SignMember.ShortS(), bh.Height)
		return
	}

	p.doVerify(mtype, msg, cgs, traceLog, blog)

	return
}

//收到组内成员的出块消息，出块人（KING）用组分片密钥进行了签名
//有可能没有收到OnMessageCurrent就提前接收了该消息（网络时序问题）
func (p *Processor) OnMessageCast(ccm *model.ConsensusCastMessage) {
	statistics.AddBlockLog(common.BootId,statistics.RcvCast,ccm.BH.Height,ccm.BH.ProveValue.Uint64(),-1,-1,
		time.Now().UnixNano(),"","",common.InstanceIndex,ccm.BH.CurTime.UnixNano())
	p.verifyCastMessage("OMC", &ccm.ConsensusBlockMessageBase)
}

//收到组内成员的出块验证通过消息（组内成员消息）
func (p *Processor) OnMessageVerify(cvm *model.ConsensusVerifyMessage) {
	statistics.AddBlockLog(common.BootId,statistics.RcvVerified,cvm.BH.Height,cvm.BH.ProveValue.Uint64(),-1,-1,
		time.Now().UnixNano(),"","",common.InstanceIndex,cvm.BH.CurTime.UnixNano())
	p.verifyCastMessage("OMV", &cvm.ConsensusBlockMessageBase)
}

func (p *Processor) receiveBlock(block *types.Block, preBH *types.BlockHeader) bool {
	if p.isCastLegal(block.Header, preBH) { //铸块组合法
		result := p.doAddOnChain(block)
		if result == 0 || result == 1 {
			return true
		}
	} else {
		//丢弃该块
		newBizLog("receiveBlock").log("received invalid new block, height = %v.", block.Header.Height)
	}
	return false
}

func (p *Processor) cleanVerifyContext(currentHeight uint64) {
	p.blockContexts.forEach(func(bc *BlockContext) bool {
		bc.CleanVerifyContext(currentHeight)
		return true
	})
}

//收到铸块上链消息(组外矿工节点处理)
func (p *Processor) OnMessageBlock(cbm *model.ConsensusBlockMessage) {
	statistics.AddBlockLog(common.BootId,statistics.RcvNewBlock,cbm.Block.Header.Height,cbm.Block.Header.ProveValue.Uint64(),len(cbm.Block.Transactions),-1,
		time.Now().UnixNano(),"","",common.InstanceIndex,cbm.Block.Header.CurTime.UnixNano())
	bh := cbm.Block.Header
	blog := newBizLog("OMB")
	tlog := newBlockTraceLog("OMB", bh.Hash, groupsig.DeserializeId(bh.Castor))
	tlog.logStart("height=%v, preHash=%v", bh.Height, bh.PreHash.ShortS())
	result := ""
	defer func() {
		tlog.logEnd("height=%v, preHash=%v, result=%v", bh.Height, bh.PreHash.ShortS(), result)
	}()

	if p.getBlockHeaderByHash(cbm.Block.Header.Hash) != nil {
		//blog.log("OMB receive block already on chain! bh=%v", p.blockPreview(cbm.Block.Header))
		result = "已经在链上"
		return
	}
	var gid = groupsig.DeserializeId(cbm.Block.Header.GroupId)

	blog.log("proc(%v) begin OMB, group=%v, height=%v, hash=%v...", p.getPrefix(),
		gid.ShortS(), cbm.Block.Header.Height, bh.Hash.ShortS())

	block := &cbm.Block

	preHeader := p.MainChain.QueryBlockByHash(block.Header.PreHash)
	if preHeader == nil {
		p.addFutureBlockMsg(cbm)
		result = "父块未到达"
		return
	}
	//panic("isBHCastLegal: cannot find pre block header!,ignore block")
	verify := p.verifyGroupSign(cbm, preHeader)
	if !verify {
		result = "组签名未通过"
		blog.log("OMB verifyGroupSign result=%v.", verify)
		return
	}

	ret := p.receiveBlock(block, preHeader)
	if ret {
		result = "上链成功"
	} else {
		result = "上链失败"
	}

	//blog.log("proc(%v) end OMB, group=%v, sender=%v...", p.getPrefix(), GetIDPrefix(cbm.GroupID), GetIDPrefix(cbm.SI.GetID()))
	return
}

//新的交易到达通知（用于处理大臣验证消息时缺失的交易）
func (p *Processor) OnMessageNewTransactions(ths []common.Hash) {
	mtype := "OMNT"
	blog := newBizLog(mtype)

	blog.log("proc(%v) begin %v, trans count=%v...", p.getPrefix(),mtype, len(ths))

	p.blockContexts.forEach(func(bc *BlockContext) bool {
		for _, vctx := range bc.SafeGetVerifyContexts() {
			for _, slot := range vctx.GetSlots() {
				acceptRet := vctx.AcceptTrans(slot, ths)
				tlog := newBlockTraceLog(mtype, slot.BH.Hash, groupsig.ID{})
				switch acceptRet {
				case TRANS_INVALID_SLOT, TRANS_DENY:

				case TRANS_ACCEPT_NOT_FULL:
					blog.log("accept trans bh=%v, ret %v", p.blockPreview(&slot.BH), acceptRet)
					tlog.log("preHash=%v, height=%v, %v,收到 %v, 总交易数 %v, 仍缺失数 %v", slot.BH.PreHash.ShortS(), slot.BH.Height, TRANS_ACCEPT_RESULT_DESC(acceptRet), len(ths), len(slot.BH.Transactions), slot.lostTransSize())

				case TRANS_ACCEPT_FULL_PIECE:
					blog.log("accept trans bh=%v, ret %v", p.blockPreview(&slot.BH), acceptRet)
					tlog.log("preHash=%v, height=%v %v, 当前分片数%v", slot.BH.PreHash.ShortS(), slot.BH.Height, TRANS_ACCEPT_RESULT_DESC(acceptRet), slot.MessageSize())

				case TRANS_ACCEPT_FULL_THRESHOLD:
					blog.log("accept trans bh=%v, ret %v", p.blockPreview(&slot.BH), acceptRet)
					tlog.log("preHash=%v, height=%v, %v", slot.BH.PreHash.ShortS(), slot.BH.Height, TRANS_ACCEPT_RESULT_DESC(acceptRet))
					if len(slot.BH.Signature) == 0 {
						blog.log("slot bh sign is empty hash=%v", slot.BH.Hash.ShortS())
					}
					p.thresholdPieceVerify(mtype, p.getPrefix(), bc.MinerID.Gid, vctx, slot, tlog)
				}

			}
		}
		return true
	})

	return
}

///////////////////////////////////////////////////////////////////////////////
//组初始化相关消息
//组初始化的相关消息都用（和组无关的）矿工ID和公钥验签

func (p *Processor) OnMessageGroupInit(grm *model.ConsensusGroupRawMessage) {
	blog := newBizLog("OMGI")
	blog.log("proc(%v) begin, sender=%v, dummy_gid=%v...", p.getPrefix(), grm.SI.GetID().ShortS(), grm.GI.DummyID.ShortS())
	tlog := newGroupTraceLog("OMGI", grm.GI.DummyID, grm.SI.GetID())
	
	if !grm.GI.CheckMemberHash(grm.MEMS) {
		panic("grm member hash diff!")
	}
	if grm.SI.DataHash != grm.GI.GenHash() {
		panic("grm gis hash diff")
	}
	parentGroup := p.getGroup(grm.GI.ParentID)
	if !parentGroup.CastQualified(grm.GI.TopHeight) {
		blog.log("parent group has no qualify to cast group. gid=%v, height=%v", parentGroup.GroupID.ShortS(), grm.GI.TopHeight)
		return
	}
	topHeight := p.MainChain.QueryTopBlock().Height
	if grm.GI.ReadyTimeout(topHeight) {
		blog.log("OMGI ready timeout, readyHeight=%v, now=%v", grm.GI.GetReadyHeight, topHeight)
		return
	}
	gpk := parentGroup.GroupPK
	if !groupsig.VerifySig(gpk, grm.SI.DataHash.Bytes(), grm.GI.Signature) {
		blog.log("OMGI verify parent groupsig fail!")
		return
	}


	if p.globalGroups.AddInitingGroup(CreateInitingGroup(grm)) {
		//to do : 从链上检查消息发起人（父亲组成员）是否有权限发该消息（鸠兹）
		//dummy 组写入组链 add by 小熊
		//staticGroupInfo := NewDummySGIFromGroupRawMessage(grm)
		//p.groupManager.AddGroupOnChain(staticGroupInfo, true)
	}

	tlog.logStart("%v", "")

	//非组内成员不走后续流程
	if !grm.MemberExist(p.GetMinerID()) {
		return
	}
	//p.globalGroups.AddDummyGroup(sgi)

	groupContext := p.joiningGroups.ConfirmGroupFromRaw(grm, p.mi)
	if groupContext == nil {
		panic("Processor::OMGI failed, ConfirmGroupFromRaw return nil.")
	}

	//提前建立组网络
	members := make([]groupsig.ID,0)
	for _,m := range grm.MEMS{
		members = append(members,m.ID)
	}
	p.NetServer.BuildGroupNet(grm.GI.DummyID, members)

	gs := groupContext.GetGroupStatus()
	blog.log("joining group(%v) status=%v.", grm.GI.DummyID.ShortS(), gs)
	if gs == GIS_RAW {
		//blog.log("begin GenSharePieces in OMGI...")
		shares := groupContext.GenSharePieces() //生成秘密分享
		//blog.log("proc(%v) end GenSharePieces in OMGI, piece size=%v.", p.getPrefix(), len(shares))

		spm := &model.ConsensusSharePieceMessage{
			GISHash: grm.GI.GenHash(),
			DummyID: grm.GI.DummyID,
		}
		ski := model.NewSecKeyInfo(p.mi.GetMinerID(), p.mi.GetDefaultSecKey())
		spm.SI.SignMember = p.GetMinerID()

		for id, piece := range shares {
			if id != "0x0" && piece.IsValid() {
				spm.Dest.SetHexString(id)
				spm.Share = piece
				spm.GenSign(ski, spm)
				//blog.log("OMGI spm.GenSign result=%v.", sb)
				blog.log("piece to ID(%v), dummyId=%v, share=%v, pub=%v.", spm.Dest.ShortS(), spm.DummyID.ShortS(), spm.Share.Share.ShortS(), spm.Share.Pub.ShortS())
				tlog.log("sharepiece to %v", spm.Dest.ShortS())
				blog.log("call network service SendKeySharePiece...")
				p.NetServer.SendKeySharePiece(spm)

			} else {
				panic("GenSharePieces data not IsValid.")
			}
		}
		//blog.log("end GenSharePieces.")
	} 

	//blog.log("proc(%v) end OMGI, sender=%v.", p.getPrefix(), GetIDPrefix(grm.SI.GetID()))
	return
}

//收到组内成员发给我的秘密分享片段消息
func (p *Processor) OnMessageSharePiece(spm *model.ConsensusSharePieceMessage) {
	blog := newBizLog("OMSP")
	blog.log("proc(%v)begin Processor::OMSP, sender=%v, dummyId=%v...", p.getPrefix(), spm.SI.GetID().ShortS(), spm.DummyID.ShortS())
	tlog := newGroupTraceLog("OMSP", spm.DummyID, spm.SI.GetID())
	
	if !spm.Dest.IsEqual(p.GetMinerID()) {
		return
	}

	gc := p.joiningGroups.GetGroup(spm.DummyID)
	if gc == nil {
		blog.log("failed, receive SHAREPIECE msg but gc=nil.")
		return
	}
	if gc.gis.GenHash() != spm.GISHash {
		blog.log("failed, gisHash diff.")
		return
	}
	topHeight := p.MainChain.QueryTopBlock().Height
	if gc.gis.ReadyTimeout(topHeight) {
		blog.log("ready timeout, readyHeight=%v, now=%v", gc.gis.GetReadyHeight, topHeight)
		return
	}

	result := gc.PieceMessage(spm)
	blog.log("proc(%v) after gc.PieceMessage, gc result=%v.", p.getPrefix(), result)

	tlog.log("收到piece数 %v", gc.node.groupInitPool.GetSize())

	if result == 1 { //已聚合出签名私钥
		jg := gc.GetGroupInfo()
		//这时还没有所有组成员的签名公钥
		if jg.GroupPK.IsValid() && jg.SignKey.IsValid() {
			{
				ski := model.NewSecKeyInfo(p.mi.GetMinerID(), p.mi.GetDefaultSecKey())
				msg := &model.ConsensusSignPubKeyMessage{
					GISHash: spm.GISHash,
					DummyID: spm.DummyID,
					SignPK:  *groupsig.NewPubkeyFromSeckey(jg.SignKey),
				}

				//对GISHash做自己的签名
				msg.GenGISSign(jg.SignKey)
				if !msg.VerifyGISSign(msg.SignPK) {
					panic("verify GISSign with group member sign pub key failed.")
				}

				msg.GenSign(ski, msg)
				//todo : 组内广播签名公钥
				blog.log("send sign pub key to group members, spk=%v...", msg.SignPK.ShortS())
				tlog.log("SendSignPubKey %v", p.getPrefix())

				blog.log("call network service SendSignPubKey...")
				p.NetServer.SendSignPubKey(msg)
				
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
	tlog := newGroupTraceLog("OMSPK", spkm.DummyID, spkm.SI.GetID())
	
	blog.log("proc(%v) begin , sender=%v, dummy_gid=%v...", p.getPrefix(), spkm.SI.GetID().ShortS(), spkm.DummyID.ShortS())

	
	gc := p.joiningGroups.GetGroup(spkm.DummyID)
	if gc == nil {
		blog.log("failed, local node not found joining group with dummy id=%v.", spkm.DummyID.ShortS())
		return
	}
	if gc.gis.GenHash() != spkm.GISHash {
		blog.log("failed, gisHash diff.")
		return
	}
	if !spkm.VerifyGISSign(spkm.SignPK) {
		panic("OMSP verify GISSign with sign pub key failed.")
	}
	topHeight := p.MainChain.QueryTopBlock().Height
	if gc.gis.ReadyTimeout(topHeight) {
		blog.log("ready timeout, readyHeight=%v, now=%v", gc.gis.GetReadyHeight, topHeight)
		return
	}

	//blog.log("before SignPKMessage already exist mem sign pks=%v.", len(gc.node.memberPubKeys))
	result := gc.SignPKMessage(spkm)
	blog.log("after SignPKMessage exist mem sign pks=%v, result=%v.", len(gc.node.memberPubKeys), result)

	tlog.log("收到签名公钥数 %v", len(gc.node.memberPubKeys))

	if result == 1 { //收到所有组成员的签名公钥
		jg := gc.GetGroupInfo()

		if jg.GroupID.IsValid() && jg.SignKey.IsValid() {
			p.joinGroup(jg, true)
			{
				var msg model.ConsensusGroupInitedMessage
				ski := model.NewSecKeyInfo(p.mi.GetMinerID(), p.mi.GetDefaultSecKey())
				msg.GI.GIS = gc.gis
				msg.GI.GroupID = jg.GroupID
				msg.GI.GroupPK = jg.GroupPK

				msg.GenSign(ski, &msg)

				tlog.log("BroadcastGroupInfo %v", jg.GroupID.ShortS())

				if !PROC_TEST_MODE {

					blog.log("call network service BroadcastGroupInfo...")
					p.NetServer.BroadcastGroupInfo(&msg)
				} else {
					blog.log("test mode, call OnMessageGroupInited direct...")
					for _, proc := range p.GroupProcs {
						proc.OnMessageGroupInited(&msg)
					}
				}
			}
		} else {
			panic("Processor::OnMessageSharePiece failed, aggr key error.")
		}
		p.joiningGroups.RemoveGroup(spkm.DummyID)
	}

	//blog.log("proc(%v) end OMSPK, sender=%v, dummy gid=%v.", p.getPrefix(), GetIDPrefix(spkm.SI.GetID()), GetIDPrefix(spkm.DummyID))
	return
}

func (p *Processor) acceptGroup(staticGroup *StaticGroupInfo) {
	add := p.globalGroups.AddStaticGroup(staticGroup)
	blog := newBizLog("acceptGroup")
	blog.log("Add to Global static groups, result=%v, groups=%v.", add, p.globalGroups.GetGroupSize())
	if staticGroup.MemExist(p.GetMinerID()) {
		jg := p.belongGroups.getJoinedGroup(staticGroup.GroupID)
		if jg != nil {
			p.prepareForCast(staticGroup)
		} else {
			blog.log("[ERROR]cannot find joined group info, gid=%v", staticGroup.GroupID.ShortS())
		}
	}
}

//全网节点收到某组已初始化完成消息（在一个时间窗口内收到该组51%成员的消息相同，才确认上链）
//最终版本修改为父亲节点进行验证（51%）和上链
//全网节点处理函数->to do : 调整为父亲组节点处理函数
func (p *Processor) OnMessageGroupInited(gim *model.ConsensusGroupInitedMessage) {
	blog := newBizLog("OMGIED")
	blog.log("proc(%v) begin, sender=%v, dummy_gid=%v, gid=%v, gpk=%v...", p.getPrefix(),
		gim.SI.GetID().ShortS(), gim.GI.GIS.DummyID.ShortS(), gim.GI.GroupID.ShortS(), gim.GI.GroupPK.ShortS())
	tlog := newGroupTraceLog("OMGIED", gim.GI.GroupID, gim.SI.GetID())

	dummyId := gim.GI.GIS.DummyID

	if gim.SI.DataHash != gim.GI.GenHash() {
		panic("grm gis hash diff")
	}
	parentGroup := p.getGroup(gim.GI.GIS.ParentID)
	if !parentGroup.CastQualified(gim.GI.GIS.TopHeight) {
		blog.log("parent group has no qualify to cast group. gid=%v, height=%v", parentGroup.GroupID.ShortS(), gim.GI.GIS.TopHeight)
		return
	}
	gpk := parentGroup.GroupPK
	if !groupsig.VerifySig(gpk, gim.GI.GIS.GenHash().Bytes(), gim.GI.GIS.Signature) {
		blog.log("verify parent groupsig fail! dummyId=%v", dummyId.ShortS())
		return
	}
	topHeight := p.MainChain.QueryTopBlock().Height
	initingGroup := p.globalGroups.GetInitingGroup(dummyId)
	if initingGroup == nil {
		blog.log("initingGroup not found!dummyId=%v", dummyId.ShortS())
		return
	}
	if initingGroup.gis.ReadyTimeout(topHeight) {
		blog.log("ready timeout, readyHeight=%v, now=%v", initingGroup.gis.GetReadyHeight, topHeight)
		return
	}
	if !initingGroup.MemberExist(gim.SI.SignMember) {
		return
	}

	if initingGroup.gis.GenHash() != gim.GI.GIS.GenHash() {
		blog.log("gisHash diff from initingGroup, dummyId=%v", dummyId.ShortS())
		return
	}
	if !gim.GI.GIS.CheckMemberHash(initingGroup.mems) {
		panic("gim member hash diff!")
	}

	var result int32
	if !initingGroup.MemberExist(p.GetMinerID()) {
		result = p.globalGroups.GroupInitedMessage(&gim.GI, gim.SI.SignMember, topHeight)

		blog.log("proc(%v) globalGroups.GroupInitedMessage result=%v.", p.getPrefix(), result)
		tlog.log("收到消息数量 %v", initingGroup.receiveSize())
	} else {
		result = INIT_SUCCESS
	}

	switch result {
	case INIT_SUCCESS: //收到组内相同消息>=阈值，可上链
		staticGroup := NewSGIFromStaticGroupSummary(&gim.GI, initingGroup)
		blog.log("SUCCESS accept a new group, gid=%v, gpk=%v, beginHeight=%v, dismissHeight=%v.", gim.GI.GroupID.ShortS(), gim.GI.GroupPK.ShortS(), staticGroup.BeginHeight, staticGroup.DismissHeight)

		//p.acceptGroup(staticGroup)
		p.groupManager.AddGroupOnChain(staticGroup, false)

		p.globalGroups.removeInitingGroup(initingGroup.gis.DummyID)

	case INIT_FAIL: //该组初始化异常，且无法恢复
		tlog.log("初始化失败")
		p.globalGroups.removeInitingGroup(initingGroup.gis.DummyID)

	case INITING:
		//继续等待下一包数据
	}
	//blog.log("proc(%v) end OMGIED, sender=%v...", p.getPrefix(), GetIDPrefix(gim.SI.GetID()))
	return
}


func (p *Processor) OnMessageCreateGroupRaw(msg *model.ConsensusCreateGroupRawMessage)  {
	blog := newBizLog("OMCRG")
	blog.log("Proc(%v) begin, dummyId=%v sender=%v", p.getPrefix(), msg.GI.DummyID.ShortS(), msg.SI.SignMember.ShortS())

	if p.GetMinerID().IsEqual(msg.SI.SignMember) {
		return
	}
	gpk := p.GetMemberSignPubKey(model.NewGroupMinerID(msg.GI.ParentID, msg.SI.SignMember))
	if !gpk.IsValid() {
		return
	}
	if !msg.SI.VerifySign(gpk) {
		return
	}

	tlog := newGroupTraceLog("OMCGR", msg.GI.DummyID, msg.SI.GetID())
	if p.groupManager.OnMessageCreateGroupRaw(msg) {
		signMsg := &model.ConsensusCreateGroupSignMessage{
			GI: msg.GI,
			Launcher: msg.SI.SignMember,
		}
		signMsg.GenSign(model.NewSecKeyInfo(p.GetMinerID(), p.getSignKey(msg.GI.ParentID)), signMsg)
		tlog.log("SendCreateGroupSignMessage id=%v", p.getPrefix())
		blog.log("OMCGR SendCreateGroupSignMessage... ")
		p.NetServer.SendCreateGroupSignMessage(signMsg)
	} else {
		tlog.log("groupManager.OnMessageCreateGroupRaw fail")

	}
}

func (p *Processor) OnMessageCreateGroupSign(msg *model.ConsensusCreateGroupSignMessage)  {
	blog := newBizLog("OMCRS")
	blog.log("Proc(%v) begin, dummyId=%v, sender=%v", p.getPrefix(), msg.GI.DummyID.ShortS(), msg.SI.SignMember.ShortS())
	if p.GetMinerID().IsEqual(msg.SI.SignMember) {
		return
	}
	mpk := p.GetMemberSignPubKey(model.NewGroupMinerID(msg.GI.ParentID, msg.SI.SignMember))
	if !mpk.IsValid() {
		return
	}
	if !msg.SI.VerifySign(mpk) {
		return
	}
	if p.groupManager.OnMessageCreateGroupSign(msg) {
		creatingGroup := p.groupManager.creatingGroups.getCreatingGroup(msg.GI.DummyID)
		if creatingGroup == nil {
			blog.log("Proc(%v) creatingGroup not found!dummyId=%v", p.getPrefix(), msg.GI.DummyID.ShortS())
			return
		}
		gpk := p.getGroupPubKey(msg.GI.ParentID)
		if !groupsig.VerifySig(gpk, msg.SI.DataHash.Bytes(), msg.GI.Signature) {
			blog.log("Proc(%v) verify group sign fail", p.getPrefix())
			return
		}
		mems := creatingGroup.pkis

		initMsg := &model.ConsensusGroupRawMessage{
			GI: msg.GI,
			MEMS: mems,
		}

		blog.log("Proc(%v) send group init Message", p.getPrefix())
		initMsg.GenSign(model.NewSecKeyInfo(p.GetMinerID(), p.getMinerInfo().GetDefaultSecKey()), initMsg)

		tlog := newGroupTraceLog("OMCGS", msg.GI.DummyID, msg.SI.GetID())
		tlog.log("SendGroupInitMessage")
		p.NetServer.SendGroupInitMessage(initMsg)

		p.groupManager.removeCreatingGroup(msg.GI.DummyID)
	}
}

func (p *Processor) signCastRewardReq(msg *model.CastRewardTransSignReqMessage, bh *types.BlockHeader, blog *bizLog) (send bool) {
	gid := groupsig.DeserializeId(bh.GroupId)
	group := p.getGroup(gid)
	reward := &msg.Reward
	if group == nil {
		panic("group is nil")
	}
	if !msg.VerifySign(p.GetMemberSignPubKey(model.NewGroupMinerID(gid, msg.SI.GetID()))) {
		return
	}

	if !bytes.Equal(bh.GroupId, reward.GroupId) {
		blog.log("groupID error")
		return
	}
	genBonus, _ := p.MainChain.GetBonusManager().GenerateBonus(reward.TargetIds, bh.Hash, bh.GroupId, model.Param.GetVerifierBonus())
	if genBonus.TxHash != reward.TxHash {
		blog.log("bonus txHash diff")
		return
	}

	bc := p.GetBlockContext(gid)
	if bc == nil {
		blog.log("blockcontext is nil, gid=%v", gid.ShortS())
		return
	}
	vctx := bc.GetVerifyContextByHeight(bh.Height)
	if vctx == nil || vctx.prevBH.Hash != bh.PreHash {
		blog.log("vctx is nil")
		return
	}

	slot := vctx.GetSlotByHash(bh.Hash)
	if slot == nil {
		blog.log("slot is nil")
		return
	}
	if len(msg.Reward.TargetIds) != len(msg.SignedPieces) {
		blog.log("targetId len differ from signedpiece len %v %v", len(msg.Reward.TargetIds), len(msg.SignedPieces))
		return
	}

	gSignGener := model.NewGroupSignGenerator(bc.threshold())

	witnesses := slot.gSignGenerator.GetWitnesses()
	for idx, idIndex := range msg.Reward.TargetIds {
		id := group.GetMemberID(int(idIndex))
		sign := msg.SignedPieces[idx]
		if sig, ok := witnesses[id.GetHexString()]; !ok {	//本地无该id签名的，需要校验签名
			pk := p.GetMemberSignPubKey(model.NewGroupMinerID(gid, id))
			if !groupsig.VerifySig(pk, bh.Hash.Bytes(), sign) {
				blog.log("verify member sign fail, id=%v", id.ShortS())
				return
			}
		} else {	//本地已有该id的签名的，只要判断是否跟本地签名一样即可
			if !sign.IsEqual(sig) {
				blog.log("member sign different id=%v", id.ShortS())
				return
			}
		}
		gSignGener.AddWitness(id, sign)
	}
	if !gSignGener.Recovered() {
		blog.log("recover group sign fail")
		return
	}
	bhSign := groupsig.DeserializeSign(bh.Signature)
	if !gSignGener.GetGroupSign().IsEqual(*bhSign) {
		blog.log("recovered sign differ from bh sign, recover %v, bh %v", gSignGener.GetGroupSign().ShortS(), bhSign.ShortS())
		return
	}

	send = true
	//自己签名
	signMsg := &model.CastRewardTransSignMessage{
		ReqHash: reward.TxHash,
		BlockHash: reward.BlockHash,
		GroupID: gid,
		Launcher: msg.SI.GetID(),
	}
	signMsg.GenSign(model.NewSecKeyInfo(p.GetMinerID(), p.getSignKey(gid)), signMsg)
	p.NetServer.SendCastRewardSign(signMsg)
	blog.log("SendCastRewardSign to %v", msg.SI.GetID().ShortS())
	return
}

func (p *Processor) OnMessageCastRewardSignReq(msg *model.CastRewardTransSignReqMessage) {
	mtype := "OMCRSR"
	blog := newBizLog(mtype)
	reward := &msg.Reward
	tlog := newBlockTraceLog("OMCRSR", reward.BlockHash, msg.SI.GetID())
	tlog.logStart("txHash=%v", reward.TxHash.ShortS())

	send := false
	defer func() {
		tlog.logEnd("%v", send)
	}()

	blog.log("begin, sender=%v, blockHash=%v, txHash=%v", msg.SI.GetID().ShortS(), reward.BlockHash.ShortS(), reward.TxHash.ShortS())
	//此时块不一定在链上
	bh := p.getBlockHeaderByHash(reward.BlockHash)
	if bh == nil {
		blog.log("future reward request receive and cached, hash=%v", reward.BlockHash.ShortS())
		p.futureRewardReqs.addMessage(reward.BlockHash, msg)
		return
	}

	send = p.signCastRewardReq(msg, bh, blog)
}

func (p *Processor) OnMessageCastRewardSign(msg *model.CastRewardTransSignMessage) {
	mtype := "OMCRS"
	blog := newBizLog(mtype)

	blog.log("begin, sender=%v, reqHash=%v", msg.SI.GetID().ShortS(), msg.ReqHash.ShortS())
	tlog := newBlockTraceLog(mtype, msg.BlockHash, msg.SI.GetID())

	tlog.logStart("txHash=%v", msg.ReqHash.ShortS())
	send := "fail"
	defer func() {
		tlog.logEnd("%v", send)
	}()

	bh := p.getBlockHeaderByHash(msg.BlockHash)
	if bh == nil {
		blog.log("block not exist, hash=%v", msg.BlockHash.ShortS())
		return
	}

	gid := groupsig.DeserializeId(bh.GroupId)
	group := p.getGroup(gid)
	if group == nil {
		panic("group is nil")
	}
	if !msg.VerifySign(p.GetMemberSignPubKey(model.NewGroupMinerID(gid, msg.SI.GetID()))) {
		return
	}

	bc := p.GetBlockContext(gid)
	if bc == nil {
		blog.log("blockcontext is nil, gid=%v", gid.ShortS())
		return
	}
	vctx := bc.GetVerifyContextByHeight(bh.Height)
	if vctx == nil || vctx.prevBH.Hash != bh.PreHash {
		blog.log("vctx is nil")
		return
	}

	slot := vctx.GetSlotByHash(bh.Hash)
	if slot == nil {
		blog.log("slot is nil")
		return
	}

	accept, recover := slot.AcceptRewardPiece(&msg.SI)
	blog.log("slot acceptRewardPiece %v %v status %v", accept, recover, slot.GetSlotStatus())
	if accept && recover && slot.StatusTransform(SS_REWARD_REQ, SS_REWARD_SEND) {
		 ok, err := p.MainChain.GetTransactionPool().Add(slot.rewardTrans)
		 send = "success && send"
		 blog.log("add rewardTrans to txPool, txHash=%v, ret=%v %v", slot.rewardTrans.Hash.ShortS(), ok, err)
	} else {
		send = "success && not send"
	}
}