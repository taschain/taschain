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
	"middleware/types"
	"time"
	"logservice"
	"gopkg.in/karalabe/cookiejar.v2/exts/mathext"
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

	p.reserveBlock(vctx, slot)

}

func (p *Processor) normalPieceVerify(mtype string, sender string, gid groupsig.ID, vctx *VerifyContext, slot *SlotContext, traceLog *msgTraceLog)  {
	bh := slot.BH
	castor := groupsig.DeserializeId(bh.Castor)
	if slot.StatusTransform(SS_WAITING, SS_SIGNED) && !castor.IsEqual(p.GetMinerID()) {
		vctx.updateSignedMaxQN(bh.TotalQN)
		vctx.increaseSignedNum()
		skey := p.getSignKey(gid)
		var cvm model.ConsensusVerifyMessage
		cvm.BlockHash = bh.Hash
		//cvm.GroupID = gId
		blog := newBizLog("normalPieceVerify")
		if cvm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), skey), &cvm) {
			cvm.GenRandomSign(skey, vctx.prevBH.Random)
			blog.debug("call network service SendVerifiedCast hash=%v, height=%v", bh.Hash.ShortS(), bh.Height)
			traceLog.log("SendVerifiedCast height=%v, castor=%v", bh.Height, slot.castor.ShortS())
			//验证消息需要给自己也发一份，否则自己的分片中将不包含自己的签名，导致分红没有
			p.NetServer.SendVerifiedCast(&cvm, gid)
		} else {
			blog.log("genSign fail, id=%v, sk=%v %v", p.GetMinerID().ShortS(), skey.ShortS(), p.IsMinerGroup(gid))
		}
	}
}

func (p *Processor) doVerify(mtype string, msg *model.ConsensusCastMessage, traceLog *msgTraceLog, blog *bizLog) (cost string, err error) {
	bh := &msg.BH
	si := &msg.SI

	begin := time.Now()
	sender := si.SignMember.ShortS()

	gid := groupsig.DeserializeId(bh.GroupId)
	castor := groupsig.DeserializeId(bh.Castor)

	preBH := p.getBlockHeaderByHash(bh.PreHash)
	if preBH == nil {
		p.addFutureVerifyMsg(msg)
		return "", fmt.Errorf("父块未到达")
	}
	if expireTime, expire := VerifyBHExpire(bh, preBH); expire {
		return "", fmt.Errorf("cast verify expire, gid=%v, preTime %v, expire %v", gid.ShortS(), preBH.CurTime, expireTime)
	} else if bh.Height > 1 {
		//设置为2倍的最大时间，防止由于时间不同步导致的跳块
		beginTime := expireTime.Add(-2*time.Second*time.Duration(model.Param.MaxGroupCastTime))
		if !time.Now().After(beginTime) {
			return "", fmt.Errorf("cast begin time illegal, expectBegin at %v, expire at %v", beginTime, expireTime)
		}

	}
	if !p.IsMinerGroup(gid) {
		return "", fmt.Errorf("%v is not in group %v", p.GetMinerID().ShortS(), gid.ShortS())
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

	vctx := bc.GetOrNewVerifyContext(bh, preBH)
	if vctx == nil {
		err = fmt.Errorf("获取vctx为空，可能preBH已经被删除")
		return
	}
	err = vctx.baseCheck(bh)
	if err != nil {
		return
	}
	cost = fmt.Sprintf("baseCheck:%v", time.Since(begin).String())

	isProposal := castor.IsEqual(si.GetID())

	if isProposal { //提案者
		castorDO := p.minerReader.getProposeMiner(castor)
		if castorDO == nil {
			err = fmt.Errorf("castorDO nil id=%v", castor.ShortS())
			return
		}
		vSignBegin := time.Now()
		b := msg.VerifySign(castorDO.PK)
		cost = fmt.Sprintf("%v,verifyCastorSign:%v", cost, time.Since(vSignBegin).String())

		if !b {
			err = fmt.Errorf("verify sign fail, id %v, pk %v, castorDO %+v", castor.ShortS(), castorDO.PK.GetHexString(), castorDO)
			return
		}

	} else {
		pk, ok := p.GetMemberSignPubKey(model.NewGroupMinerID(gid, si.GetID()))
		if !ok {
			blog.log("GetMemberSignPubKey not ok, ask id %v", si.GetID().ShortS())
			return
		}

		vSignBegin := time.Now()
		b := msg.VerifySign(pk)
		cost = fmt.Sprintf("%v,verifySign:%v", cost, time.Since(vSignBegin).String())

		if !b {
			err = fmt.Errorf("verify sign fail, id %v, pk %v, sig %v hash %v", si.GetID().ShortS(), pk.GetHexString(), si.DataSign.GetHexString(), si.DataHash.Hex())
			return
		}
		vRSignBegin := time.Now()
		b = msg.VerifyRandomSign(pk, preBH.Random)
		cost = fmt.Sprintf("%v,verifyRSign:%v", cost, time.Since(vRSignBegin).String())

		if !b {
			err = fmt.Errorf("random sign verify fail, gid %v, pk %v, sign=%v", gid.ShortS(), pk.ShortS(), groupsig.DeserializeSign(msg.BH.Random).GetHexString())
			return
		}
	}

	legalBegin := time.Now()
	ok, _, err2 := p.isCastLegal(bh, preBH)
	cost = fmt.Sprintf("%v,legalCheck:%v", cost, time.Since(legalBegin).String())
	if !ok {
		err = err2
		return
	}

	sampleCheckBegin := time.Now()
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
	cost = fmt.Sprintf("%v,sampleCheck:%v", cost, time.Since(sampleCheckBegin).String())

	uvBegin := time.Now()
	blog.debug("%v start UserVerified, height=%v, hash=%v", mtype, bh.Height, bh.Hash.ShortS())
	verifyResult := vctx.UserVerified(bh, si)

	cost = fmt.Sprintf("%v,uvCheck:%v", cost, time.Since(uvBegin).String())

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
			thresholdBegin := time.Now()
			p.thresholdPieceVerify(mtype, sender, gid, vctx, slot, traceLog)
			cost = fmt.Sprintf("%v,threVerify:%v", cost, time.Since(thresholdBegin).String())
		}

	case CBMR_PIECE_NORMAL:
		normBegin := time.Now()
		p.normalPieceVerify(mtype, sender, gid, vctx, slot, traceLog)
		cost = fmt.Sprintf("%v,normVerify:%v", cost, time.Since(normBegin).String())

	case CBMR_PIECE_LOSINGTRANS: //交易缺失
	}
	return
}

func (p *Processor) verifyCastMessage(mtype string, msg *model.ConsensusCastMessage) {
	bh := &msg.BH
	si := &msg.SI
	blog := newBizLog(mtype)
	traceLog := newHashTraceLog(mtype, bh.Hash, si.GetID())
	castor := groupsig.DeserializeId(bh.Castor)
	groupId := groupsig.DeserializeId(bh.GroupId)

	traceLog.logStart("height=%v, castor=%v", bh.Height, castor.ShortS())
	blog.debug("proc(%v) begin hash=%v, height=%v, sender=%v, castor=%v, groupId=%v", p.getPrefix(), bh.Hash.ShortS(), bh.Height, si.GetID().ShortS(), castor.ShortS(), groupId.ShortS())

	result := ""
	begin := time.Now()
	costString := ""

	defer func() {
		traceLog.logEnd("height=%v, hash=%v, preHash=%v,groupId=%v, result=%v", bh.Height, bh.Hash.ShortS(), bh.PreHash.ShortS(),groupId.ShortS(), result)
		blog.debug("height=%v, hash=%v, preHash=%v, groupId=%v, result=%v", bh.Height, bh.Hash.ShortS(), bh.PreHash.ShortS(), groupId.ShortS(), result)
		if time.Since(begin).Seconds() > 0.4 {
			slowLogger.Warnf("handle slow:%v, sender=%v, hash=%v, gid=%v, height=%v, preHash=%v, cost %v, detail %v", mtype, si.GetID().ShortS(), bh.Hash.ShortS(), groupId.ShortS(), bh.Height, bh.PreHash.ShortS(), time.Since(begin).String(), costString)
		}
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
	bc := p.GetBlockContext(groupId)
	if bc == nil {
		result = fmt.Sprintf("未获取到blockcontext, gid=" + groupId.ShortS())
		return
	}
	vctx := bc.GetVerifyContextByHeight(bh.Height)
	if vctx != nil {
		err := vctx.baseCheck(bh)
		if err != nil {
			result = err.Error()
			return
		}
	}

	onChainCheckBegin := time.Now()
	if p.blockOnChain(bh.Hash) {
		result = "block onchain already"
	}
	costString = fmt.Sprintf("onchainCheck:%v", time.Since(onChainCheckBegin).String())

	verifyBegin := time.Now()
	cost, err := p.doVerify(mtype, msg, traceLog, blog)
	if err != nil {
		result = err.Error()
	}
	costString = fmt.Sprintf("%v;verify:%v(detail:%v)", costString, time.Since(verifyBegin).String(), cost)
	return
}

func (p *Processor) verifyWithCache(cache *verifyMsgCache, vmsg *model.ConsensusVerifyMessage)  {
	msg := &model.ConsensusCastMessage{
		BH: cache.castMsg.BH,
		ProveHash: cache.castMsg.ProveHash,
		BaseSignedMessage: vmsg.BaseSignedMessage,
	}
	msg.BH.Random = vmsg.RandomSign.Serialize()
	p.verifyCastMessage("OMV", msg)
}

//收到组内成员的出块消息，出块人（KING）用组分片密钥进行了签名
//有可能没有收到OnMessageCurrent就提前接收了该消息（网络时序问题）
func (p *Processor) OnMessageCast(ccm *model.ConsensusCastMessage) {
	//statistics.AddBlockLog(common.BootId, statistics.RcvCast, ccm.BH.Height, ccm.BH.ProveValue.Uint64(), -1, -1,
	//	time.Now().UnixNano(), "", "", common.InstanceIndex, ccm.BH.CurTime.UnixNano())
	bh := &ccm.BH
	le := &logservice.LogEntry{
		LogType: logservice.LogTypeProposal,
		Height: bh.Height,
		Hash: bh.Hash.Hex(),
		PreHash: bh.PreHash.Hex(),
		Proposer: ccm.SI.GetID().GetHexString(),
		Verifier: groupsig.DeserializeId(bh.GroupId).GetHexString(),
		Ext: fmt.Sprintf("external:qn:%v,totalQN:%v", 0, bh.TotalQN),
	}
	group := p.GetGroup(groupsig.DeserializeId(bh.GroupId))

	detalHeight := int(bh.Height - p.MainChain.Height())
	if mathext.AbsInt(detalHeight) < 100 && logservice.Instance.IsFirstNInternalNodesInGroup(group.GetMembers(), 20) {
		logservice.Instance.AddLogIfNotInternalNodes(le)
	}

	p.addCastMsgToCache(ccm)
	cache := p.getVerifyMsgCache(ccm.BH.Hash)

	p.verifyCastMessage("OMC", ccm)


	verifys := cache.getVerifyMsgs()
	if len(verifys) > 0 {
		stdLogger.Infof("OMC:getVerifyMsgs from cache size %v, hash=%v", len(verifys), ccm.BH.Hash.ShortS())
		for _, vmsg := range verifys {
			p.verifyWithCache(cache, vmsg)
		}
		cache.removeVerifyMsgs()
	}

}

//收到组内成员的出块验证通过消息（组内成员消息）
func (p *Processor) OnMessageVerify(cvm *model.ConsensusVerifyMessage) {
	//statistics.AddBlockLog(common.BootId, statistics.RcvVerified, cvm.BH.Height, cvm.BH.ProveValue.Uint64(), -1, -1,
	//	time.Now().UnixNano(), "", "", common.InstanceIndex, cvm.BH.CurTime.UnixNano())
	if p.blockOnChain(cvm.BlockHash) {
		return
	}
	cache := p.getVerifyMsgCache(cvm.BlockHash)
	if cache != nil && cache.castMsg != nil {
		p.verifyWithCache(cache, cvm)
	} else {
		stdLogger.Infof("OMV:no cast msg, cached, hash=%v", cvm.BlockHash.ShortS())
		p.addVerifyMsgToCache(cvm)
	}
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


func (p *Processor) signCastRewardReq(msg *model.CastRewardTransSignReqMessage, bh *types.BlockHeader) (send bool, err error) {
	gid := groupsig.DeserializeId(bh.GroupId)
	group := p.GetGroup(gid)
	reward := &msg.Reward
	if group == nil {
		panic("group is nil")
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
		err = fmt.Errorf("vctx is nil,%v height=%v", vctx == nil, bh.Height)
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

	mpk, ok := p.GetMemberSignPubKey(model.NewGroupMinerID(gid, msg.SI.GetID()))
	if !ok {
		err = fmt.Errorf("GetMemberSignPubKey not ok, ask id %v", gid.ShortS())
		return
	}
	if !msg.VerifySign(mpk) {
		err = fmt.Errorf("verify sign fail, gid=%v, uid=%v", gid.ShortS(), msg.SI.GetID().ShortS())
		return
	}

	gSignGener := model.NewGroupSignGenerator(bc.threshold())

	//witnesses := slot.gSignGenerator.GetWitnesses()
	for idx, idIndex := range msg.Reward.TargetIds {
		id := group.GetMemberID(int(idIndex))
		sign := msg.SignedPieces[idx]
		if sig, ok := slot.gSignGenerator.GetWitness(id); !ok { //本地无该id签名的，需要校验签名
			pk, exist := p.GetMemberSignPubKey(model.NewGroupMinerID(gid, id))
			if !exist {
				continue
			}
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

	begin := time.Now()
	defer func() {
		tlog.logEnd("%v %v", send, err)
		blog.log("blockHash=%v, result=%v %v", reward.BlockHash.ShortS(), send, err)
		if time.Since(begin).Seconds() > 0.5 {
			gid := groupsig.DeserializeId(reward.GroupId)
			slowLogger.Warnf("handle slow:%v, sender=%v, hash=%v, gid=%v, cost %v", mtype, msg.SI.GetID().ShortS(), reward.BlockHash.ShortS(), gid.ShortS(), time.Since(begin).String())
		}
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
	pk, ok := p.GetMemberSignPubKey(model.NewGroupMinerID(gid, msg.SI.GetID()))
	if !ok {
		err = fmt.Errorf("GetMemberSignPubKey not ok, ask id %v", gid.ShortS())
		return
	}
	if !msg.VerifySign(pk) {
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

		////发送日志
		//le := &logservice.LogEntry{
		//	LogType: logservice.LogTypeBonusBroadcast,
		//	Height: bh.Height,
		//	Hash: bh.Hash.Hex(),
		//	PreHash: bh.PreHash.Hex(),
		//	Proposer: slot.castor.GetHexString(),
		//	Verifier: gid.GetHexString(),
		//}
		//logservice.Instance.AddLog(le)

	} else {
		err = fmt.Errorf("accept %v, recover %v, %v", accept, recover, slot.rewardGSignGen.Brief())
	}
}
