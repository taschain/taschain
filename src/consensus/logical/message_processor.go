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
	"bytes"
	"common"
	"consensus/groupsig"
	"consensus/model"
	"fmt"
	"middleware/statistics"
	"middleware/types"
	"time"
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

func (p *Processor) thresholdPieceVerify(mtype string, sender string, gid groupsig.ID, vctx *VerifyContext, slot *SlotContext, traceLog *msgTraceLog) {
	blog := newBizLog("thresholdPieceVerify")
	bh := slot.BH
	if vctx.castSuccess() {
		blog.debug("already cast success, height=%v", bh.Height)
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
		p.reserveBlock(vctx, slot)
	}

}

func (p *Processor) normalPieceVerify(mtype string, sender string, gid groupsig.ID, proveHash []common.Hash, vctx *VerifyContext, slot *SlotContext, traceLog *msgTraceLog)  {
	bh := slot.BH
	castor := groupsig.DeserializeId(bh.Castor)
	if slot.StatusTransform(SS_WAITING, SS_SIGNED) && !castor.IsEqual(p.GetMinerID()) {
		skey := p.getSignKey(gid)
		var cvm model.ConsensusVerifyMessage
		cvm.BH = *bh
		cvm.ProveHash = proveHash
		//cvm.GroupID = gId
		blog := newBizLog("normalPieceVerify")
		if cvm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), skey), &cvm) {
			cvm.GenRandomSign(skey, vctx.prevBH.Random)
			blog.debug("call network service SendVerifiedCast hash=%v, height=%v", bh.Hash.ShortS(), bh.Height)
			traceLog.log("SendVerifiedCast height=%v, castor=%v", bh.Height, slot.castor.ShortS())
			//验证消息需要给自己也发一份，否则自己的分片中将不包含自己的签名，导致分红没有
			p.NetServer.SendVerifiedCast(&cvm)
		} else {
			blog.log("genSign fail, id=%v, sk=%v %v", p.GetMinerID().ShortS(), skey.ShortS(), p.IsMinerGroup(gid))
		}
	}
}

func (p *Processor) doVerify(mtype string, msg *model.ConsensusBlockMessageBase, traceLog *msgTraceLog, blog *bizLog) (err error) {
	bh := &msg.BH
	si := &msg.SI

	sender := si.SignMember.ShortS()

	gid := groupsig.DeserializeId(bh.GroupId)
	castor := groupsig.DeserializeId(bh.Castor)

	preBH := p.getBlockHeaderByHash(bh.PreHash)
	if preBH == nil {
		p.addFutureVerifyMsg(msg)
		return fmt.Errorf("父块未到达")
	}
	if expireTime, expire := VerifyBHExpire(bh, preBH); expire {
		return fmt.Errorf("cast verify expire, preTime %v, expire %v", preBH.CurTime, expireTime)
	}
	if !p.IsMinerGroup(gid) {
		return fmt.Errorf("%v is not in group %v", p.GetMinerID().ShortS(), gid.ShortS())
	}
	bc := p.GetBlockContext(gid)
	if bc == nil {
		err = fmt.Errorf("未获取到blockcontext, gid=" + gid.ShortS())
		return
	}

	if _, same := bc.IsHeightCasted(bh.Height, bh.PreHash); same {
		err = fmt.Errorf("该高度已铸过 %v", bh.Height)
		return
	}

	//非提案节点消息，即组内验证消息，需要验证随机数签名
	if !castor.IsEqual(si.GetID()) {
		pk := p.GetMemberSignPubKey(model.NewGroupMinerID(gid, si.GetID()))
		if !msg.VerifyRandomSign(pk, preBH.Random) {
			err = fmt.Errorf("random sign verify fail, gid %v, pk %v", gid.ShortS(), pk.ShortS())
			return
		}
	}
	if ok, _, err2 := p.isCastLegal(bh, preBH); !ok {
		err = err2
		return
	}
	//校验提案者是否有全量账本
	sampleHeight := p.sampleBlockHeight(bh.Height, preBH.Random, p.GetMinerID())
	realHeight, existHash := p.getNearestVerifyHashByHeight(sampleHeight)
	if !existHash.IsValid() {
		err = fmt.Errorf("MainChain GetCheckValue error, height=%v, err=%v", sampleHeight, err)
		return
	}
	vHash := msg.ProveHash[p.getMinerPos(gid, p.GetMinerID())]
	if vHash != existHash {
		err = fmt.Errorf("check prove hash fail, sampleHeight=%v, realHeight=%v, receive hash=%v, exist hash=%v", sampleHeight, realHeight, vHash.ShortS(), existHash.ShortS())
		return
	}

	vctx := bc.GetOrNewVerifyContext(bh, preBH)
	if vctx == nil {
		err = fmt.Errorf("获取vctx为空，可能preBH已经被删除")
		return
	}

	if vctx.castSuccess() || vctx.broadCasted() {
		err = fmt.Errorf("已出块")
		return
	}
	if vctx.castExpire() {
		vctx.markTimeout()
		err = fmt.Errorf("已超时" + vctx.expireTime.String())
		return
	}

	blog.debug("%v start UserVerified, height=%v, hash=%v", mtype, bh.Height, bh.Hash.ShortS())
	verifyResult := vctx.UserVerified(bh, si)
	blog.log("proc(%v) UserVerified height=%v, hash=%v, result=%v.", p.getPrefix(), bh.Height, bh.Hash.ShortS(), CBMR_RESULT_DESC(verifyResult))
	slot := vctx.GetSlotByHash(bh.Hash)
	if slot == nil {
		err = fmt.Errorf("找不到合适的验证槽, 放弃验证")
		return
	}

	err = fmt.Errorf("%v, %v, %v", CBMR_RESULT_DESC(verifyResult), slot.gSignGenerator.Brief(), slot.TransBrief())

	switch verifyResult {
	case CBMR_THRESHOLD_SUCCESS:
		if !slot.HasTransLost() {
			p.thresholdPieceVerify(mtype, sender, gid, vctx, slot, traceLog)
		}

	case CBMR_PIECE_NORMAL:
		p.normalPieceVerify(mtype, sender, gid, msg.ProveHash, vctx, slot, traceLog)

	case CBMR_PIECE_LOSINGTRANS: //交易缺失
	}
	return
}

func (p *Processor) verifyCastMessage(mtype string, msg *model.ConsensusBlockMessageBase) {
	bh := &msg.BH
	si := &msg.SI
	blog := newBizLog(mtype)
	traceLog := newHashTraceLog(mtype, bh.Hash, si.GetID())
	castor := groupsig.DeserializeId(bh.Castor)
	groupId := groupsig.DeserializeId(bh.GroupId)

	traceLog.logStart("height=%v, castor=%v", bh.Height, castor.ShortS())
	blog.debug("proc(%v) begin hash=%v, height=%v, sender=%v, castor=%v", p.getPrefix(), bh.Hash.ShortS(), bh.Height, si.GetID().ShortS(), castor.ShortS())

	result := ""
	defer func() {
		traceLog.logEnd("height=%v, hash=%v, preHash=%v, result=%v", bh.Height, bh.Hash.ShortS(), bh.PreHash.ShortS(), result)
		blog.debug("height=%v, hash=%v, preHash=%v, result=%v", bh.Height, bh.Hash.ShortS(), bh.PreHash.ShortS(), result)
	}()

	if !p.IsMinerGroup(groupId) { //检测当前节点是否在该铸块组
		result = fmt.Sprintf("don't belong to group, gid=%v, hash=%v, id=%v", groupId.ShortS(), bh.Hash.ShortS(), p.GetMinerID().ShortS())
		return
	}

	//castor要忽略自己的消息
	if castor.IsEqual(p.GetMinerID()) && si.GetID().IsEqual(p.GetMinerID()) {
		result = "ignore self message"
		return
	}

	if msg.GenHash() != si.DataHash {
		blog.debug("msg proveHash=%v", msg.ProveHash)
		result = fmt.Sprintf("msg genHash %v diff from si.DataHash %v", msg.GenHash().ShortS(), si.DataHash.ShortS())
		return
	}

	isProposal := castor.IsEqual(si.GetID())

	if isProposal { //提案者
		castorDO := p.minerReader.getProposeMiner(castor)
		if castorDO == nil {
			result = fmt.Sprintf("castorDO nil id=%v", castor.ShortS())
			return
		}
		if !msg.VerifySign(castorDO.PK) {
			result = fmt.Sprintf("verify sign fail, id %v, pk %v, castorDO %+v", castor.ShortS(), castorDO.PK.GetHexString(), castorDO)
			return
		}
	} else {
		pk := p.GetMemberSignPubKey(model.NewGroupMinerID(groupId, si.GetID()))
		if !msg.VerifySign(pk) {
			result = fmt.Sprintf("verify sign fail, id %v, pk %v, jg %+v", si.GetID().ShortS(), pk.GetHexString(), p.belongGroups.getJoinedGroup(groupId))
			return
		}
	}
	if p.blockOnChain(bh) {
		result = "block onchain already"
	}

	if err := p.doVerify(mtype, msg, traceLog, blog); err != nil {
		result = err.Error()
	}

	return
}

//收到组内成员的出块消息，出块人（KING）用组分片密钥进行了签名
//有可能没有收到OnMessageCurrent就提前接收了该消息（网络时序问题）
func (p *Processor) OnMessageCast(ccm *model.ConsensusCastMessage) {
	statistics.AddBlockLog(common.BootId, statistics.RcvCast, ccm.BH.Height, ccm.BH.ProveValue.Uint64(), -1, -1,
		time.Now().UnixNano(), "", "", common.InstanceIndex, ccm.BH.CurTime.UnixNano())
	p.verifyCastMessage("OMC", &ccm.ConsensusBlockMessageBase)
}

//收到组内成员的出块验证通过消息（组内成员消息）
func (p *Processor) OnMessageVerify(cvm *model.ConsensusVerifyMessage) {
	statistics.AddBlockLog(common.BootId, statistics.RcvVerified, cvm.BH.Height, cvm.BH.ProveValue.Uint64(), -1, -1,
		time.Now().UnixNano(), "", "", common.InstanceIndex, cvm.BH.CurTime.UnixNano())
	p.verifyCastMessage("OMV", &cvm.ConsensusBlockMessageBase)
}

//func (p *Processor) receiveBlock(block *types.Block, preBH *types.BlockHeader) bool {
//	if ok, err := p.isCastLegal(block.Header, preBH); ok { //铸块组合法
//		result := p.doAddOnChain(block)
//		if result == 0 || result == 1 {
//			return true
//		}
//	} else {
//		//丢弃该块
//		newBizLog("receiveBlock").log("received invalid new block, height=%v, err=%v", block.Header.Height, err.Error())
//	}
//	return false
//}

func (p *Processor) cleanVerifyContext(currentHeight uint64) {
	p.blockContexts.forEachBlockContext(func(bc *BlockContext) bool {
		bc.CleanVerifyContext(currentHeight)
		return true
	})
}

//收到铸块上链消息(组外矿工节点处理)
func (p *Processor) OnMessageBlock(cbm *model.ConsensusBlockMessage) {
	//statistics.AddBlockLog(common.BootId,statistics.RcvNewBlock,cbm.Block.Header.Height,cbm.Block.Header.ProveValue.Uint64(),len(cbm.Block.Transactions),-1,
	//	time.Now().UnixNano(),"","",common.InstanceIndex,cbm.Block.Header.CurTime.UnixNano())
	//bh := cbm.Block.Header
	//blog := newBizLog("OMB")
	//tlog := newHashTraceLog("OMB", bh.Hash, groupsig.DeserializeId(bh.Castor))
	//tlog.logStart("height=%v, preHash=%v", bh.Height, bh.PreHash.ShortS())
	//result := ""
	//defer func() {
	//	tlog.logEnd("height=%v, preHash=%v, result=%v", bh.Height, bh.PreHash.ShortS(), result)
	//}()
	//
	//if p.getBlockHeaderByHash(cbm.Block.Header.Hash) != nil {
	//	//blog.log("OMB receive block already on chain! bh=%v", p.blockPreview(cbm.Block.Header))
	//	result = "已经在链上"
	//	return
	//}
	//var gid = groupsig.DeserializeId(cbm.Block.Header.GroupId)
	//
	//blog.log("proc(%v) begin OMB, group=%v, height=%v, hash=%v...", p.getPrefix(),
	//	gid.ShortS(), cbm.Block.Header.Height, bh.Hash.ShortS())
	//
	//block := &cbm.Block
	//
	//preHeader := p.MainChain.GetTraceHeader(block.Header.PreHash.Bytes())
	//if preHeader == nil {
	//	p.addFutureBlockMsg(cbm)
	//	result = "父块未到达"
	//	return
	//}
	////panic("isBHCastLegal: cannot find pre block header!,ignore block")
	//verify := p.verifyGroupSign(cbm, preHeader)
	//if !verify {
	//	result = "组签名未通过"
	//	blog.log("OMB verifyGroupSign result=%v.", verify)
	//	return
	//}
	//
	//ret := p.receiveBlock(block, preHeader)
	//if ret {
	//	result = "上链成功"
	//} else {
	//	result = "上链失败"
	//}

	//blog.log("proc(%v) end OMB, group=%v, sender=%v...", p.getPrefix(), GetIDPrefix(cbm.GroupID), GetIDPrefix(cbm.SI.GetID()))
	return
}

//新的交易到达通知（用于处理大臣验证消息时缺失的交易）
func (p *Processor) OnMessageNewTransactions(ths []common.Hash) {
	mtype := "OMNT"
	blog := newBizLog(mtype)

	txstrings := make([]string, len(ths))
	for idx, tx := range ths {
		txstrings[idx] = tx.ShortS()
	}

	blog.debug("proc(%v) begin %v, trans count=%v %v...", p.getPrefix(),mtype, len(ths), txstrings)

	p.blockContexts.forEachBlockContext(func(bc *BlockContext) bool {
		for _, vctx := range bc.SafeGetVerifyContexts() {
			for _, slot := range vctx.GetSlots() {
				acceptRet := vctx.AcceptTrans(slot, ths)
				tlog := newHashTraceLog(mtype, slot.BH.Hash, groupsig.ID{})
				switch acceptRet {
				case TRANS_INVALID_SLOT, TRANS_DENY:

				case TRANS_ACCEPT_NOT_FULL:
					blog.debug("accept trans bh=%v, ret %v", p.blockPreview(slot.BH), acceptRet)
					tlog.log("preHash=%v, height=%v, %v,收到 %v, 总交易数 %v, 仍缺失数 %v", slot.BH.PreHash.ShortS(), slot.BH.Height, TRANS_ACCEPT_RESULT_DESC(acceptRet), len(ths), len(slot.BH.Transactions), slot.lostTransSize())

				case TRANS_ACCEPT_FULL_PIECE:
					blog.debug("accept trans bh=%v, ret %v", p.blockPreview(slot.BH), acceptRet)
					tlog.log("preHash=%v, height=%v %v, 当前分片数%v", slot.BH.PreHash.ShortS(), slot.BH.Height, TRANS_ACCEPT_RESULT_DESC(acceptRet), slot.MessageSize())

				case TRANS_ACCEPT_FULL_THRESHOLD:
					blog.debug("accept trans bh=%v, ret %v", p.blockPreview(slot.BH), acceptRet)
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
	gHash := grm.GInfo.GroupHash()
	gis := &grm.GInfo.GI
	gh := gis.GHeader

	blog.log("proc(%v) begin, sender=%v, gHash=%v...", p.getPrefix(), grm.SI.GetID().ShortS(), gHash.ShortS())
	tlog := newHashTraceLog("OMGI", gHash, grm.SI.GetID())

	if grm.SI.DataHash != grm.GenHash() || gh.Hash != gh.GenHash() {
		panic("grm gis hash diff")
	}

	topHeight := p.MainChain.QueryTopBlock().Height
	if gis.ReadyTimeout(topHeight) {
		blog.debug("OMGI ready timeout, readyHeight=%v, now=%v", gh.ReadyHeight, topHeight)
		return
	}

	if ok, err := p.CheckGroupHeader(gh, gis.Signature); !ok {
		blog.debug("group header illegal, err=%v", err)
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
	p.NetServer.BuildGroupNet(gHash.Hex(), grm.GInfo.Mems)

	gs := groupContext.GetGroupStatus()
	blog.debug("joining group(%v) status=%v.", gHash.ShortS(), gs)
	if gs == GIS_RAW {
		//blog.log("begin GenSharePieces in OMGI...")
		shares := groupContext.GenSharePieces() //生成秘密分享
		//blog.log("proc(%v) end GenSharePieces in OMGI, piece size=%v.", p.getPrefix(), len(shares))

		spm := &model.ConsensusSharePieceMessage{
			GHash: gHash,
		}
		ski := model.NewSecKeyInfo(p.GetMinerID(), p.mi.GetDefaultSecKey())
		spm.SI.SignMember = p.GetMinerID()

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

	//blog.log("proc(%v) end OMGI, sender=%v.", p.getPrefix(), GetIDPrefix(grm.SI.GetID()))
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
	blog.log("proc(%v) after gc.PieceMessage, gc result=%v.", p.getPrefix(), result)

	tlog.log("收到piece数 %v", gc.node.groupInitPool.GetSize())

	if result == 1 { //已聚合出签名私钥
		jg := gc.GetGroupInfo()
		//这时还没有所有组成员的签名公钥
		if jg.GroupPK.IsValid() && jg.SignKey.IsValid() {
			{
				ski := model.NewSecKeyInfo(p.mi.GetMinerID(), p.mi.GetDefaultSecKey())
				msg := &model.ConsensusSignPubKeyMessage{
					GHash: spm.GHash,
					SignPK:  *groupsig.NewPubkeyFromSeckey(jg.SignKey),
				}

				//对GISHash做自己的签名
				msg.GenGSign(jg.SignKey)
				if !msg.VerifyGSign(msg.SignPK) {
					panic("verify GSign with group member sign pub key failed.")
				}

				if msg.GenSign(ski, msg) {
					//todo : 组内广播签名公钥
					blog.debug("send sign pub key to group members, spk=%v...", msg.SignPK.ShortS())
					tlog.log("SendSignPubKey %v", p.getPrefix())

					blog.debug("call network service SendSignPubKey...")
					p.NetServer.SendSignPubKey(msg)
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

	blog.log("proc(%v) begin , sender=%v, gHash=%v...", p.getPrefix(), spkm.SI.GetID().ShortS(), spkm.GHash.ShortS())

	gc := p.joiningGroups.GetGroup(spkm.GHash)
	if gc == nil {
		blog.debug("failed, local node not found joining group with dummy id=%v.", spkm.GHash.ShortS())
		return
	}
	if spkm.GenHash() != spkm.SI.DataHash {
		blog.log("spkm hash diff")
		return
	}
	if gc.gInfo.GI.GHeader.GenHash() != spkm.GHash {
		blog.log("failed, gisHash diff.")
		return
	}
	pk := GetMinerPK(spkm.SI.GetID())
	if pk == nil {
		blog.log("miner pk is nil, id=%v", spkm.SI.GetID().ShortS())
		return
	}
	if !spkm.VerifySign(*pk) {
		blog.log("miner sign verify fail")
		return
	}
	if !spkm.VerifyGSign(spkm.SignPK) {
		panic("OMSP verify GSign with sign pub key failed.")
	}
	topHeight := p.MainChain.QueryTopBlock().Height
	if gc.gInfo.GI.ReadyTimeout(topHeight) {
		blog.log("ready timeout, readyHeight=%v, now=%v", gc.gInfo.GI.GHeader.ReadyHeight, topHeight)
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
				msg := &model.ConsensusGroupInitedMessage{
					GHash: spkm.GHash,
					GroupPK: jg.GroupPK,
					GroupID: jg.GroupID,
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
			panic("Processor::OnMessageSharePiece failed, aggr key error.")
		}
		p.joiningGroups.RemoveGroup(spkm.GHash)
	}

	//blog.log("proc(%v) end OMSPK, sender=%v, dummy gid=%v.", p.getPrefix(), GetIDPrefix(spkm.SI.GetID()), GetIDPrefix(spkm.DummyID))
	return
}

func (p *Processor) acceptGroup(staticGroup *StaticGroupInfo) {
	add := p.globalGroups.AddStaticGroup(staticGroup)
	blog := newBizLog("acceptGroup")
	blog.debug("Add to Global static groups, result=%v, groups=%v.", add, p.globalGroups.GetGroupSize())
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
	gHash := gim.GHash

	blog.log("proc(%v) begin, sender=%v, gHash=%v, gid=%v, gpk=%v...", p.getPrefix(),
		gim.SI.GetID().ShortS(), gHash.ShortS(), gim.GroupID.ShortS(), gim.GroupPK.ShortS())
	tlog := newHashTraceLog("OMGIED", gHash, gim.SI.GetID())

	if gim.SI.DataHash != gim.GenHash() {
		panic("grm gis hash diff")
	}
	initingGroup := p.globalGroups.GetInitingGroup(gHash)
	if initingGroup == nil {
		blog.log("initingGroup not found!gHash=%v", gHash.ShortS())
		return
	}
	topHeight := p.MainChain.QueryTopBlock().Height
	if initingGroup.ReadyTimeout(topHeight) {
		blog.log("ready timeout, readyHeight=%v, now=%v", initingGroup.gInfo.GI.GHeader.ReadyHeight, topHeight)
		return
	}

	gis := &initingGroup.gInfo.GI

	parentGroup := p.GetGroup(gis.ParentID())

	gpk := parentGroup.GroupPK
	if !groupsig.VerifySig(gpk, gis.GetHash().Bytes(), gis.Signature) {
		blog.log("verify parent groupsig fail! gHash=%v", gHash.ShortS())
		return
	}
	if !initingGroup.MemberExist(gim.SI.SignMember) {
		return
	}
	//上链前检查
	//if ok, err := p.groupManager.isGroupHeaderLegal(gis.GHeader); !ok {
	//	blog.log("group header illegal, gHash=%v, err=%v", gHash.ShortS(), err)
	//	return
	//}

	var result int32
	if !initingGroup.MemberExist(p.GetMinerID()) {
		result = p.globalGroups.GroupInitedMessage(gim, topHeight)

		blog.debug("proc(%v) globalGroups.GroupInitedMessage result=%v.", p.getPrefix(), result)
		tlog.log("收到消息数量 %v", initingGroup.receiveSize())
	} else {
		result = INIT_SUCCESS
		tlog.log("组内成员，收到组初始化完成消息")
	}

	switch result {
	case INIT_SUCCESS: //收到组内相同消息>=阈值，可上链
		staticGroup := NewSGIFromStaticGroupSummary(gim.GroupID, gim.GroupPK, initingGroup)
		gh := staticGroup.getGroupHeader()
		blog.debug("SUCCESS accept a new group, gHash=%v, gid=%v, workHeight=%v, dismissHeight=%v.", gHash.ShortS(), gim.GroupID.ShortS(), gh.WorkHeight, gh.DismissHeight)

		//p.acceptGroup(staticGroup)
		p.groupManager.AddGroupOnChain(staticGroup)

		p.globalGroups.removeInitingGroup(gHash)

	case INIT_FAIL: //该组初始化异常，且无法恢复
		tlog.log("初始化失败")
		p.globalGroups.removeInitingGroup(gHash)

	case INITING:
		//继续等待下一包数据
	}
	//blog.log("proc(%v) end OMGIED, sender=%v...", p.getPrefix(), GetIDPrefix(gim.SI.GetID()))
	return
}

func (p *Processor) OnMessageCreateGroupRaw(msg *model.ConsensusCreateGroupRawMessage) {
	blog := newBizLog("OMCRG")
	gh := msg.GInfo.GI.GHeader
	blog.log("Proc(%v) begin, gHash=%v sender=%v", p.getPrefix(), gh.Hash.ShortS(), msg.SI.SignMember.ShortS())

	if p.GetMinerID().IsEqual(msg.SI.SignMember) {
		return
	}
	parentGid := msg.GInfo.GI.ParentID()

	gpk := p.GetMemberSignPubKey(model.NewGroupMinerID(parentGid, msg.SI.SignMember))
	if !gpk.IsValid() {
		return
	}
	if !msg.SI.VerifySign(gpk) {
		return
	}
	if gh.Hash != gh.GenHash() || gh.Hash != msg.SI.DataHash {
		blog.log("hash diff expect %v, receive %v", gh.GenHash().ShortS(), gh.Hash.ShortS())
		return
	}

	tlog := newHashTraceLog("OMCGR", gh.Hash, msg.SI.GetID())
	if p.groupManager.OnMessageCreateGroupRaw(msg) {
		signMsg := &model.ConsensusCreateGroupSignMessage{
			Launcher: msg.SI.SignMember,
			GHash: gh.Hash,
		}
		ski := model.NewSecKeyInfo(p.GetMinerID(), p.getSignKey(parentGid))
		if signMsg.GenSign(ski, signMsg) {
			tlog.log("SendCreateGroupSignMessage id=%v", p.getPrefix())
			blog.debug("OMCGR SendCreateGroupSignMessage... ")
			p.NetServer.SendCreateGroupSignMessage(signMsg, parentGid)
		} else {
			blog.debug("SendCreateGroupSignMessage sign fail, ski=%v, %v", ski.ID.ShortS(), ski.SK.ShortS(), p.IsMinerGroup(parentGid))
		}

	} else {
		tlog.log("groupManager.OnMessageCreateGroupRaw fail")

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
	creating := p.groupManager.creatingGroups.getCreatingGroup(msg.GHash)
	if creating == nil {
		blog.log("get creating group nil!gHash=%v", msg.GHash.ShortS())
		return
	}

	parentGid := creating.gInfo.GI.ParentID()

	mpk := p.GetMemberSignPubKey(model.NewGroupMinerID(parentGid, msg.SI.SignMember))
	if !mpk.IsValid() {
		return
	}
	if !msg.VerifySign(mpk) {
		return
	}
	if p.groupManager.OnMessageCreateGroupSign(msg, creating) {
		gpk := p.getGroupPubKey(parentGid)
		if !groupsig.VerifySig(gpk, msg.SI.DataHash.Bytes(), creating.gInfo.GI.Signature) {
			blog.log("Proc(%v) verify group sign fail", p.getPrefix())
			return
		}
		initMsg := &model.ConsensusGroupRawMessage{
			GInfo:   *creating.gInfo,
		}

		blog.debug("Proc(%v) send group init Message", p.getPrefix())
		ski := model.NewSecKeyInfo(p.GetMinerID(), p.getMinerInfo().GetDefaultSecKey())
		if initMsg.GenSign(ski, initMsg) {
			tlog := newHashTraceLog("OMCGS", msg.GHash, msg.SI.GetID())
			tlog.log("SendGroupInitMessage")
			p.NetServer.SendGroupInitMessage(initMsg)

			p.groupManager.removeCreatingGroup(msg.GHash)
		} else {
			blog.log("genSign fail, id=%v, sk=%v", ski.ID.ShortS(), ski.SK.ShortS())
		}

	}
}

func (p *Processor) signCastRewardReq(msg *model.CastRewardTransSignReqMessage, bh *types.BlockHeader) (send bool, err error) {
	gid := groupsig.DeserializeId(bh.GroupId)
	group := p.GetGroup(gid)
	reward := &msg.Reward
	if group == nil {
		panic("group is nil")
	}
	if !msg.VerifySign(p.GetMemberSignPubKey(model.NewGroupMinerID(gid, msg.SI.GetID()))) {
		err = fmt.Errorf("verify sign fail, gid=%v, uid=%v", gid.ShortS(), msg.SI.GetID().ShortS())
		return
	}

	if !bytes.Equal(bh.GroupId, reward.GroupId) {
		err = fmt.Errorf("groupID error %v %v", bh.GroupId, reward.GroupId)
		return
	}
	genBonus, _ := p.MainChain.GetBonusManager().GenerateBonus(reward.TargetIds, bh.Hash, bh.GroupId, model.Param.VerifyBonus)
	if genBonus.TxHash != reward.TxHash {
		err = fmt.Errorf("bonus txHash diff %v %v", genBonus.TxHash.ShortS(), reward.TxHash.ShortS())
		return
	}

	bc := p.GetBlockContext(gid)
	if bc == nil {
		err = fmt.Errorf("blockcontext is nil, gid=%v", gid.ShortS())
		return
	}
	vctx := bc.GetVerifyContextByHeight(bh.Height)
	if vctx == nil || vctx.prevBH.Hash != bh.PreHash {
		err = fmt.Errorf("vctx is nil")
		return
	}

	slot := vctx.GetSlotByHash(bh.Hash)
	if slot == nil {
		err = fmt.Errorf("slot is nil")
		return
	}
	if slot.IsRewardSent() { //已发送过分红交易，不再为此签名
		err = fmt.Errorf("alreayd sent reward trans")
		return
	}

	if len(msg.Reward.TargetIds) != len(msg.SignedPieces) {
		err = fmt.Errorf("targetId len differ from signedpiece len %v %v", len(msg.Reward.TargetIds), len(msg.SignedPieces))
		return
	}

	gSignGener := model.NewGroupSignGenerator(bc.threshold())

	//witnesses := slot.gSignGenerator.GetWitnesses()
	for idx, idIndex := range msg.Reward.TargetIds {
		id := group.GetMemberID(int(idIndex))
		sign := msg.SignedPieces[idx]
		if sig, ok := slot.gSignGenerator.GetWitness(id); !ok { //本地无该id签名的，需要校验签名
			pk := p.GetMemberSignPubKey(model.NewGroupMinerID(gid, id))
			if !groupsig.VerifySig(pk, bh.Hash.Bytes(), sign) {
				err = fmt.Errorf("verify member sign fail, id=%v", id.ShortS())
				return
			}
		} else { //本地已有该id的签名的，只要判断是否跟本地签名一样即可
			if !sign.IsEqual(sig) {
				err = fmt.Errorf("member sign different id=%v", id.ShortS())
				return
			}
		}
		gSignGener.AddWitness(id, sign)
	}
	if !gSignGener.Recovered() {
		err = fmt.Errorf("recover group sign fail")
		return
	}
	bhSign := groupsig.DeserializeSign(bh.Signature)
	if !gSignGener.GetGroupSign().IsEqual(*bhSign) {
		err = fmt.Errorf("recovered sign differ from bh sign, recover %v, bh %v", gSignGener.GetGroupSign().ShortS(), bhSign.ShortS())
		return
	}

	send = true
	//自己签名
	signMsg := &model.CastRewardTransSignMessage{
		ReqHash:   reward.TxHash,
		BlockHash: reward.BlockHash,
		GroupID:   gid,
		Launcher:  msg.SI.GetID(),
	}
	ski := model.NewSecKeyInfo(p.GetMinerID(), p.getSignKey(gid))
	if signMsg.GenSign(ski, signMsg) {
		p.NetServer.SendCastRewardSign(signMsg)
	} else {
		err = fmt.Errorf("signCastRewardReq genSign fail, id=%v, sk=%v, %v", ski.ID.ShortS(), ski.SK.ShortS(), p.IsMinerGroup(gid))
	}
	return
}

func (p *Processor) OnMessageCastRewardSignReq(msg *model.CastRewardTransSignReqMessage) {
	mtype := "OMCRSR"
	blog := newBizLog(mtype)
	reward := &msg.Reward
	tlog := newHashTraceLog("OMCRSR", reward.BlockHash, msg.SI.GetID())
	blog.log("begin, sender=%v, blockHash=%v, txHash=%v", msg.SI.GetID().ShortS(), reward.BlockHash.ShortS(), reward.TxHash.ShortS())
	tlog.logStart("txHash=%v", reward.TxHash.ShortS())

	var (
		send bool
		err  error
	)

	defer func() {
		tlog.logEnd("%v %v", send, err)
		blog.log("blockHash=%v, result=%v %v", reward.BlockHash.ShortS(), send, err)
	}()

	//此时块不一定在链上
	bh := p.getBlockHeaderByHash(reward.BlockHash)
	if bh == nil {
		err = fmt.Errorf("future reward request receive and cached, hash=%v", reward.BlockHash.ShortS())
		msg.ReceiveTime = time.Now()
		p.futureRewardReqs.addMessage(reward.BlockHash, msg)
		return
	}

	send, err = p.signCastRewardReq(msg, bh)
	return
}

// 收到分红奖励消息
func (p *Processor) OnMessageCastRewardSign(msg *model.CastRewardTransSignMessage) {
	mtype := "OMCRS"
	blog := newBizLog(mtype)

	blog.log("begin, sender=%v, reqHash=%v", msg.SI.GetID().ShortS(), msg.ReqHash.ShortS())
	tlog := newHashTraceLog(mtype, msg.BlockHash, msg.SI.GetID())

	tlog.logStart("txHash=%v", msg.ReqHash.ShortS())

	var (
		send bool
		err  error
	)

	defer func() {
		tlog.logEnd("bonus send:%v, ret:%v", send, err)
		blog.log("blockHash=%v, send=%v, result=%v", msg.BlockHash.ShortS(), send, err)
	}()

	bh := p.getBlockHeaderByHash(msg.BlockHash)
	if bh == nil {
		err = fmt.Errorf("block not exist, hash=%v", msg.BlockHash.ShortS())
		return
	}

	gid := groupsig.DeserializeId(bh.GroupId)
	group := p.GetGroup(gid)
	if group == nil {
		panic("group is nil")
	}
	if !msg.VerifySign(p.GetMemberSignPubKey(model.NewGroupMinerID(gid, msg.SI.GetID()))) {
		err = fmt.Errorf("verify sign fail")
		return
	}

	bc := p.GetBlockContext(gid)
	if bc == nil {
		err = fmt.Errorf("blockcontext is nil, gid=%v", gid.ShortS())
		return
	}
	vctx := bc.GetVerifyContextByHeight(bh.Height)
	if vctx == nil || vctx.prevBH.Hash != bh.PreHash {
		err = fmt.Errorf("vctx is nil")
		return
	}

	slot := vctx.GetSlotByHash(bh.Hash)
	if slot == nil {
		err = fmt.Errorf("slot is nil")
		return
	}

	accept, recover := slot.AcceptRewardPiece(&msg.SI)
	blog.log("slot acceptRewardPiece %v %v status %v", accept, recover, slot.GetSlotStatus())
	if accept && recover && slot.StatusTransform(SS_REWARD_REQ, SS_REWARD_SEND) {
		_, err2 := p.MainChain.GetTransactionPool().AddTransaction(slot.rewardTrans)
		send = true
		err = fmt.Errorf("add rewardTrans to txPool, txHash=%v, ret=%v", slot.rewardTrans.Hash.ShortS(), err2)
	} else {
		err = fmt.Errorf("accept %v, recover %v, %v", accept, recover, slot.rewardGSignGen.Brief())
	}
}
