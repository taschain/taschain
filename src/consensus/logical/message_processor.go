package logical

import (
	"common"
	"consensus/groupsig"
	//"consensus/net"
	"core"
	"fmt"
	"time"
	"log"
)

//收到成为当前铸块组消息
func (p *Processer) OnMessageCurrent(ccm ConsensusCurrentMessage) {
	beginTime := time.Now()
	p.castLock.Lock()
	defer p.castLock.Unlock()

	log.Printf("proc(%v) begin OMCur, sender=%v, time=%v, height=%v...\n", p.getPrefix(),
		GetIDPrefix(ccm.SI.GetID()), beginTime.Format(time.Stamp), ccm.BlockHeight)
	var gid groupsig.ID
	if gid.Deserialize(ccm.GroupID) != nil {
		panic("Processer::OMCur failed, reason=group id Deserialize.")
	}

	//如果是自己发的, 不处理
	if p.GetMinerID().IsEqual(ccm.SI.SignMember) {
		log.Printf("OMC receive self msg, ingore! \n")
		return
	}
	//
	//topBH := p.MainChain.QueryTopBlock()
	//if topBH.Height == ccm.BlockHeight && topBH.PreHash == ccm.PreHash {	//已经在链上了
	//	log.Printf("OMCur block height already on chain!, ingore!, topBlockHeight=%v, ccm.Height=%v, topHash=%v, ccm.PreHash=%v", topBH.Height, ccm.BlockHeight, GetHashPrefix(topBH.PreHash), GetHashPrefix(ccm.PreHash))
	//	return
	//}


	var cgs CastGroupSummary
	cgs.GroupID = gid
	cgs.PreHash = ccm.PreHash
	cgs.PreTime = ccm.PreTime
	cgs.BlockHeight = ccm.BlockHeight
	log.Printf("OMCur::SIGN_INFO: id=%v, data hash=%v, sign=%v.\n",
		GetIDPrefix(ccm.SI.GetID()), GetHashPrefix(ccm.SI.DataHash), GetSignPrefix(ccm.SI.DataSign))
	bc, cast := p.beingCastGroup(&cgs, ccm.SI)
	if bc == nil {
		log.Printf("proc(%v) OMCur can't get valid bc, ignore message.\n", p.getPrefix())
		return
	}
	if !cast {
		log.Println("OMCur being castgroup failed!")
		return
	}

	log.Printf("OMCur after beingCastGroup, bc.height=%v, first=%v.\n", bc.CastHeight, bc.ConsensusStatus)
	if bc != nil {
		//switched := bc.Switch2Height(cgs)
		//if !switched {
		//	log.Printf("bc::Switch2Height failed, ignore message.\n")
		//	return
		//}
		//if first { //第一次收到“当前组成为铸块组”消息
		//	ccm_local := ccm
		//	ccm_local.GenSign(SecKeyInfo{p.GetMinerID(), p.getSignKey(gid)})
		//	if locked {
		//		p.castLock.Unlock()
		//		locked = false
		//	}
		//	if !PROC_TEST_MODE {
		//		log.Printf("call network service SendCurrentGroupCast...\n")
		//		SendCurrentGroupCast(&ccm_local)
		//	} else {
		//		log.Printf("proc(%v) OMCur BEGIN SEND OnMessageCurrent 2 ALL PROCS...\n", p.getPrefix())
		//		for _, v := range p.GroupProcs {
		//			v.OnMessageCurrent(ccm_local)
		//		}
		//	}
		//}
	}
	log.Printf("proc(%v) end OMCur, time=%v. cost=%v\n", p.getPrefix(), time.Now().Format(time.Stamp), time.Since(beginTime).String())
	return
}

//收到组内成员的出块消息，出块人（KING）用组分片密钥进行了签名
//有可能没有收到OnMessageCurrent就提前接收了该消息（网络时序问题）
func (p *Processer) OnMessageCast(ccm ConsensusCastMessage) {
	begin := time.Now()
	defer func() {
		log.Printf("OMC begin at %v, cost %v\n", begin.String(), time.Since(begin).String())
	}()
	p.castLock.Lock()
	locked := true
	defer func() {
		if locked {
			p.castLock.Unlock()
		}
	}()


	var g_id groupsig.ID
	if g_id.Deserialize(ccm.BH.GroupId) != nil {
		panic("OMC Deserialize group_id failed")
	}
	var castor groupsig.ID
	castor.Deserialize(ccm.BH.Castor)
	log.Printf("proc(%v) begin OMC, group=%v, sender=%v, height=%v, qn=%v, castor=%v...\n", p.getPrefix(),
		GetIDPrefix(g_id), GetIDPrefix(ccm.SI.GetID()), ccm.BH.Height, ccm.BH.QueueNumber, GetIDPrefix(castor))

	//如果是自己发的, 不处理
	if p.GetMinerID().IsEqual(ccm.SI.SignMember) {
		log.Printf("OMC receive self msg, ingore! \n")
		return
	}

	log.Printf("OMCCCCC message bh %v\n", ccm.BH)
	log.Printf("OMCCCCC chain top bh %v\n", p.MainChain.QueryTopBlock())
	outputBlockHeaderAndSign("castBlock", &ccm.BH, &ccm.SI)

	exist := p.MainChain.QueryBlockByHeight(ccm.BH.Height)
	if exist != nil && exist.Hash == ccm.BH.Hash && exist.PreHash == ccm.BH.PreHash {	//已经上链
		log.Printf("OMC receive block already onchain! , height = %v\n", exist.Height)
		return
	}
	//
	//pre := p.MainChain.QueryBlockByHeight(ccm.BH.Height - 1)
	//if pre != nil && pre.Hash != ccm.BH.PreHash {
	//	log.Printf("OMC recevie error block, chain pre blockheader=%v", pre)
	//	p.castLock.Unlock()
	//	return
	//}

	log.Printf("proc(%v) OMC rece hash=%v.\n", p.getPrefix(), GetHashPrefix(ccm.SI.DataHash))
	var cgs CastGroupSummary
	cgs.BlockHeight = ccm.BH.Height
	cgs.GroupID = g_id
	cgs.PreHash = ccm.BH.PreHash
	cgs.PreTime = ccm.BH.PreTime
	bc, cast := p.beingCastGroup(&cgs, ccm.SI)
	if bc == nil {
		log.Printf("proc(%v) OMC can't get valid bc, ignore message.\n", p.getPrefix())
		return
	}
	if !cast {
		log.Println("being castgroup failed!")
		return
	}

	log.Printf("OMC after beingCastGroup, bc.Height=%v, first=%v.\n", bc.CastHeight, bc.ConsensusStatus)
	//switched := bc.Switch2Height(cgs)
	//if !switched {
	//	log.Printf("bc::Switch2Height failed, ignore message.\n")
	//	if locked {
	//		p.castLock.Unlock()
	//		locked = false
	//	}
	//	return
	//}

	if !bc.IsCasting() { //当前没有在组铸块中
		log.Printf("proc(%v) OMC failed, group not in cast.\n", p.getPrefix())
		return
	}

	var ccr int8
	var lost_trans_list []common.Hash

	slot := bc.getSlotByQN(int64(ccm.BH.QueueNumber))
	if slot != nil {
		if slot.IsFailed() {
			log.Printf("proc(%v) OMC slot irreversible failed, ignore message.\n", p.getPrefix())
			return
		}
		if slot.isAllTransExist() { //所有交易都已本地存在
			ccr = 0
		} else {
			if !PROC_TEST_MODE {
				lost_trans_list, ccr, _, _ = p.MainChain.VerifyCastingBlock(ccm.BH)
				log.Printf("proc(%v) OMC chain check result=%v, lost_count=%v.\n", p.getPrefix(), ccr, len(lost_trans_list))
				//slot.InitLostingTrans(lost_trans_list)
			}
		}
	}

	cs := GenConsensusSummary(ccm.BH)
	switch ccr {
	case 0: //主链验证通过
		n := bc.UserCasted(ccm.BH, ccm.SI)
		log.Printf("proc(%v) OMC UserCasted result=%v.\n", p.getPrefix(), n)
		switch n {
		case CBMR_THRESHOLD_SUCCESS:
			log.Printf("proc(%v) OMC msg_count reach threshold!\n", p.getPrefix())
			b := bc.VerifyGroupSign(cs, p.getGroupPubKey(g_id))
			log.Printf("proc(%v) OMC VerifyGroupSign=%v.\n", p.getPrefix(), b)
			if b { //组签名验证通过
				slot := bc.getSlotByQN(cs.QueueNumber)
				if slot == nil {
					panic("getSlotByQN return nil.")
				} else {
					sign := slot.GetGroupSign()
					gpk := p.getGroupPubKey(g_id)
					if !slot.VerifyGroupSign(gpk) {
						log.Printf("OMC group pub key local check failed, gpk=%v, sign=%v, hash in slot=%v, hash in bh=%v.\n",
							GetPubKeyPrefix(gpk), GetSignPrefix(sign), GetHashPrefix(slot.BH.Hash), GetHashPrefix(ccm.BH.Hash))
						panic("OMC group pub key local check failed")
					} else {
						log.Printf("OMC group pub key local check OK, gpk=%v, sign=%v, hash in slot=%v, hash in bh=%v.\n",
							GetPubKeyPrefix(gpk), GetSignPrefix(sign), GetHashPrefix(slot.BH.Hash), GetHashPrefix(ccm.BH.Hash))
					}
					ccm.BH.Signature = sign.Serialize()
					log.Printf("OMC BH hash=%v, update group sign data=%v.\n", GetHashPrefix(ccm.BH.Hash), GetSignPrefix(sign))

				}
				log.Printf("proc(%v) OMC SUCCESS CAST GROUP BLOCK, height=%v, qn=%v!!!\n", p.getPrefix(), ccm.BH.Height, cs.QueueNumber)
				p.SuccessNewBlock(&ccm.BH, g_id) //上链和组外广播
			} else {
				panic("proc OMC VerifyGroupSign failed.")
			}
		case CBMR_PIECE:
			var cvm ConsensusVerifyMessage
			cvm.BH = ccm.BH
			//cvm.GroupID = g_id
			cvm.GenSign(SecKeyInfo{p.GetMinerID(), p.getSignKey(g_id)})
			if ccm.SI.SignMember.GetHexString() == p.GetMinerID().GetHexString() { //local node is KING
				equal := cvm.SI.IsEqual(ccm.SI)
				if !equal {
					log.Printf("proc(%v) cur prov is KING, but cast sign and verify sign diff.\n", p.getPrefix())
					log.Printf("proc(%v) cast sign: id=%v, hash=%v, sign=%v.\n", p.getPrefix(), GetIDPrefix(ccm.SI.SignMember), ccm.SI.DataHash.Hex(), ccm.SI.DataSign.GetHexString())
					log.Printf("proc(%v) verify sign: id=%v, hash=%v, sign=%v.\n", p.getPrefix(), GetIDPrefix(cvm.SI.SignMember), cvm.SI.DataHash.Hex(), cvm.SI.DataSign.GetHexString())
					panic("cur prov is KING, but cast sign and verify sign diff.")
				}
			}
			if locked {
				p.castLock.Unlock()
				locked = false
			}
			if !PROC_TEST_MODE {
				log.Printf("call network service SendVerifiedCast...\n")
				go SendVerifiedCast(&cvm)
			} else {
				log.Printf("proc(%v) OMC BEGIN SEND OnMessageVerify 2 ALL PROCS...\n", p.getPrefix())
				for _, v := range p.GroupProcs {
					v.OnMessageVerify(cvm)
				}
			}

		}
	case 1: //本地交易缺失
		n := bc.UserCasted(ccm.BH, ccm.SI)
		log.Printf("proc(%v) OMC UserCasted result=%v.\n", p.getPrefix(), n)
		switch n {
		case CBMR_THRESHOLD_SUCCESS:
			log.Printf("proc(%v) OMC msg_count reach threshold, but local missing trans, still waiting.\n", p.getPrefix())
		case CBMR_PIECE:
			log.Printf("proc(%v) OMC normal receive verify, but local missing trans, waiting.\n", p.getPrefix())
		}
	case -1:
		slot.statusChainFailed()
	}
	log.Printf("proc(%v) end OMC.\n", p.getPrefix())
	return
}

//收到组内成员的出块验证通过消息（组内成员消息）
func (p *Processer) OnMessageVerify(cvm ConsensusVerifyMessage) {
	begin := time.Now()
	defer func() {
		log.Printf("OMV begin at %v, cost %v\n", begin.String(), time.Since(begin).String())
	}()

	p.castLock.Lock()
	locked := true
	defer func() {
		if locked {
			p.castLock.Unlock()
		}
	}()


	var g_id groupsig.ID
	if g_id.Deserialize(cvm.BH.GroupId) != nil {
		panic("OMV Deserialize group_id failed")
	}
	var castor groupsig.ID
	castor.Deserialize(cvm.BH.Castor)
	log.Printf("proc(%v) begin OMV, group=%v, sender=%v, height=%v, qn=%v, rece hash=%v castor=%v...\n", p.getPrefix(),
		GetIDPrefix(g_id), GetIDPrefix(cvm.SI.GetID()), cvm.BH.Height, cvm.BH.QueueNumber, cvm.SI.DataHash.Hex(), GetIDPrefix(castor))

	//如果是自己发的, 不处理
	if p.GetMinerID().IsEqual(cvm.SI.SignMember) {
		log.Printf("OMC receive self msg, ingore! \n")
		return
	}

	log.Printf("OMVVVVVV message bh %v\n", cvm.BH)
	log.Printf("OMVVVVVV message bh hash %v\n", GetHashPrefix(cvm.BH.Hash))
	log.Printf("OMVVVVVV chain top bh %v\n", p.MainChain.QueryTopBlock())
	outputBlockHeaderAndSign("castBlock", &cvm.BH, &cvm.SI)

	exist := p.MainChain.QueryBlockByHeight(cvm.BH.Height)
	if exist != nil && exist.Hash == cvm.BH.Hash && exist.PreHash == cvm.BH.PreHash {	//已经上链
		log.Printf("OMC receive block already onchain! , height = %v\n", exist.Height)
		return
	}

	//pre := p.MainChain.QueryBlockByHeight(cvm.BH.Height - 1)
	//if pre != nil && pre.Hash != cvm.BH.PreHash {
	//	log.Printf("OMC recevie error block, chain pre blockheader=%v", pre)
	//	p.castLock.Unlock()
	//	return
	//}

	var cgs CastGroupSummary
	cgs.BlockHeight = cvm.BH.Height
	cgs.GroupID = g_id
	cgs.PreHash = cvm.BH.PreHash
	cgs.PreTime = cvm.BH.PreTime
	bc, cast := p.beingCastGroup(&cgs, cvm.SI)
	if bc == nil {
		log.Printf("proc(%v) OMV can't get valid bc, ignore message.\n", p.getPrefix())
		return
	}
	if !cast {
		log.Println("OMV being castgroup failed!")
		return
	}

	log.Printf("OMV after beingCastGroup, bc.Height=%v, first=%v.\n", bc.CastHeight, bc.ConsensusStatus)

	//switched := bc.Switch2Height(cgs)
	//if !switched {
	//	log.Printf("bc::Switch2Height failed, ignore message.\n")
	//	if locked {
	//		p.castLock.Unlock()
	//		locked = false
	//	}
	//	return
	//}

	if !bc.IsCasting() { //当前没有在组铸块中
		log.Printf("proc(%v) OMV failed, group not in cast, ignore message.\n", p.getPrefix())
		return
	}

	var ccr int8 = 0
	var lost_trans_list []common.Hash

	slot := bc.getSlotByQN(int64(cvm.BH.QueueNumber))
	if slot != nil {
		if slot.IsFailed() {
			log.Printf("proc(%v) OMV slot irreversible failed, ignore message.\n", p.getPrefix())
			return
		}
		if slot.isAllTransExist() { //所有交易都已本地存在
			ccr = 0
		} else {
			if !PROC_TEST_MODE {
				lost_trans_list, ccr, _, _ = p.MainChain.VerifyCastingBlock(cvm.BH)
				log.Printf("proc(%v) OMV chain check result=%v, lost_trans_count=%v.\n", p.getPrefix(), ccr, len(lost_trans_list))
				//slot.LostingTrans(lost_trans_list)
			}
		}
	}

	cs := GenConsensusSummary(cvm.BH)
	switch ccr { //链检查结果
	case 0: //验证通过
		n := bc.UserVerified(cvm.BH, cvm.SI)
		log.Printf("proc(%v) OMV UserVerified result=%v.\n", p.getPrefix(), n)
		switch n {
		case CBMR_THRESHOLD_SUCCESS:
			log.Printf("proc(%v) OMV msg_count reach threshold!\n", p.getPrefix())
			b := bc.VerifyGroupSign(cs, p.getGroupPubKey(g_id))
			log.Printf("proc(%v) OMV VerifyGroupSign=%v.\n", p.getPrefix(), b)
			if b { //组签名验证通过
				log.Printf("proc(%v) OMV SUCCESS CAST GROUP BLOCK, height=%v, qn=%v.!!!\n", p.getPrefix(), cvm.BH.Height, cvm.BH.QueueNumber)
				slot := bc.getSlotByQN(int64(cvm.BH.QueueNumber))
				if slot == nil {
					panic("getSlotByQN return nil.")
				} else {
					sign := slot.GetGroupSign()

					gpk := p.getGroupPubKey(g_id)
					if !slot.VerifyGroupSign(gpk) {
						log.Printf("OMC group pub key local check failed, gpk=%v, sign=%v, hash in slot=%v, hash in bh=%v.\n",
							GetPubKeyPrefix(gpk), GetSignPrefix(sign), GetHashPrefix(slot.BH.Hash), GetHashPrefix(cvm.BH.Hash))
						panic("OMC group pub key local check failed")
					} else {
						log.Printf("OMC group pub key local check OK, gpk=%v, sign=%v, hash in slot=%v, hash in bh=%v.\n",
							GetPubKeyPrefix(gpk), GetSignPrefix(sign), GetHashPrefix(slot.BH.Hash), GetHashPrefix(cvm.BH.Hash))
					}
					cvm.BH.Signature = sign.Serialize()
					log.Printf("OMV BH hash=%v, update group sign data=%v.\n", GetHashPrefix(cvm.BH.Hash), GetSignPrefix(sign))
				}
				log.Printf("proc(%v) OMV SUCCESS CAST GROUP BLOCK!!!\n", p.getPrefix())
				p.SuccessNewBlock(&cvm.BH, g_id) //上链和组外广播

			} else {
				panic("proc OMV VerifyGroupSign failed.")
			}
		case CBMR_PIECE:
			var send_message ConsensusVerifyMessage
			send_message.BH = cvm.BH
			//send_message.GroupID = g_id
			send_message.GenSign(SecKeyInfo{p.GetMinerID(), p.getSignKey(g_id)})
			if locked {
				p.castLock.Unlock()
				locked = false
			}
			if !PROC_TEST_MODE {
				log.Printf("call network service SendVerifiedCast...\n")
				go SendVerifiedCast(&send_message)
			} else {
				log.Printf("proc(%v) OMV BEGIN SEND OnMessageVerify 2 ALL PROCS...\n", p.getPrefix())
				for _, v := range p.GroupProcs {
					v.OnMessageVerify(send_message)
				}
			}
		}
	case 1: //本地交易缺失
		n := bc.UserVerified(cvm.BH, cvm.SI)
		log.Printf("proc(%v) OnMessageVerify UserVerified result=%v.\n", p.getPrefix(), n)
		switch n {
		case CBMR_THRESHOLD_SUCCESS:
			log.Printf("proc(%v) OMV msg_count reach threshold, but local missing trans, still waiting.\n", p.getPrefix())
		case CBMR_PIECE:
			log.Printf("proc(%v) OMV normal receive verify, but local missing trans, waiting.\n", p.getPrefix())
		}
	case -1: //不可逆异常
		slot.statusChainFailed()
	}
	log.Printf("proc(%v) end OMV.\n", p.getPrefix())
	return
}

//收到铸块上链消息(组外矿工节点处理)
func (p *Processer) OnMessageBlock(cbm ConsensusBlockMessage) *core.Block {
	begin := time.Now()
	defer func() {
		log.Printf("OMB begin at %v, cost %v\n", begin.String(), time.Since(begin).String())
	}()

	p.castLock.Lock()
	locked := true
	defer func() {
		if locked {
			p.castLock.Unlock()
		}
	}()

	var g_id groupsig.ID
	if g_id.Deserialize(cbm.Block.Header.GroupId) != nil {
		panic("OMB Deserialize group_id failed")
	}
	log.Printf("proc(%v) begin OMB, group=%v(bh gid=%v), sender=%v, height=%v, qn=%v...\n", p.getPrefix(),
		GetIDPrefix(cbm.GroupID), GetIDPrefix(g_id), GetIDPrefix(cbm.SI.GetID()), cbm.Block.Header.Height, cbm.Block.Header.QueueNumber)

	if p.GetMinerID().IsEqual(cbm.SI.SignMember) {
		fmt.Println("OMB receive self msg, ingored!")
		return &cbm.Block
	}

	outputBlockHeaderAndSign("castBlock", cbm.Block.Header, &cbm.SI)

	var block *core.Block
	//bc := p.GetBlockContext(cbm.GroupID.GetHexString())
	if p.isBHCastLegal(*cbm.Block.Header, cbm.SI) { //铸块头合法

		//上链
		//onchain := p.MainChain.AddBlockOnChain(&cbm.Block)
		result, future := p.AddOnChain(&cbm.Block)
		log.Printf("OMB onchain result %v, %v\n", result, future)

	} else {
		//丢弃该块
		log.Printf("OMB received invalid new block, height = %v.\n", cbm.Block.Header.Height)
	}

	//收到不合法的块也要做一次检查自己是否属于下一个铸块组, 否则会形成死循环, 没有组出块
	//preHeader := p.MainChain.QueryTopBlock()
	//if preHeader == nil {
	//	panic("cannot find top block header!")
	//}
	//
	//var sign groupsig.Signature
	//if sign.Deserialize(preHeader.Signature) != nil {
	//	panic("OMB group sign Deserialize failed.")
	//}
	//broadcast, ccm := p.checkCastingGroup(cbm.GroupID, sign, preHeader.Height, preHeader.CurTime, preHeader.Hash)
	//if locked {
	//	p.castLock.Unlock()
	//	locked = false
	//}
	//
	//if broadcast {
	//	log.Printf("OMB current proc being casting group...\n")
	//	log.Printf("OMB call network service SendCurrentGroupCast...\n")
	//	go SendCurrentGroupCast(&ccm) //通知所有组员“我们组成为当前铸块组”
	//}
	block = &cbm.Block //返回成功的块

	log.Printf("proc(%v) end OMB, group=%v, sender=%v...\n", p.getPrefix(), GetIDPrefix(cbm.GroupID), GetIDPrefix(cbm.SI.GetID()))
	return block
}

//新的交易到达通知（用于处理大臣验证消息时缺失的交易）
func (p *Processer) OnMessageNewTransactions(ths []common.Hash) {
	p.castLock.Lock()
	locked := true
	log.Printf("proc(%v) begin OMNT, trans count=%v...\n", p.getPrefix(), len(ths))
	if len(ths) > 0 {
		log.Printf("proc(%v) OMNT, first trans=%v.\n", p.getPrefix(), ths[0].Hex())
	}

	bc := p.GetCastingBC() //to do ：实际的场景，可能会返回多个bc，需要处理。（一个矿工加入多个组，考虑现在测试的极端情况，矿工加入了连续出块的2个组）
	if bc != nil {
		log.Printf("OMNT, bc height=%v, status=%v. slots_info=%v.\n", bc.CastHeight, bc.ConsensusStatus, bc.PrintSlotInfo())
		qns := bc.ReceTrans(ths)
		log.Printf("OMNT, bc.ReceTrans result qns_count=%v.\n", len(qns))
		for _, v := range qns { //对不再缺失交易集的插槽处理
			slot := bc.getSlotByQN(int64(v))
			if slot != nil {
				lost_trans_list, result, _, _ := p.MainChain.VerifyCastingBlock(slot.BH)
				log.Printf("OMNT slot (qn=%v) info : lost_trans=%v, mainchain check result=%v.\n", v, len(lost_trans_list), result)
				if len(lost_trans_list) > 0 {
					if locked {
						p.castLock.Unlock()
						locked = false
					}
					panic("OMNT still losting trans on main chain, ERROR.")
				}
				switch result {
				case 0: //验证通过
					var send_message ConsensusVerifyMessage
					send_message.BH = slot.BH
					//send_message.GroupID = bc.MinerID.gid
					send_message.GenSign(SecKeyInfo{p.GetMinerID(), p.getSignKey(bc.MinerID.gid)})
					if locked {
						p.castLock.Unlock()
						locked = false
					}
					if !PROC_TEST_MODE {
						log.Printf("call network service SendVerifiedCast...\n")
						SendVerifiedCast(&send_message)
					} else {
						log.Printf("proc(%v) OMV BEGIN SEND OnMessageVerify 2 ALL PROCS...\n", p.getPrefix())
						for _, v := range p.GroupProcs {
							v.OnMessageVerify(send_message)
						}
					}
				case 1:
					panic("Processer::OMNT failed, check xiaoxiong's src code.")
				case -1:
					log.Printf("OMNT set slot (qn=%v) failed irreversible.\n", v)
					slot.statusChainFailed()
				}
			} else {
				log.Printf("OMNT failed, after ReceTrans slot %v is nil.\n", v)
				panic("OMNT failed, slot is nil.")
			}
		}
	} else {
		log.Printf("OMNT, current proc not in casting, ignore OMNT message.\n")
	}
	if locked {
		p.castLock.Unlock()
		locked = false
	}
	log.Printf("proc(%v) end OMNT.\n", p.getPrefix())
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
					e := core.GroupChainImpl.AddGroup(&group, nil, nil)
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

		log.Printf("begin sleeping 5 seconds, now=%v...\n", time.Now().Format(time.Stamp))
		sleep_d, err := time.ParseDuration("5s")
		if err == nil {
			time.Sleep(sleep_d)
		} else {
			panic("time.ParseDuration 5s failed.")
		}
		log.Printf("end sleeping, now=%v.\n", time.Now().Format(time.Stamp))
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
