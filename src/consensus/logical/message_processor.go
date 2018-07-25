package logical

import (
	"common"
	"consensus/groupsig"

	"fmt"
	"time"
	"log"
	"sync/atomic"
	"middleware/types"
)

func (p *Processor) genCastGroupSummary(bh *types.BlockHeader) *CastGroupSummary {
	var gid groupsig.ID
	if err := gid.Deserialize(bh.GroupId); err != nil {
		log.Printf("fail to deserialize groupId: gid=%v, err=%v\n", bh.GroupId, err)
		return nil
	}
	var castor groupsig.ID
	if err := castor.Deserialize(bh.Castor); err != nil {
		log.Printf("fail to deserialize castor: castor=%v, err=%v\n", bh.Castor, err)
		return nil
	}
	cgs := &CastGroupSummary{
		PreHash:     bh.Hash,
		PreTime:     bh.PreTime,
		BlockHeight: bh.Height,
		GroupID:     gid,
		Castor:      castor,
	}
	cgs.CastorPos = p.getMinerPos(cgs.GroupID, cgs.Castor)
	return cgs
}

func (p *Processor) thresholdPieceVerify(mtype string, sender string, gid groupsig.ID, vctx *VerifyContext, slot *SlotContext, bh *types.BlockHeader)  {
	gpk := p.getGroupPubKey(gid)
	if !slot.GenGroupSign() {
		logHalfway(mtype, bh.Height, bh.QueueNumber, sender, "gen group sign fail")
		return
	}
	sign := slot.GetGroupSign()
	if !slot.VerifyGroupSign(gpk) { //组签名验证通过
		log.Printf("%v group pub key local check failed, gpk=%v, sign=%v, hash in slot=%v, hash in bh=%v.\n", mtype,
			GetPubKeyPrefix(gpk), GetSignPrefix(sign), GetHashPrefix(slot.BH.Hash), GetHashPrefix(bh.Hash))
		return
	}

	bh.Signature = sign.Serialize()

	if atomic.CompareAndSwapInt32(&slot.SlotStatus, SS_VERIFIED, SS_ONCHAIN) {
		p.SuccessNewBlock(bh, vctx, gid) //上链和组外广播
		//log.Printf("%v remove verifycontext from bccontext! remain size=%v\n", mtype, len(bc.verifyContexts))
	}

}

func (p *Processor) normalPieceVerify(mtype string, sender string, gid groupsig.ID, slot *SlotContext, bh *types.BlockHeader)  {
	castor := groupsig.DeserializeId(bh.Castor)
	if atomic.CompareAndSwapInt32(&slot.SlotStatus, SS_WAITING, SS_BRAODCASTED) && !castor.IsEqual(p.GetMinerID()) {
		var cvm ConsensusVerifyMessage
		cvm.BH = *bh
		//cvm.GroupID = gId
		cvm.GenSign(SecKeyInfo{p.GetMinerID(), p.getSignKey(gid)}, &cvm)
		if !PROC_TEST_MODE {
			log.Printf("call network service SendVerifiedCast...\n")
			logHalfway(mtype, bh.Height, bh.QueueNumber, sender, "SendVerifiedCast")
			go SendVerifiedCast(&cvm)
		} else {
			for _, v := range p.GroupProcs {
				v.OnMessageVerify(cvm)
			}
		}

	}
}

func (p *Processor) doVerify(mtype string, msg *ConsensusBlockMessageBase, cgs *CastGroupSummary) {
	bh := &msg.BH
	si := &msg.SI

	sender := GetIDPrefix(si.SignMember)
	result := ""

	log.Printf("%v message bh %v, top bh %v\n", mtype, p.blockPreview(bh), p.blockPreview(p.MainChain.QueryTopBlock()))

	if p.blockOnChain(bh) {
		log.Printf("%v receive block already onchain! , height = %v\n", mtype, bh.Height)
		result = "已经上链"
		logHalfway(mtype, bh.Height, bh.QueueNumber, sender, "preHash %v, doVerify begin: %v", GetHashPrefix(bh.PreHash), result)
		return
	}

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
		logHalfway(mtype, bh.Height, bh.QueueNumber, sender, "preHash %v, doVerify begin: %v", GetHashPrefix(bh.PreHash), result)
		return
	}

	if !p.isCastGroupLegal(bh, preBH) {
		result = "非法的铸块组"
		logHalfway(mtype, bh.Height, bh.QueueNumber, sender, "preHash %v, doVerify begin: %v", GetHashPrefix(bh.PreHash), result)
		log.Printf("not the casting group!bh=%v, preBH=%v", p.blockPreview(bh), p.blockPreview(preBH))
		panic("cast !!")
		return
	}

	bc := p.GetBlockContext(gid)
	if bc == nil {
		result = "未获取到blockcontext"
		logHalfway(mtype, bh.Height, bh.QueueNumber, sender, "preHash %v, doVerify begin: %v", GetHashPrefix(bh.PreHash), result)
		log.Printf("[ERROR]blockcontext is nil!, gid=" + GetIDPrefix(gid))
		return
	}

	_, vctx := bc.GetOrNewVerifyContext(bh, preBH)

	verifyResult := vctx.UserVerified(bh, si, cgs)
	log.Printf("proc(%v) %v UserVerified result=%v.\n", mtype, p.getPrefix(), verifyResult)
	slot := vctx.GetSlotByQN(int64(bh.QueueNumber))
	if slot == nil {
		result = "找不到合适的验证槽, 放弃验证"
		logHalfway(mtype, bh.Height, bh.QueueNumber, sender, "preHash %v, doVerify begin: %v", GetHashPrefix(bh.PreHash), result)
		return
	}

	result = fmt.Sprintf("%v, 当前分片数 %v, %v, %v", CBMR_RESULT_DESC(verifyResult), len(slot.MapWitness), slot.thresholdWitnessGot(), slot.threshold)
	logHalfway(mtype, bh.Height, bh.QueueNumber, sender, "preHash %v, doVerify begin: %v", GetHashPrefix(bh.PreHash), result)

	switch verifyResult {
	case CBMR_THRESHOLD_SUCCESS:
		log.Printf("proc(%v) %v msg_count reach threshold!\n", mtype, p.getPrefix())
		if slot.IsTransFull() {
			p.thresholdPieceVerify(mtype, sender, gid, vctx, slot, bh)
		} else {
			logHalfway(mtype, bh.Height, bh.QueueNumber, sender, "preHash %v, 收到所有分片, 但缺失交易, 总共交易%v, 缺失%v", GetHashPrefix(bh.PreHash), len(bh.Transactions), slot.lostTransSize())
		}

	case CBMR_PIECE_NORMAL:
		p.normalPieceVerify(mtype, sender, gid, slot, bh)

	case CBMR_PIECE_LOSINGTRANS: //交易缺失
		logHalfway(mtype, bh.Height, bh.QueueNumber, sender, "preHash %v, 总交易数 %v, 仍缺失数 %v", GetHashPrefix(bh.PreHash), len(bh.Transactions), len(slot.LosingTrans))
	}
}

func (p *Processor) verifyCastMessage(mtype string, msg *ConsensusBlockMessageBase) {
	bh := &msg.BH
	si := &msg.SI

	logStart(mtype, bh.Height, bh.QueueNumber, GetIDPrefix(si.SignMember), "")

	defer func() {
		logEnd(mtype, bh.Height, bh.QueueNumber, GetIDPrefix(si.SignMember))
	}()

	cgs := p.genCastGroupSummary(bh)
	if cgs == nil {
		log.Printf("[ERROR]%v gen castGroupSummary fail!\n", mtype)
		return
	}
	log.Printf("proc(%v) begin %v, group=%v, sender=%v, height=%v, qn=%v, castor=%v...\n", p.getPrefix(), mtype, GetIDPrefix(cgs.GroupID), GetIDPrefix(si.GetID()), bh.Height, bh.QueueNumber, GetIDPrefix(cgs.Castor))

	//如果是自己发的, 不处理
	if p.GetMinerID().IsEqual(si.SignMember) {
		return
	}

	outputBlockHeaderAndSign(mtype, bh, si)

	if !p.verifyCastSign(cgs, si) {
		log.Printf("%v verify failed!\n", mtype)
		return
	}

	p.doVerify(mtype, msg, cgs)

	return
}

//收到组内成员的出块消息，出块人（KING）用组分片密钥进行了签名
//有可能没有收到OnMessageCurrent就提前接收了该消息（网络时序问题）
func (p *Processor) OnMessageCast(ccm ConsensusCastMessage) {
	p.verifyCastMessage("OMC", &ccm.ConsensusBlockMessageBase)
}

//收到组内成员的出块验证通过消息（组内成员消息）
func (p *Processor) OnMessageVerify(cvm ConsensusVerifyMessage) {
	p.verifyCastMessage("OMV", &cvm.ConsensusBlockMessageBase)
}

func (p *Processor) triggerFutureVerifyMsg(hash common.Hash) {
	futures := p.getFutureVerifyMsgs(hash)
	if futures == nil || len(futures) == 0 {
		return
	}
	p.removeFutureVerifyMsgs(hash)

	for _, msg := range futures {
		logStart("FUTURE_VERIFY", msg.BH.Height, msg.BH.QueueNumber, GetIDPrefix(msg.SI.SignMember), "size %v", len(futures))
		p.doVerify("FUTURE_VERIFY", msg, nil)
	}

}

func (p *Processor) receiveBlock(msg *ConsensusBlockMessage, preBH *types.BlockHeader) bool {
	if p.isCastGroupLegal(msg.Block.Header, preBH) { //铸块组合法
		result := p.doAddOnChain(&msg.Block)
		if result == 0 || result == 1 {
			return true
		}
	} else {
		//丢弃该块
		log.Printf("OMB received invalid new block, height = %v.\n", msg.Block.Header.Height)
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
func (p *Processor) OnMessageBlock(cbm ConsensusBlockMessage) {
	bh := cbm.Block.Header
	logStart("OMB", bh.Height, bh.QueueNumber, GetIDPrefix(cbm.SI.SignMember), "castor=%v", GetIDPrefix(*groupsig.DeserializeId(bh.Castor)))
	result := ""
	defer func() {
		logHalfway("OMB", bh.Height, bh.QueueNumber, GetIDPrefix(cbm.SI.SignMember), "OMB result %v", result)
		logEnd("OMB", bh.Height, bh.QueueNumber, GetIDPrefix(cbm.SI.SignMember))
	}()

	if p.MainChain.QueryBlockByHash(cbm.Block.Header.Hash) != nil {
		log.Printf("OMB receive block already on chain! bh=%v\n", p.blockPreview(cbm.Block.Header))
		result = "已经在链上"
		return
	}
	if p.GetMinerID().IsEqual(cbm.SI.SignMember) {
		result = "自己发的消息, 忽略"
		return
	}
	var gid groupsig.ID
	if gid.Deserialize(cbm.Block.Header.GroupId) != nil {
		panic("OMB Deserialize group_id failed")
	}
	log.Printf("proc(%v) begin OMB, group=%v(bh gid=%v), sender=%v, height=%v, qn=%v...\n", p.getPrefix(),
		GetIDPrefix(cbm.GroupID), GetIDPrefix(gid), GetIDPrefix(cbm.SI.GetID()), cbm.Block.Header.Height, cbm.Block.Header.QueueNumber)

	block := &cbm.Block
	//panic("isBHCastLegal: cannot find pre block header!,ignore block")
	verify := p.verifyGroupSign(block, cbm.SI)
	if !verify {
		result = "组签名未通过"
		log.Printf("OMB verifyGroupSign result=%v.\n", verify)
		return
	}

	topBH := p.MainChain.QueryTopBlock()

	preHeader := p.MainChain.QueryBlockByHash(block.Header.PreHash)
	if preHeader == nil {
		p.addFutureBlockMsg(&cbm)
		result = "父块未到达"
		return
	}

	ret := p.receiveBlock(&cbm, preHeader)
	if ret {
		preHeader = block.Header
		for {
			futureMsgs := p.getFutureBlockMsgs(preHeader.Hash)
			if futureMsgs == nil || len(futureMsgs) == 0 {
				break
			}
			log.Printf("handle future blocks, size=%v\n", len(futureMsgs))
			for _, msg := range futureMsgs {
				tbh := msg.Block.Header
				logHalfway("OMB", tbh.Height, tbh.QueueNumber, GetIDPrefix(msg.SI.SignMember), "trigger cached future block")
				p.receiveBlock(msg, preHeader)
			}
			p.removeFutureBlockMsgs(preHeader.Hash)
			preHeader = p.MainChain.QueryTopBlock()
		}
		result = "上链成功"
	} else {
		result = "上链失败"
	}

	nowTop := p.MainChain.QueryTopBlock()
	if topBH.Hash != nowTop.Hash {
		p.triggerCastCheck()
	}

	log.Printf("proc(%v) end OMB, group=%v, sender=%v...\n", p.getPrefix(), GetIDPrefix(cbm.GroupID), GetIDPrefix(cbm.SI.GetID()))
	return
}

//新的交易到达通知（用于处理大臣验证消息时缺失的交易）
func (p *Processor) OnMessageNewTransactions(ths []common.Hash) {
	begin := time.Now()
	mtype := "OMNT"
	logStart(mtype, 0, 0, "", "count=%v,txHash[0]=%v", len(ths), GetHashPrefix(ths[0]))
	defer func() {
		log.Printf("%v begin at %v, cost %v\n", mtype, begin.String(), time.Since(begin).String())
		logEnd(mtype, 0, 0, "")
	}()

	log.Printf("proc(%v) begin %v, trans count=%v...\n", p.getPrefix(),mtype, len(ths))

	p.blockContexts.forEach(func(bc *BlockContext) bool {
		for _, vctx := range bc.SafeGetVerifyContexts() {
			for _, slot := range vctx.slots {
				acceptRet := vctx.AcceptTrans(slot, ths)
				switch acceptRet {
				case TRANS_INVALID_SLOT, TRANS_DENY:

				case TRANS_ACCEPT_NOT_FULL:
					log.Printf("OMNT accept trans bh=%v, ret %v\n", p.blockPreview(&slot.BH), acceptRet)
					logHalfway(mtype, slot.BH.Height, slot.BH.QueueNumber, p.getPrefix(), "preHash %v, %v,收到 %v, 总交易数 %v, 仍缺失数 %v", GetHashPrefix(slot.BH.PreHash), TRANS_ACCEPT_RESULT_DESC(acceptRet), len(ths), len(slot.BH.Transactions), len(slot.LosingTrans))

				case TRANS_ACCEPT_FULL_PIECE:
					log.Printf("OMNT accept trans bh=%v, ret %v\n", p.blockPreview(&slot.BH), acceptRet)
					//_, ret := p.verifyBlock(&slot.BH)
					//if ret != 0 {
					//	logHalfway(mtype, slot.BH.Height, slot.BH.QueueNumber, p.getPrefix(), "all trans got, but verify fail, result=%v", ret)
					//	log.Printf("verify block failed!, won't sendVerifiedCast!bh=%v, ret=%v\n", p.blockPreview(&slot.BH), ret)
					//} else {
					//	p.normalPieceVerify(mtype, p.getPrefix(), bc.MinerID.gid, slot, &slot.BH)
					//}
					logHalfway(mtype, slot.BH.Height, slot.BH.QueueNumber, p.getPrefix(), "preHash %v, %v, 当前分片数%v", GetHashPrefix(slot.BH.PreHash), TRANS_ACCEPT_RESULT_DESC(acceptRet), len(slot.MapWitness))

				case TRANS_ACCEPT_FULL_THRESHOLD:
					//_, ret := p.verifyBlock(&slot.BH)
					//if ret != 0 {
					//	logHalfway(mtype, slot.BH.Height, slot.BH.QueueNumber, p.getPrefix(), "all trans got, but verify fail, result=%v", ret)
					//	log.Printf("verify block failed!, won't sendVerifiedCast!bh=%v, ret=%v\n", p.blockPreview(&slot.BH), ret)
					//	continue
					//}
					log.Printf("OMNT accept trans bh=%v, ret %v\n", p.blockPreview(&slot.BH), acceptRet)
					logHalfway(mtype, slot.BH.Height, slot.BH.QueueNumber, p.getPrefix(), "preHash %v, %v", GetHashPrefix(slot.BH.PreHash), TRANS_ACCEPT_RESULT_DESC(acceptRet))
					p.thresholdPieceVerify(mtype, p.getPrefix(), bc.MinerID.gid, vctx, slot, &slot.BH)
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

func (p *Processor) OnMessageGroupInit(grm ConsensusGroupRawMessage) {
	log.Printf("proc(%v) begin OMGI, sender=%v, dummy_gid=%v...\n", p.getPrefix(), GetIDPrefix(grm.SI.GetID()), GetIDPrefix(grm.GI.DummyID))

	if !grm.GI.checkMemberHash(grm.MEMS) {
		panic("grm member hash diff!")
	}
	if grm.SI.DataHash != grm.GI.GenHash() {
		panic("grm gis hash diff")
	}
	parentGroup := p.getGroup(grm.GI.ParentID)
	if !parentGroup.CastQualified(grm.GI.TopHeight) {
		log.Printf("OMGI parent group has no qualify to cast group. gid=%v, height=%v\n", GetIDPrefix(parentGroup.GroupID), grm.GI.TopHeight)
		return
	}
	gpk := parentGroup.GroupPK
	if !groupsig.VerifySig(gpk, grm.SI.DataHash.Bytes(), grm.GI.Signature) {
		log.Printf("OMGI verify parent groupsig fail!\n")
		return
	}


	if p.globalGroups.AddInitingGroup(CreateInitingGroup(&grm)) {
		//to do : 从链上检查消息发起人（父亲组成员）是否有权限发该消息（鸠兹）
		//dummy 组写入组链 add by 小熊
		staticGroupInfo := NewDummySGIFromGroupRawMessage(&grm)
		p.groupManager.AddGroupOnChain(staticGroupInfo, true)
	}

	//非组内成员不走后续流程
	if !grm.MemberExist(p.GetMinerID()) {
		return
	}
	//p.globalGroups.AddDummyGroup(sgi)

	groupContext := p.joiningGroups.ConfirmGroupFromRaw(&grm, p.mi)
	if groupContext == nil {
		panic("Processor::OMGI failed, ConfirmGroupFromRaw return nil.")
	}
	gs := groupContext.GetGroupStatus()
	log.Printf("OMGI joining group(%v) status=%v.\n", GetIDPrefix(grm.GI.DummyID), gs)
	if gs == GIS_RAW {
		log.Printf("begin GenSharePieces in OMGI...\n")
		shares := groupContext.GenSharePieces() //生成秘密分享
		log.Printf("proc(%v) end GenSharePieces in OMGI, piece size=%v.\n", p.getPrefix(), len(shares))

		spm := ConsensusSharePieceMessage{
			GISHash: grm.GI.GenHash(),
			DummyID: grm.GI.DummyID,
		}
		ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
		spm.SI.SignMember = p.GetMinerID()

		for id, piece := range shares {
			if id != "0x0" && piece.IsValid() {
				spm.Dest.SetHexString(id)
				spm.Share = piece
				sb := spm.GenSign(ski, &spm)
				log.Printf("OMGI spm.GenSign result=%v.\n", sb)
				log.Printf("OMGI piece to ID(%v), dummyId=%v, share=%v, pub=%v.\n", GetIDPrefix(spm.Dest), GetIDPrefix(spm.DummyID), GetSecKeyPrefix(spm.Share.Share), GetPubKeyPrefix(spm.Share.Pub))
				if !PROC_TEST_MODE {
					log.Printf("call network service SendKeySharePiece...\n")
					SendKeySharePiece(spm)
				} else {
					log.Printf("test mode, call OMSP direct...\n")
					destProc, ok := p.GroupProcs[spm.Dest.GetHexString()]
					if ok {
						destProc.OnMessageSharePiece(spm)
					} else {
						panic("ERROR, dest proc not found!\n")
					}
				}

			} else {
				panic("GenSharePieces data not IsValid.\n")
			}
		}
		log.Printf("end GenSharePieces.\n")
	} else {
		log.Printf("group(%v) status=%v, ignore init message.\n", GetIDPrefix(grm.GI.DummyID), gs)
	}

	log.Printf("proc(%v) end OMGI, sender=%v.\n", p.getPrefix(), GetIDPrefix(grm.SI.GetID()))
	return
}

//收到组内成员发给我的秘密分享片段消息
func (p *Processor) OnMessageSharePiece(spm ConsensusSharePieceMessage) {
	log.Printf("proc(%v)begin Processor::OMSP, sender=%v, dummyId=%v...\n", p.getPrefix(), GetIDPrefix(spm.SI.GetID()), GetIDPrefix(spm.DummyID))

	gc := p.joiningGroups.GetGroup(spm.DummyID)
	if gc == nil {
		panic("OMSP failed, receive SHAREPIECE msg but gc=nil.\n")
		return
	}
	if gc.gis.GenHash() != spm.GISHash {
		log.Printf("OMSPK failed, gisHash diff.\n")
		return
	}

	result := gc.PieceMessage(spm)
	log.Printf("proc(%v) OMSP after gc.PieceMessage, gc result=%v.\n", p.getPrefix(), result)

	if result == 1 { //已聚合出签名私钥
		jg := gc.GetGroupInfo()
		//这时还没有所有组成员的签名公钥
		if jg.GroupPK.IsValid() && jg.SignKey.IsValid() {
			log.Printf("OMSP SUCCESS gen sign sec key and group pub key, msk=%v, gpk=%v.\n", GetSecKeyPrefix(jg.SignKey), GetPubKeyPrefix(jg.GroupPK))
			{
				ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
				msg := ConsensusSignPubKeyMessage{
					GISHash: spm.GISHash,
					DummyID: spm.DummyID,
					SignPK:  *groupsig.NewPubkeyFromSeckey(jg.SignKey),
				}

				//对GISHash做自己的签名
				msg.GenGISSign(jg.SignKey)
				if !msg.VerifyGISSign(msg.SignPK) {
					panic("verify GISSign with group member sign pub key failed.")
				}

				msg.GenSign(ski, &msg)
				//todo : 组内广播签名公钥
				log.Printf("OMSP send sign pub key to group members, spk=%v...\n", GetPubKeyPrefix(msg.SignPK))

				if !PROC_TEST_MODE {
					log.Printf("OMSP call network service SendSignPubKey...\n")
					SendSignPubKey(msg)
				} else {
					log.Printf("test mode, call OnMessageSignPK direct...\n")
					for _, proc := range p.GroupProcs {
						proc.OnMessageSignPK(msg)
					}
				}
			}

		} else {
			panic("Processor::OMSP failed, aggr key error.")
		}
	}

	log.Printf("prov(%v) end OMSP, sender=%v.\n", p.getPrefix(), GetIDPrefix(spm.SI.GetID()))
	return
}

//收到组内成员发给我的组成员签名公钥消息
func (p *Processor) OnMessageSignPK(spkm ConsensusSignPubKeyMessage) {
	log.Printf("proc(%v) begin OMSPK, sender=%v, dummy_gid=%v...\n", p.getPrefix(), GetIDPrefix(spkm.SI.GetID()), GetIDPrefix(spkm.DummyID))

	gc := p.joiningGroups.GetGroup(spkm.DummyID)
	if gc == nil {
		log.Printf("OMSPK failed, local node not found joining group with dummy id=%v.\n", GetIDPrefix(spkm.DummyID))
		return
	}
	if gc.gis.GenHash() != spkm.GISHash {
		log.Printf("OMSPK failed, gisHash diff.\n")
		return
	}
	if !spkm.VerifyGISSign(spkm.SignPK) {
		panic("OMSP verify GISSign with sign pub key failed.")
	}

	log.Printf("before SignPKMessage already exist mem sign pks=%v.\n", len(gc.node.memberPubKeys))
	result := gc.SignPKMessage(&spkm)
	log.Printf("after SignPKMessage exist mem sign pks=%v, result=%v.\n", len(gc.node.memberPubKeys), result)
	if result == 1 { //收到所有组成员的签名公钥
		jg := gc.GetGroupInfo()

		jg.setGroupSecretHeight(p.MainChain.QueryTopBlock().Height)

		if jg.GroupID.IsValid() && jg.SignKey.IsValid() {
			p.joinGroup(jg, true)
			log.Printf("SUCCESS INIT GROUP: gid=%v, gpk=%v.\n", GetIDPrefix(jg.GroupID), GetPubKeyPrefix(jg.GroupPK))
			{
				var msg ConsensusGroupInitedMessage
				ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
				msg.GI.GIS = gc.gis
				msg.GI.GroupID = jg.GroupID
				msg.GI.GroupPK = jg.GroupPK

				msg.GenSign(ski, &msg)

				if !PROC_TEST_MODE {

					log.Printf("call network service BroadcastGroupInfo...\n")
					BroadcastGroupInfo(msg)
				} else {
					log.Printf("test mode, call OnMessageGroupInited direct...\n")
					for _, proc := range p.GroupProcs {
						proc.OnMessageGroupInited(msg)
					}
				}
			}
		} else {
			panic("Processor::OnMessageSharePiece failed, aggr key error.")
		}
		p.joiningGroups.RemoveGroup(spkm.DummyID)
	}

	log.Printf("proc(%v) end OMSPK, sender=%v, dummy gid=%v.\n", p.getPrefix(), GetIDPrefix(spkm.SI.GetID()), GetIDPrefix(spkm.DummyID))
	return
}

//全网节点收到某组已初始化完成消息（在一个时间窗口内收到该组51%成员的消息相同，才确认上链）
//最终版本修改为父亲节点进行验证（51%）和上链
//全网节点处理函数->to do : 调整为父亲组节点处理函数
func (p *Processor) OnMessageGroupInited(gim ConsensusGroupInitedMessage) {
	log.Printf("proc(%v) begin OMGIED, sender=%v, dummy_gid=%v, gid=%v, gpk=%v...\n", p.getPrefix(),
		GetIDPrefix(gim.SI.GetID()), GetIDPrefix(gim.GI.GIS.DummyID), GetIDPrefix(gim.GI.GroupID), GetPubKeyPrefix(gim.GI.GroupPK))

	dummyId := gim.GI.GIS.DummyID


	if gim.SI.DataHash != gim.GI.GenHash() {
		panic("grm gis hash diff")
	}
	parentGroup := p.getGroup(gim.GI.GIS.ParentID)
	if !parentGroup.CastQualified(gim.GI.GIS.TopHeight) {
		log.Printf("OMGI parent group has no qualify to cast group. gid=%v, height=%v\n", GetIDPrefix(parentGroup.GroupID), gim.GI.GIS.TopHeight)
		return
	}
	gpk := parentGroup.GroupPK
	if !groupsig.VerifySig(gpk, gim.GI.GIS.GenHash().Bytes(), gim.GI.GIS.Signature) {
		log.Printf("OMGIED verify parent groupsig fail! dummyId=%v\n", GetIDPrefix(dummyId))
		return
	}
	topHeight := p.MainChain.QueryTopBlock().Height

	initingGroup := p.globalGroups.GetInitingGroup(dummyId)
	if initingGroup == nil {
		log.Printf("initingGroup not found!dummyId=%v\n", GetIDPrefix(dummyId))
		return
	}
	if !initingGroup.MemberExist(gim.SI.SignMember) {
		return
	}

	if initingGroup.gis.GenHash() != gim.GI.GIS.GenHash() {
		log.Printf("gisHash diff from initingGroup, dummyId=%v\n", GetIDPrefix(dummyId))
		return
	}
	if !gim.GI.GIS.checkMemberHash(initingGroup.mems) {
		panic("gim member hash diff!")
	}

	result := p.globalGroups.GroupInitedMessage(&gim.GI, gim.SI.SignMember, topHeight)

	log.Printf("proc(%v) OMGIED globalGroups.GroupInitedMessage result=%v.\n", p.getPrefix(), result)

	switch result {
	case INIT_SUCCESS: //收到组内相同消息>=阈值，可上链
		staticGroup := NewSGIFromStaticGroupSummary(&gim.GI, initingGroup)
		log.Printf("OMGIED SUCCESS accept a new group, gid=%v, gpk=%v, beginHeight=%v, dismissHeight=%v.\n", GetIDPrefix(gim.GI.GroupID), GetPubKeyPrefix(gim.GI.GroupPK), staticGroup.BeginHeight, staticGroup.DismissHeight)
		add := p.globalGroups.AddStaticGroup(staticGroup)
		log.Printf("OMGIED Add to Global static groups, result=%v, groups=%v.\n", add, p.globalGroups.GetGroupSize())

		if add {
			p.groupManager.AddGroupOnChain(staticGroup, false)

			if p.IsMinerGroup(gim.GI.GroupID) && p.GetBlockContext(gim.GI.GroupID) == nil {
				p.prepareForCast(staticGroup)
			}
		}

		p.globalGroups.removeInitingGroup(initingGroup.gis.DummyID)

	case INIT_FAIL: //该组初始化异常，且无法恢复
		p.globalGroups.removeInitingGroup(initingGroup.gis.DummyID)

	case INITING:
		//继续等待下一包数据
	}
	log.Printf("proc(%v) end OMGIED, sender=%v...\n", p.getPrefix(), GetIDPrefix(gim.SI.GetID()))
	return
}


func (p *Processor) OnMessageCreateGroupRaw(msg ConsensusCreateGroupRawMessage)  {
	log.Printf("Proc(%v) OMCGR begin, dummyId=%v sender=%v\n", p.getPrefix(), GetIDPrefix(msg.GI.DummyID), GetIDPrefix(msg.SI.SignMember))

	if p.GetMinerID().IsEqual(msg.SI.SignMember) {
		return
	}
	gpk := p.GetMemberSignPubKey(GroupMinerID{gid:msg.GI.ParentID, uid:msg.SI.SignMember})
	if !gpk.IsValid() {
		return
	}
	if !msg.SI.VerifySign(gpk) {
		return
	}
	if p.groupManager.OnMessageCreateGroupRaw(&msg) {
		signMsg := &ConsensusCreateGroupSignMessage{
			GI: msg.GI,
			Launcher: msg.SI.SignMember,
		}
		signMsg.GenSign(SecKeyInfo{ID: p.GetMinerID(), SK: p.getSignKey(msg.GI.ParentID)}, signMsg)
		log.Printf("OMCGR SendCreateGroupSignMessage... ")
		SendCreateGroupSignMessage(signMsg)
	}
}

func (p *Processor) OnMessageCreateGroupSign(msg ConsensusCreateGroupSignMessage)  {
	log.Printf("Proc(%v) OMCGS begin, dummyId=%v, sender=%v\n", p.getPrefix(), GetIDPrefix(msg.GI.DummyID), GetIDPrefix(msg.SI.SignMember))
	if p.GetMinerID().IsEqual(msg.SI.SignMember) {
		return
	}
	mpk := p.GetMemberSignPubKey(GroupMinerID{gid:msg.GI.ParentID, uid:msg.SI.SignMember})
	if !mpk.IsValid() {
		return
	}
	if !msg.SI.VerifySign(mpk) {
		return
	}
	if p.groupManager.OnMessageCreateGroupSign(&msg) {
		gpk := p.getGroupPubKey(msg.GI.ParentID)
		if !groupsig.VerifySig(gpk, msg.SI.DataHash.Bytes(), msg.GI.Signature) {
			log.Printf("Proc(%v) OMCGS verify group sign fail\n", p.getPrefix())
			return
		}
		creatingGroup := p.groupManager.creatingGroups.getCreatingGroup(msg.GI.DummyID)
		mems := make([]PubKeyInfo, len(creatingGroup.ids))
		pubkeys := p.groupManager.getPubkeysByIds(creatingGroup.ids)
		if len(pubkeys) != len(creatingGroup.ids) {
			panic("get all pubkey failed")
		}
		for i, id := range creatingGroup.ids {
			mems[i] = PubKeyInfo{ID: id, PK: pubkeys[i]}
		}
		initMsg := &ConsensusGroupRawMessage{
			GI: msg.GI,
			MEMS: mems,
		}

		log.Printf("Proc(%v) OMCGS send group init Message\n", p.getPrefix())
		initMsg.GenSign(SecKeyInfo{ID: p.GetMinerID(), SK: p.getMinerInfo().GetDefaultSecKey()}, initMsg)
		SendGroupInitMessage(*initMsg)

		p.groupManager.removeCreatingGroup(msg.GI.DummyID)
	}
}