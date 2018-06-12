package logical

import (
	"common"
	"consensus/groupsig"
	//"consensus/net"
	"core"
	"fmt"
	"time"
	"log"
	"sync/atomic"
	"middleware/types"
)

func (p *Processer) genCastGroupSummary(bh *types.BlockHeader) *CastGroupSummary {
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

func (p *Processer) doVerify(mtype string, msg *ConsensusBlockMessageBase, cgs *CastGroupSummary) {
	bh := &msg.BH
	si := &msg.SI

	sender := GetIDPrefix(si.SignMember)
	logHalfway(mtype, bh.Height, bh.QueueNumber, sender, "doVerify begin")

	log.Printf("%v message bh %v\n", mtype, p.blockPreview(bh))
	log.Printf("%v chain top bh %v\n", mtype, p.blockPreview(p.MainChain.QueryTopBlock()))

	if p.blockOnChain(bh) {
		log.Printf("%v receive block already onchain! , height = %v\n", mtype, bh.Height)
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
		return
	}

	if !p.isCastGroupLeagal(bh, preBH) {
		log.Printf("not the casting group!bh=%v, preBH=%v", bh, preBH)
		return
	}

	bc := p.GetBlockContext(gid.GetHexString())
	if bc == nil {
		log.Printf("[ERROR]blockcontext is nil!, gid=" + GetIDPrefix(gid))
		return
	}

	_, vctx := bc.GetOrNewVerifyContext(bh)

	verifyResult := vctx.UserVerified(bh, si, cgs)
	log.Printf("proc(%v) %v UserVerified result=%v.\n", mtype, p.getPrefix(), verifyResult)
	slot := vctx.GetSlotByQN(int64(bh.QueueNumber))

	logHalfway(mtype, bh.Height, bh.QueueNumber, sender, "UserVerified result:%v", verifyResult)

	switch verifyResult {
	case CBMR_THRESHOLD_SUCCESS:
		log.Printf("proc(%v) %v msg_count reach threshold!\n", mtype, p.getPrefix())

		gpk := p.getGroupPubKey(gid)
		sign := slot.GetGroupSign()
		if !slot.VerifyGroupSign(gpk) { //组签名验证通过
			log.Printf("%v group pub key local check failed, gpk=%v, sign=%v, hash in slot=%v, hash in bh=%v.\n", mtype,
				GetPubKeyPrefix(gpk), GetSignPrefix(sign), GetHashPrefix(slot.BH.Hash), GetHashPrefix(bh.Hash))
			return
		} else {
			//log.Printf("%v group pub key local check OK, gpk=%v, sign=%v, hash in slot=%v, hash in bh=%v.\n", mtype,
			//	GetPubKeyPrefix(gpk), GetSignPrefix(sign), GetHashPrefix(slot.BH.Hash), GetHashPrefix(bh.Hash))
		}
		bh.Signature = sign.Serialize()
		log.Printf("proc(%v) %v SUCCESS CAST GROUP BLOCK, height=%v, qn=%v!!!\n", mtype, p.getPrefix(), bh.Height, bh.QueueNumber)

		if atomic.CompareAndSwapInt32(&slot.SlotStatus, SS_VERIFIED, SS_ONCHAIN) {
			p.SuccessNewBlock(bh, vctx, gid) //上链和组外广播
			//log.Printf("%v remove verifycontext from bccontext! remain size=%v\n", mtype, len(bc.verifyContexts))
		} else {
			log.Printf("%v already broadcast new block! slotStatus=%v\n", mtype, slot.SlotStatus)
		}

	case CBMR_PIECE_NORMAL:
		if atomic.CompareAndSwapInt32(&slot.SlotStatus, SS_WAITING, SS_BRAODCASTED) && !cgs.Castor.IsEqual(p.GetMinerID()) {
			var cvm ConsensusVerifyMessage
			cvm.BH = *bh
			//cvm.GroupID = gId
			cvm.GenSign(SecKeyInfo{p.GetMinerID(), p.getSignKey(gid)})
			if !PROC_TEST_MODE {
				log.Printf("call network service SendVerifiedCast...\n")
				logHalfway(mtype, bh.Height, bh.QueueNumber, sender, "SendVerifiedCast")
				go SendVerifiedCast(&cvm)
			} else {
				log.Printf("proc(%v) OMC BEGIN SEND OnMessageVerify 2 ALL PROCS...\n", p.getPrefix())
				for _, v := range p.GroupProcs {
					v.OnMessageVerify(cvm)
				}
			}

		}
	case CBMR_PIECE_LOSINGTRANS: //交易缺失
		log.Printf("%v lost trans!", mtype)

	}
}

func (p *Processer) verifyCastMessage(mtype string, msg *ConsensusBlockMessageBase) {
	bh := &msg.BH
	si := &msg.SI
	log.Printf("Proc(%v) begin %v, height=%v, qn=%v\n", p.getPrefix(), mtype, bh.Height, bh.QueueNumber)
	logStart(mtype, bh.Height, bh.QueueNumber, GetIDPrefix(si.SignMember), "")

	begin := time.Now()
	defer func() {
		log.Printf("%v begin at %v, cost %v\n", mtype, begin.String(), time.Since(begin).String())
		logEnd(mtype, bh.Height, bh.QueueNumber, GetIDPrefix(si.SignMember))
	}()

	cgs := p.genCastGroupSummary(bh)
	if cgs == nil {
		log.Printf("[ERROR]%v gen castGroupSummary fail!\n", mtype)
		return
	}
	log.Printf("proc(%v) begin %v, group=%v, sender=%v, height=%v, qn=%v, castor=%v...\n", p.getPrefix(), mtype,
		GetIDPrefix(cgs.GroupID), GetIDPrefix(si.GetID()), bh.Height, bh.QueueNumber, GetIDPrefix(cgs.Castor))

	//如果是自己发的, 不处理
	if p.GetMinerID().IsEqual(si.SignMember) {
		log.Printf("%v receive self msg, ingore! \n", mtype)
		return
	}

	outputBlockHeaderAndSign(mtype, bh, si)

	log.Printf("proc(%v) %v verifyCast, sender=%v, height=%v, pre_time=%v...\n", p.getPrefix(), mtype, GetIDPrefix(si.GetID()), cgs.BlockHeight, cgs.PreTime.Format(time.Stamp))
	if !p.verifyCastSign(cgs, si) {
		log.Printf("%v verify failed!\n", mtype)
		return
	}

	p.doVerify(mtype, msg, cgs)

	//case 1: //本地交易缺失
	//	n := bc.UserCasted(ccm.BH, ccm.SI)
	//	log.Printf("proc(%v) OMC UserCasted result=%v.\n", p.getPrefix(), n)
	//	switch n {
	//	case CBMR_THRESHOLD_SUCCESS:
	//		log.Printf("proc(%v) OMC msg_count reach threshold, but local missing trans, still waiting.\n", p.getPrefix())
	//	case CBMR_PIECE:
	//		log.Printf("proc(%v) OMC normal receive verify, but local missing trans, waiting.\n", p.getPrefix())
	//	}
	//case -1:
	//	slot.statusChainFailed()
	//}
	return
}

//收到组内成员的出块消息，出块人（KING）用组分片密钥进行了签名
//有可能没有收到OnMessageCurrent就提前接收了该消息（网络时序问题）
func (p *Processer) OnMessageCast(ccm ConsensusCastMessage) {
	p.verifyCastMessage("OMC", &ccm.ConsensusBlockMessageBase)
}

//收到组内成员的出块验证通过消息（组内成员消息）
func (p *Processer) OnMessageVerify(cvm ConsensusVerifyMessage) {
	p.verifyCastMessage("OMV", &cvm.ConsensusBlockMessageBase)
}

func (p *Processer) triggerFutureVerifyMsg(hash common.Hash) {
	futures := p.getFutureVerifyMsgs(hash)
	if futures == nil || len(futures) == 0 {
		return
	}
	p.removeFutureVerifyMsgs(hash)

	for _, msg := range futures {
		p.doVerify("FUTURE_VERIFY", msg, nil)
	}

}

func (p *Processer) receiveBlock(msg *ConsensusBlockMessage, preBH *types.BlockHeader) bool {
	if p.isCastGroupLeagal(msg.Block.Header, preBH) { //铸块组合法
		result := p.doAddOnChain(&msg.Block)
		log.Printf("OMB onchain result %v\n", result)
		if result == 0 || result == 1 {
			return true
		}
	} else {
		//丢弃该块
		log.Printf("OMB received invalid new block, height = %v.\n", msg.Block.Header.Height)
	}
	return false
}

func (p *Processer) cleanVerifyContext(currentHeight uint64) {
	for _, bc := range p.bcs {
		ctxs := bc.SafeGetVerifyContexts()
		delCtx := make([]*VerifyContext, 0)
		for _, ctx := range ctxs {
			if ctx.ShouldRemove(currentHeight) {
				delCtx = append(delCtx, ctx)
			}
		}
		for _, ctx := range delCtx {
			log.Printf("cleanVerifyContext: ctx.castHeight=%v, ctx.prevHash=%v, ctx.signedMaxQN=%v\n", ctx.castHeight, GetHashPrefix(ctx.prevHash), ctx.signedMaxQN)
		}
		bc.RemoveVerifyContexts(delCtx)
	}
}

//收到铸块上链消息(组外矿工节点处理)
func (p *Processer) OnMessageBlock(cbm ConsensusBlockMessage) {
	begin := time.Now()
	bh := cbm.Block.Header
	logStart("OMB", bh.Height, bh.QueueNumber, GetIDPrefix(cbm.SI.SignMember), "castor=%v", GetIDPrefix(*groupsig.NewIdFromBytes(bh.Castor)))
	defer func() {
		log.Printf("OMB begin at %v, cost %v\n", begin.String(), time.Since(begin).String())
		logEnd("OMB", bh.Height, bh.QueueNumber, GetIDPrefix(cbm.SI.SignMember))
	}()

	if p.MainChain.QueryBlockByHash(cbm.Block.Header.Hash) != nil {
		log.Printf("OMB receive block already on chain! bh=%v\n", p.blockPreview(cbm.Block.Header))
		return
	}
	if p.GetMinerID().IsEqual(cbm.SI.SignMember) {
		fmt.Println("OMB receive self msg, ingored!")
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
		log.Printf("OMB verifyGroupSign result=%v.\n", verify)
		return
	}

	topBH := p.MainChain.QueryTopBlock()

	preHeader := p.MainChain.QueryBlockByHash(block.Header.PreHash)
	if preHeader == nil {
		log.Printf("OMB receive future block!, bh=%v, topHash=%v, topHeight=%v\n", p.blockPreview(block.Header), GetHashPrefix(topBH.Hash), topBH.Height)
		p.addFutureBlockMsg(&cbm)
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
				log.Printf("receive cached future block msg: bh=%v, preHeader=%v\n", msg.Block.Header, preHeader)
				tbh := msg.Block.Header
				logHalfway("OMB", tbh.Height, tbh.QueueNumber, GetIDPrefix(msg.SI.SignMember), "trigger cached future block")
				p.receiveBlock(msg, preHeader)
			}
			p.removeFutureBlockMsgs(preHeader.Hash)
			preHeader = p.MainChain.QueryTopBlock()
		}
	}

	nowTop := p.MainChain.QueryTopBlock()
	if topBH.Hash != nowTop.Hash {
		p.triggerCastCheck()
		p.cleanVerifyContext(nowTop.Height)
	}

	log.Printf("proc(%v) end OMB, group=%v, sender=%v...\n", p.getPrefix(), GetIDPrefix(cbm.GroupID), GetIDPrefix(cbm.SI.GetID()))
	return
}

//新的交易到达通知（用于处理大臣验证消息时缺失的交易）
func (p *Processer) OnMessageNewTransactions(ths []common.Hash) {
	begin := time.Now()
	logStart("OMNT", 0, 0, "", "count=%v,txHash[0]=%v", len(ths), GetHashPrefix(ths[0]))
	defer func() {
		log.Printf("OMNT begin at %v, cost %v\n", begin.String(), time.Since(begin).String())
		logEnd("OMNT", 0, 0, "")
	}()

	log.Printf("proc(%v) begin OMNT, trans count=%v...\n", p.getPrefix(), len(ths))
	if len(ths) > 0 {
		log.Printf("proc(%v) OMNT, first trans=%v.\n", p.getPrefix(), ths[0].Hex())
	}

	for gid, bc := range p.bcs {
		slots := bc.receiveTrans(ths)
		log.Printf("group %v lost trans slot size %v\n", gid, len(slots))
		if len(slots) == 0 {
			continue
		}
		for _, slot := range slots { //对不再缺失交易集的插槽处理
			var sendMessage ConsensusVerifyMessage
			sendMessage.BH = slot.BH
			//sendMessage.GroupID = bc.MinerID.gid
			sendMessage.GenSign(SecKeyInfo{p.GetMinerID(), p.getSignKey(bc.MinerID.gid)})
			if atomic.CompareAndSwapInt32(&slot.SlotStatus, SS_WAITING, SS_BRAODCASTED) {
				log.Printf("call network service SendVerifiedCast...\n")
				logHalfway("OMNT", 0, 0, p.getPrefix(), "SendVerifiedCast")
				go SendVerifiedCast(&sendMessage)
			}
		}
	}

	return
}

///////////////////////////////////////////////////////////////////////////////
//组初始化相关消息
//组初始化的相关消息都用（和组无关的）矿工ID和公钥验签

func (p *Processer) OnMessageGroupInit(grm ConsensusGroupRawMessage) {
	log.Printf("proc(%v) begin OMGI, sender=%v, dummy_gid=%v...\n", p.getPrefix(), GetIDPrefix(grm.SI.GetID()), GetIDPrefix(grm.GI.DummyID))
	p.initLock.Lock()
	locked := true

	//to do : 从链上检查消息发起人（父亲组成员）是否有权限发该消息（鸠兹）
	sgi_info := NewSGIFromRawMessage(grm)
	//p.gg.AddDummyGroup(sgi)
	p.gg.ngg.addInitingGroup(CreateInitingGroup(sgi_info))

	//非组内成员不走后续流程
	if !sgi_info.MemExist(p.GetMinerID()) {
		p.initLock.Unlock()
		locked = false
		return
	}

	gc := p.jgs.ConfirmGroupFromRaw(grm, p.mi)
	if gc == nil {
		panic("Processer::OMGI failed, ConfirmGroupFromRaw return nil.")
	}
	gs := gc.GetGroupStatus()
	log.Printf("OMGI joining group(%v) status=%v.\n", GetIDPrefix(grm.GI.DummyID), gs)
	if gs == GIS_RAW {
		log.Printf("begin GenSharePieces in OMGI...\n")
		shares := gc.GenSharePieces() //生成秘密分享
		log.Printf("proc(%v) end GenSharePieces in OMGI, piece size=%v.\n", p.getPrefix(), len(shares))

		if locked {
			p.initLock.Unlock()
			locked = false
		}

		var spm ConsensusSharePieceMessage
		spm.GISHash = grm.GI.GenHash()
		spm.DummyID = grm.GI.DummyID
		ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
		spm.SI.SignMember = p.GetMinerID()
		for id, piece := range shares {
			if id != "0x0" && piece.IsValid() {
				spm.Dest.SetHexString(id)
				spm.Share = piece
				sb := spm.GenSign(ski)
				log.Printf("OMGI spm.GenSign result=%v.\n", sb)
				log.Printf("OMGI piece to ID(%v), share=%v, pub=%v.\n", GetIDPrefix(spm.Dest), GetSecKeyPrefix(spm.Share.Share), GetPubKeyPrefix(spm.Share.Pub))
				if !PROC_TEST_MODE {
					log.Printf("call network service SendKeySharePiece...\n")
					SendKeySharePiece(spm)
				} else {
					log.Printf("test mode, call OMSP direct...\n")
					dest_proc, ok := p.GroupProcs[spm.Dest.GetHexString()]
					if ok {
						dest_proc.OnMessageSharePiece(spm)
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
	if locked {
		p.initLock.Unlock()
		locked = false
	}
	log.Printf("proc(%v) end OMGI, sender=%v.\n", p.getPrefix(), GetIDPrefix(grm.SI.GetID()))
	return
}

//收到组内成员发给我的秘密分享片段消息
func (p *Processer) OnMessageSharePiece(spm ConsensusSharePieceMessage) {
	log.Printf("proc(%v)begin Processer::OMSP, sender=%v...\n", p.getPrefix(), GetIDPrefix(spm.SI.GetID()))
	p.initLock.Lock()
	locked := true

	gc := p.jgs.ConfirmGroupFromPiece(spm, p.mi)
	if gc == nil {
		if locked {
			p.initLock.Unlock()
			locked = false
		}
		panic("OMSP failed, receive SHAREPIECE msg but gc=nil.\n")
		return
	}
	result := gc.PieceMessage(spm)
	log.Printf("proc(%v) OMSP after gc.PieceMessage, piecc_count=%v, gc result=%v.\n", p.getPrefix(), p.piece_count, result)
	p.piece_count++
	if result < 0 {
		panic("OMSP failed, gc.PieceMessage result less than 0.\n")
	}
	if result == 1 { //已聚合出签名私钥
		jg := gc.GetGroupInfo()
		//这时还没有所有组成员的签名公钥
		if jg.GroupPK.IsValid() && jg.SignKey.IsValid() {
			log.Printf("OMSP SUCCESS gen sign sec key and group pub key, msk=%v, gpk=%v.\n", GetSecKeyPrefix(jg.SignKey), GetPubKeyPrefix(jg.GroupPK))
			{
				ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
				var msg ConsensusSignPubKeyMessage
				msg.GISHash = spm.GISHash
				msg.DummyID = spm.DummyID
				msg.SignPK = *groupsig.NewPubkeyFromSeckey(jg.SignKey)
				msg.GenGISSign(jg.SignKey)
				if !msg.VerifyGISSign(msg.SignPK) {
					panic("verify GISSign with group member sign pub key failed.")
				}

				msg.GenSign(ski)
				//todo : 组内广播签名公钥
				log.Printf("OMSP send sign pub key to group members, spk=%v...\n", GetPubKeyPrefix(msg.SignPK))
				if locked {
					p.initLock.Unlock()
					locked = false
				}
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
			panic("Processer::OMSP failed, aggr key error.")
		}
	}
	if locked {
		p.initLock.Unlock()
		locked = false
	}
	log.Printf("prov(%v) end OMSP, sender=%v.\n", p.getPrefix(), GetIDPrefix(spm.SI.GetID()))
	return
}

//收到组内成员发给我的组成员签名公钥消息
func (p *Processer) OnMessageSignPK(spkm ConsensusSignPubKeyMessage) {
	log.Printf("proc(%v) begin OMSPK, sender=%v, dummy_gid=%v...\n", p.getPrefix(), GetIDPrefix(spkm.SI.GetID()), GetIDPrefix(spkm.DummyID))
	p.initLock.Lock()
	locked := true

	/* 待小熊增加GISSign成员的流化后打开
	if !spkm.VerifyGISSign(spkm.SignPK) {
		panic("OMSP verify GISSign with sign pub key failed.")
	}
	*/

	gc := p.jgs.GetGroup(spkm.DummyID)
	if gc == nil {
		if locked {
			p.initLock.Unlock()
			locked = false
		}
		log.Printf("OMSPK failed, local node not found joining group with dummy id=%v.\n", GetIDPrefix(spkm.DummyID))
		return
	}
	log.Printf("before SignPKMessage already exist mem sign pks=%v.\n", len(gc.node.m_sign_pks))
	result := gc.SignPKMessage(spkm)
	log.Printf("after SignPKMessage exist mem sign pks=%v, result=%v.\n", len(gc.node.m_sign_pks), result)
	if result == 1 { //收到所有组成员的签名公钥
		jg := gc.GetGroupInfo()
		if jg.GroupID.IsValid() && jg.SignKey.IsValid() {
			p.addInnerGroup(jg)
			log.Printf("SUCCESS INIT GROUP: gid=%v, gpk=%v.\n", GetIDPrefix(jg.GroupID), GetPubKeyPrefix(jg.GroupPK))
			{
				var msg ConsensusGroupInitedMessage
				ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
				msg.GI.GIS = gc.gis
				msg.GI.GroupID = jg.GroupID
				msg.GI.GroupPK = jg.GroupPK
				var mems []PubKeyInfo
				for _, v := range gc.mems {
					mems = append(mems, v)
				}
				msg.GI.Members = mems
				pTop := p.MainChain.QueryTopBlock()
				if 0 == pTop.Height {
					msg.GI.BeginHeight = 1
				} else {
					msg.GI.BeginHeight = pTop.Height + uint64(GROUP_INIT_IDLE_HEIGHT)
				}
				msg.GenSign(ski)
				if locked {
					p.initLock.Unlock()
					locked = false
				}
				if !PROC_TEST_MODE {
					//组写入组链 add by 小熊
					members := make([]core.Member, 0)
					for _, miner := range mems {
						member := core.Member{Id: miner.ID.Serialize(), PubKey: miner.PK.Serialize()}
						members = append(members, member)
					}
					group := core.Group{Id: msg.GI.GroupID.Serialize(), Members: members, PubKey: msg.GI.GroupPK.Serialize(), Parent: msg.GI.GIS.ParentID.Serialize()}
					e := p.GroupChain.AddGroup(&group, nil, nil)
					if e != nil {
						log.Printf("group inited add group error:%s\n", e.Error())
					} else {
						log.Printf("group inited add group success. count: %d, now: %d\n", core.GroupChainImpl.Count(), len(core.GroupChainImpl.GetAllGroupID()))
					}
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
			panic("Processer::OnMessageSharePiece failed, aggr key error.")
		}
	}
	if locked {
		p.initLock.Unlock()
		locked = false
	}
	log.Printf("proc(%v) end OMSPK, sender=%v, dummy gid=%v.\n", p.getPrefix(), GetIDPrefix(spkm.SI.GetID()), GetIDPrefix(spkm.DummyID))
	return
}

//全网节点收到某组已初始化完成消息（在一个时间窗口内收到该组51%成员的消息相同，才确认上链）
//最终版本修改为父亲节点进行验证（51%）和上链
//全网节点处理函数->to do : 调整为父亲组节点处理函数
func (p *Processer) OnMessageGroupInited(gim ConsensusGroupInitedMessage) {
	log.Printf("proc(%v) begin OMGIED, sender=%v, dummy_gid=%v, gid=%v, gpk=%v...\n", p.getPrefix(),
		GetIDPrefix(gim.SI.GetID()), GetIDPrefix(gim.GI.GIS.DummyID), GetIDPrefix(gim.GI.GroupID), GetPubKeyPrefix(gim.GI.GroupPK))

	p.initLock.Lock()
	defer p.initLock.Unlock()

	var ngmd NewGroupMemberData
	ngmd.h = gim.GI.GIS.GenHash()
	ngmd.gid = gim.GI.GroupID
	ngmd.gpk = gim.GI.GroupPK
	var mid GroupMinerID
	mid.gid = gim.GI.GIS.DummyID
	mid.uid = gim.SI.SignMember
	result := p.gg.GroupInitedMessage(mid, ngmd)
	p.inited_count++
	log.Printf("proc(%v) OMGIED gg.GroupInitedMessage result=%v, inited_count=%v.\n", p.getPrefix(), result, p.inited_count)
	if result < 0 {
		panic("OMGIED gg.GroupInitedMessage failed because of return value.")
	}
	switch result {
	case 1: //收到组内相同消息>=阈值，可上链
		log.Printf("OMGIED SUCCESS accept a new group, gid=%v, gpk=%v.\n", GetIDPrefix(gim.GI.GroupID), GetPubKeyPrefix(gim.GI.GroupPK))
		b := p.gg.AddGroup(gim.GI)
		log.Printf("OMGIED Add to Global static groups, result=%v, groups=%v.\n", b, p.gg.GetGroupSize())

		//检查是否写入配置文件
		if p.save_data == 1 {
			p.Save()
		}

		//上链
		members := make([]core.Member, 0)
		for _, miner := range gim.GI.Members {
			member := core.Member{Id: miner.ID.Serialize(), PubKey: miner.PK.Serialize()}
			members = append(members, member)
		}
		group := core.Group{Id: gim.GI.GroupID.Serialize(), Members: members, PubKey: gim.GI.GroupPK.Serialize(), Parent: gim.GI.GIS.ParentID.Serialize()}
		e := p.GroupChain.AddGroup(&group, nil, nil)
		if e != nil {
			log.Printf("OMGIED group inited add group error:%s\n", e.Error())
		} else {
			log.Printf("OMGIED group inited add group success. count: %d, now: %d\n", core.GroupChainImpl.Count(), len(core.GroupChainImpl.GetAllGroupID()))
		}

		if p.IsMinerGroup(gim.GI.GroupID) && p.GetBlockContext(gim.GI.GroupID.GetHexString()) == nil {
			bc := new(BlockContext)
			bc.Proc = p
			bc.Init(GroupMinerID{gim.GI.GroupID, p.GetMinerID()})
			sgi, err := p.gg.GetGroupByID(gim.GI.GroupID)
			if err != nil {
				panic("OMGIED GetGroupByID failed.\n")
			}
			bc.pos = sgi.GetMinerPos(p.GetMinerID())
			log.Printf("OMGIED current ID in group pos=%v.\n", bc.pos)
			//to do:只有自己属于这个组的节点才需要调用AddBlockConext
			b = p.AddBlockContext(bc)
			log.Printf("(proc:%v) OMGIED Add BlockContext result = %v, bc_size=%v.\n", p.getPrefix(), b, len(p.bcs))
			//to do : 上链已初始化的组
			//to do ：从待初始化组中删除

			p.Ticker.RegisterRoutine(bc.getKingCheckRoutineName(), bc.kingTickerRoutine, uint32(MAX_USER_CAST_TIME))
			p.triggerCastCheck()
		}
		//
		//log.Printf("begin sleeping 5 seconds, now=%v...\n", time.Now().Format(time.Stamp))
		//sleep_d, err := time.ParseDuration("5s")
		//if err == nil {
		//	time.Sleep(sleep_d)
		//} else {
		//	panic("time.ParseDuration 5s failed.")
		//}
		//log.Printf("end sleeping, now=%v.\n", time.Now().Format(time.Stamp))
		//拉取当前最高块
		//if !PROC_TEST_MODE {
		//	top_bh := p.MainChain.QueryTopBlock()
		//	if top_bh == nil {
		//		panic("QueryTopBlock failed")
		//	} else {
		//		log.Printf("top height on chain=%v.\n", top_bh.Height)
		//	}
		//	var g_sign groupsig.Signature
		//	if g_sign.Deserialize(top_bh.Signature) != nil {
		//		panic("OMGIED group sign Deserialize failed.")
		//	}
		//	broadcast, ccm := p.checkCastingGroup(gim.GI.GroupID, g_sign, top_bh.Height, top_bh.CurTime, top_bh.Hash)
		//	log.Printf("checkCastingGroup, current proc being casting group=%v.", broadcast)
		//	if broadcast {
		//		log.Printf("OMB: id=%v, data hash=%v, sign=%v.\n",
		//			GetIDPrefix(ccm.SI.GetID()), GetHashPrefix(ccm.SI.DataHash), GetSignPrefix(ccm.SI.DataSign))
		//		log.Printf("OMB call network service SendCurrentGroupCast...\n")
		//		SendCurrentGroupCast(&ccm) //通知所有组员“我们组成为当前铸块组”
		//	}
		//}
	case -1: //该组初始化异常，且无法恢复
		//to do : 从待初始化组中删除
	case 0:
		//继续等待下一包数据
	}
	log.Printf("proc(%v) end OMGIED, sender=%v...\n", p.getPrefix(), GetIDPrefix(gim.SI.GetID()))
	return
}
