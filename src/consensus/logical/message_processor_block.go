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
	"gopkg.in/karalabe/cookiejar.v2/exts/mathext"
	"math"
	"middleware/types"
	"monitor"
	"taslog"
	"time"
)

func (p *Processor) thresholdPieceVerify(vctx *VerifyContext, slot *SlotContext) {
	p.reserveBlock(vctx, slot)
}

func (p *Processor) verifyCastMessage(mtype string, msg *model.ConsensusCastMessage, preBH *types.BlockHeader) (ok bool, err error) {
	bh := &msg.BH
	si := &msg.SI
	castor := groupsig.DeserializeId(bh.Castor)
	groupId := groupsig.DeserializeId(bh.GroupId)

	defer func() {
		if ok {
			go func() {
				verifys := p.blockContexts.getVerifyMsgCache(bh.Hash)
				if verifys != nil {
					for _, vmsg := range verifys.verifyMsgs {
						p.OnMessageVerify(vmsg)
					}
				}
				p.blockContexts.removeVerifyMsgCache(bh.Hash)
			}()
		}
	}()

	if p.blockOnChain(bh.Hash) {
		err = fmt.Errorf("block onchain already")
		return
	}

	expireTime := ExpireTime(bh, preBH)
	if p.ts.NowAfter(expireTime) {
		err = fmt.Errorf("cast verify expire, gid=%v, preTime %v, expire %v", groupId.ShortS(), preBH.CurTime, expireTime)
		return
	} else if bh.Height > 1 {
		beginTime := expireTime.Add(-int64(model.Param.MaxGroupCastTime + 1))
		if !p.ts.NowAfter(beginTime) {
			err = fmt.Errorf("cast begin time illegal, expectBegin at %v, expire at %v", beginTime, expireTime)
			return
		}

	}
	if _, same := p.blockContexts.isHeightCasted(bh.Height, bh.PreHash); same {
		err = fmt.Errorf("该高度已铸过 %v", bh.Height)
		return
	}

	group := p.GetGroup(groupId)
	if group == nil {
		err = fmt.Errorf("group is nil:groupId=%v", groupId.GetHexString())
		return
	}

	vctx := p.blockContexts.getVctxByHeight(bh.Height)
	if vctx != nil {
		if vctx.blockSigned(bh.Hash) {
			err = fmt.Errorf("block signed")
			return
		}
		if vctx.prevBH.Hash == bh.PreHash {
			err = vctx.baseCheck(bh, si.GetID())
			if err != nil {
				return
			}
		}
	}
	castorDO := p.minerReader.getProposeMiner(castor)
	if castorDO == nil {
		err = fmt.Errorf("castorDO nil id=%v", castor.ShortS())
		return
	}
	pk := castorDO.PK

	if !msg.VerifySign(pk) {
		err = fmt.Errorf("verify sign fail")
		return
	}

	//提案是否合法
	ok, _, err = p.isCastLegal(bh, preBH)
	if !ok {
		return
	}

	//校验提案者是否有全量账本
	existHash := p.proveChecker.genProveHash(bh.Height, preBH.Random, p.GetMinerID())
	if msg.ProveHash != existHash {
		err = fmt.Errorf("check p rove hash fail, receive hash=%v, exist hash=%v", msg.ProveHash.ShortS(), existHash.ShortS())
		return
	}

	vctx = p.blockContexts.getOrNewVerifyContext(group, bh, preBH)
	if vctx == nil {
		err = fmt.Errorf("获取vctx为空，可能preBH已经被删除")
		return
	}

	slot, err := vctx.PrepareSlot(bh)
	if err != nil {
		return
	}
	if !slot.IsWaiting() {
		err = fmt.Errorf("slot status %v, not waiting", slot.GetSlotStatus())
		return
	}

	skey := p.getSignKey(groupId)
	var cvm model.ConsensusVerifyMessage
	cvm.BlockHash = bh.Hash
	if cvm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), skey), &cvm) {
		cvm.GenRandomSign(skey, vctx.prevBH.Random)
		p.NetServer.SendVerifiedCast(&cvm, groupId)
		slot.setSlotStatus(slSigned)
		p.blockContexts.attachVctx(bh, vctx)
		vctx.markSignedBlock(bh)
		ok = true
	} else {
		err = fmt.Errorf("gen sign fail")
	}
	return
}

//收到组内成员的出块消息，出块人（KING）用组分片密钥进行了签名
//有可能没有收到OnMessageCurrent就提前接收了该消息（网络时序问题）
func (p *Processor) OnMessageCast(ccm *model.ConsensusCastMessage) {
	slog := taslog.NewSlowLog("OnMessageCast", 0.5)
	bh := &ccm.BH
	traceLog := monitor.NewPerformTraceLogger("OnMessageCast", bh.Hash, bh.Height)

	defer func() {
		slog.Log("hash=%v, sender=%v, height=%v, preHash=%v", bh.Hash.ShortS(), ccm.SI.GetID().ShortS(), bh.Height, bh.PreHash.ShortS())
	}()

	le := &monitor.LogEntry{
		LogType:  monitor.LogTypeProposal,
		Height:   bh.Height,
		Hash:     bh.Hash.Hex(),
		PreHash:  bh.PreHash.Hex(),
		Proposer: ccm.SI.GetID().GetHexString(),
		Verifier: groupsig.DeserializeId(bh.GroupId).GetHexString(),
		Ext:      fmt.Sprintf("external:qn:%v,totalQN:%v", 0, bh.TotalQN),
	}
	slog.AddStage("getGroup")
	group := p.GetGroup(groupsig.DeserializeId(bh.GroupId))
	slog.EndStage()

	slog.AddStage("addLog-height")
	detalHeight := int(bh.Height - p.MainChain.Height())
	slog.EndStage()
	slog.AddStage("checkAddLog")
	if mathext.AbsInt(detalHeight) < 100 && monitor.Instance.IsFirstNInternalNodesInGroup(group.GetMembers(), 3) {
		monitor.Instance.AddLogIfNotInternalNodes(le)
	}
	slog.EndStage()
	mtype := "OMC"

	si := &ccm.SI
	tlog := newHashTraceLog(mtype, bh.Hash, si.GetID())
	castor := groupsig.DeserializeId(bh.Castor)
	groupId := groupsig.DeserializeId(bh.GroupId)

	tlog.logStart("%v:height=%v, castor=%v", mtype, bh.Height, castor.ShortS())

	var err error

	defer func() {
		result := "signed"
		if err != nil {
			result = err.Error()
		}
		tlog.logEnd("%v:height=%v, hash=%v, preHash=%v,groupId=%v, result=%v", mtype, bh.Height, bh.Hash.ShortS(), bh.PreHash.ShortS(), groupId.ShortS(), result)
		slog.Log("senderShort=%v, hash=%v, gid=%v, height=%v", si.GetID().ShortS(), bh.Hash.ShortS(), groupId.ShortS(), bh.Height)
		traceLog.Log("PreHash=%v,castor=%v,result=%v", bh.PreHash.ShortS(), ccm.SI.GetID().ShortS(), result)

	}()
	if ccm.GenHash() != ccm.SI.DataHash {
		err = fmt.Errorf("msg genHash %v diff from si.DataHash %v", ccm.GenHash().ShortS(), ccm.SI.DataHash.ShortS())
		return
	}
	//castor要忽略自己的消息
	if castor.IsEqual(p.GetMinerID()) && si.GetID().IsEqual(p.GetMinerID()) {
		err = fmt.Errorf("ignore self message")
		return
	}
	if !p.IsMinerGroup(groupId) { //检测当前节点是否在该铸块组
		err = fmt.Errorf("don't belong to group, gid=%v, hash=%v, id=%v", groupId.ShortS(), bh.Hash.ShortS(), p.GetMinerID().ShortS())
		return
	}

	if bh.Elapsed <= 0 {
		err = fmt.Errorf("elapsed error %v", bh.Elapsed)
		return
	}

	if p.ts.Since(bh.CurTime) < -1 {
		err = fmt.Errorf("block too early: now %v, curtime %v", p.ts.Now(), bh.CurTime)
		return
	}

	slog.AddStage("checkOnChain")
	if p.blockOnChain(bh.Hash) {
		slog.EndStage()
		err = fmt.Errorf("block onchain already")
		return
	}
	slog.EndStage()

	slog.AddStage("checkPreBlock")
	preBH := p.getBlockHeaderByHash(bh.PreHash)
	slog.EndStage()

	slog.AddStage("baseCheck")
	if preBH == nil {
		p.addFutureVerifyMsg(ccm)
		err = fmt.Errorf("父块未到达")
		return
	}

	verifyTraceLog := monitor.NewPerformTraceLogger("verifyCastMessage", bh.Hash, bh.Height)
	verifyTraceLog.SetParent("OnMessageCast")
	defer verifyTraceLog.Log("")
	slog.AddStage("OMC")
	_, err = p.verifyCastMessage("OMC", ccm, preBH)
	slog.EndStage()

}

func (p *Processor) doVerify(cvm *model.ConsensusVerifyMessage, vctx *VerifyContext) (ret int8, err error) {
	blockHash := cvm.BlockHash
	if p.blockOnChain(blockHash) {
		return
	}
	slog := taslog.NewSlowLog("OMV", 0.5)

	slot := vctx.GetSlotByHash(blockHash)
	if slot == nil {
		err = fmt.Errorf("slot is nil")
		return
	}
	//castor要忽略自己的消息
	if slot.castor.IsEqual(p.GetMinerID()) && cvm.SI.GetID().IsEqual(p.GetMinerID()) {
		err = fmt.Errorf("ignore self message")
		return
	}
	bh := slot.BH
	groupId := vctx.group.GroupID

	if err = vctx.baseCheck(bh, cvm.SI.GetID()); err != nil {
		return
	}

	if !p.IsMinerGroup(groupId) { //检测当前节点是否在该铸块组
		err = fmt.Errorf("don't belong to group, gid=%v, hash=%v, id=%v", groupId.ShortS(), bh.Hash.ShortS(), p.GetMinerID().ShortS())
		return
	}
	if !p.blockOnChain(vctx.prevBH.Hash) {
		err = fmt.Errorf("pre not on chain:hash=%v", vctx.prevBH.Hash.ShortS())
		return
	}

	if cvm.GenHash() != cvm.SI.DataHash {
		err = fmt.Errorf("msg genHash %v diff from si.DataHash %v", cvm.GenHash().ShortS(), cvm.SI.DataHash.ShortS())
		return
	}

	if _, same := p.blockContexts.isHeightCasted(bh.Height, bh.PreHash); same {
		err = fmt.Errorf("该高度已铸过 %v", bh.Height)
		return
	}

	slog.AddStage("getPK")
	pk, ok := p.GetMemberSignPubKey(model.NewGroupMinerID(groupId, cvm.SI.GetID()))
	if !ok {
		err = fmt.Errorf("get member sign pubkey fail: gid=%v, uid=%v", groupId.ShortS(), cvm.SI.GetID().ShortS())
		return
	}
	slog.EndStage()

	slog.AddStage("vMemSign")
	if !cvm.VerifySign(pk) {
		err = fmt.Errorf("verify sign fail")
		return
	}
	slog.EndStage()
	slog.AddStage("vRandSign")
	if !groupsig.VerifySig(pk, vctx.prevBH.Random, cvm.RandomSign) {
		err = fmt.Errorf("verify random sign fail")
		return
	}
	slog.EndStage()

	slog.AddStage("acceptPiece")
	ret, err = slot.AcceptVerifyPiece(cvm.SI.GetID(), cvm.SI.DataSign, cvm.RandomSign)
	vctx.increaseVerifyNum()
	if err != nil {
		return
	}
	slog.EndStage()
	if ret == pieceThreshold {
		slog.AddStage("reserveBlock")
		p.reserveBlock(vctx, slot)
		vctx.increaseAggrNum()
		slog.EndStage()
	}
	return
}

//收到组内成员的出块验证通过消息（组内成员消息）
func (p *Processor) OnMessageVerify(cvm *model.ConsensusVerifyMessage) {
	blockHash := cvm.BlockHash
	tlog := newHashTraceLog("OMV", blockHash, cvm.SI.GetID())
	traceLog := monitor.NewPerformTraceLogger("OnMessageVerify", blockHash, 0)


	var (
		err error
		ret int8
	)
	defer func() {
		tlog.logEnd("sender=%v, ret=%v %v", cvm.SI.GetID().ShortS(), ret, err)
		traceLog.Log("result=%v, %v", ret, err)
	}()

	vctx := p.blockContexts.getVctxByHash(blockHash)
	if vctx == nil {
		err = fmt.Errorf("verify context is nil, cache msg")
		p.blockContexts.addVerifyMsg(cvm)
		return
	}
	traceLog.SetHeight(vctx.castHeight)
	ret, err = p.doVerify(cvm, vctx)

	return
}

//收到铸块上链消息(组外矿工节点处理)
func (p *Processor) OnMessageBlock(cbm *model.ConsensusBlockMessage) {
	return
}

//新的交易到达通知（用于处理大臣验证消息时缺失的交易）
func (p *Processor) OnMessageNewTransactions(ths []common.Hash) {
	return
}

func (p *Processor) signCastRewardReq(msg *model.CastRewardTransSignReqMessage, bh *types.BlockHeader, slog *taslog.SlowLog) (send bool, err error) {
	gid := groupsig.DeserializeId(bh.GroupId)
	group := p.GetGroup(gid)
	reward := &msg.Reward
	if group == nil {
		panic("group is nil")
	}
	slog.AddStage("baseCheck")

	vctx := p.blockContexts.getVctxByHeight(bh.Height)
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

	if !bytes.Equal(bh.GroupId, reward.GroupId) {
		err = fmt.Errorf("groupID error %v %v", bh.GroupId, reward.GroupId)
		return
	}
	slog.EndStage()
	if !slot.hasSignedTxHash(reward.TxHash) {

		slog.AddStage("GenerateBonus")
		genBonus, _ := p.MainChain.GetBonusManager().GenerateBonus(reward.TargetIds, bh.Hash, bh.GroupId, model.Param.VerifyBonus)
		if genBonus.TxHash != reward.TxHash {
			err = fmt.Errorf("bonus txHash diff %v %v", genBonus.TxHash.ShortS(), reward.TxHash.ShortS())
			return
		}
		slog.EndStage()

		if len(msg.Reward.TargetIds) != len(msg.SignedPieces) {
			err = fmt.Errorf("targetId len differ from signedpiece len %v %v", len(msg.Reward.TargetIds), len(msg.SignedPieces))
			return
		}

		mpk, ok := p.GetMemberSignPubKey(model.NewGroupMinerID(gid, msg.SI.GetID()))
		if !ok {
			err = fmt.Errorf("GetMemberSignPubKey not ok, ask id %v", gid.ShortS())
			return
		}
		slog.AddStage("vMemSign")
		if !msg.VerifySign(mpk) {
			slog.EndStage()
			err = fmt.Errorf("verify sign fail, gid=%v, uid=%v", gid.ShortS(), msg.SI.GetID().ShortS())
			return
		}
		slog.EndStage()

		//复用原来的generator，避免重复签名验证
		gSignGener := slot.gSignGenerator

		slog.AddStage("checkTargetSign")
		//witnesses := slot.gSignGenerator.GetWitnesses()
		for idx, idIndex := range msg.Reward.TargetIds {
			id := group.GetMemberID(int(idIndex))
			sign := msg.SignedPieces[idx]
			if sig, ok := gSignGener.GetWitness(id); !ok { //本地无该id签名的，需要校验签名
				pk, exist := p.GetMemberSignPubKey(model.NewGroupMinerID(gid, id))
				if !exist {
					continue
				}
				slog.AddStage(fmt.Sprintf("checkSignMem%v", idx))
				if !groupsig.VerifySig(pk, bh.Hash.Bytes(), sign) {
					err = fmt.Errorf("verify member sign fail, id=%v", id.ShortS())
					return
				}
				slog.EndStage()
				//加入generator中
				gSignGener.AddWitnessForce(id, sign)
			} else { //本地已有该id的签名的，只要判断是否跟本地签名一样即可
				if !sign.IsEqual(sig) {
					err = fmt.Errorf("member sign different id=%v", id.ShortS())
					return
				}
			}
		}
		slog.EndStage()

		if !gSignGener.Recovered() {
			err = fmt.Errorf("recover group sign fail")
			return
		}

		bhSign := groupsig.DeserializeSign(bh.Signature)
		if !gSignGener.GetGroupSign().IsEqual(*bhSign) {
			err = fmt.Errorf("recovered sign differ from bh sign, recover %v, bh %v", gSignGener.GetGroupSign().ShortS(), bhSign.ShortS())
			return
		}

		slot.addSignedTxHash(reward.TxHash)
	}

	slog.AddStage("EndSend")
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
	slog.EndStage()
	return
}

func (p *Processor) OnMessageCastRewardSignReq(msg *model.CastRewardTransSignReqMessage) {
	mtype := "OMCRSR"
	blog := newBizLog(mtype)
	reward := &msg.Reward
	tlog := newHashTraceLog("OMCRSR", reward.BlockHash, msg.SI.GetID())
	blog.log("begin, sender=%v, blockHash=%v, txHash=%v", msg.SI.GetID().ShortS(), reward.BlockHash.ShortS(), reward.TxHash.ShortS())
	tlog.logStart("txHash=%v", reward.TxHash.ShortS())
	slog := taslog.NewSlowLog(mtype, 0.5)

	var (
		send bool
		err  error
	)

	defer func() {
		tlog.logEnd("txHash=%v, %v %v", reward.TxHash.ShortS(), send, err)
		blog.log("blockHash=%v, txHash=%v, result=%v %v", reward.BlockHash.ShortS(), reward.TxHash.ShortS(), send, err)
		slog.Log("sender=%v, hash=%v, txHash=%v", msg.SI.GetID().ShortS(), reward.BlockHash.ShortS(), reward.TxHash.ShortS())
	}()

	//此时块不一定在链上
	slog.AddStage("ChecBlock")
	bh := p.getBlockHeaderByHash(reward.BlockHash)
	if bh == nil {
		slog.EndStage()
		err = fmt.Errorf("future reward request receive and cached, hash=%v", reward.BlockHash.ShortS())
		msg.ReceiveTime = time.Now()
		p.futureRewardReqs.addMessage(reward.BlockHash, msg)
		return
	}
	slog.EndStage()

	send, err = p.signCastRewardReq(msg, bh, slog)
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

	vctx := p.blockContexts.getVctxByHeight(bh.Height)
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
	if accept && recover && slot.StatusTransform(slRewardSignReq, slRewardSent) {
		_, err2 := p.MainChain.GetTransactionPool().AddTransaction(slot.rewardTrans)
		send = true
		err = fmt.Errorf("add rewardTrans to txPool, txHash=%v, ret=%v", slot.rewardTrans.Hash.ShortS(), err2)

	} else {
		err = fmt.Errorf("accept %v, recover %v, %v", accept, recover, slot.rewardGSignGen.Brief())
	}
}

func (p *Processor) OnMessageReqProposalBlock(msg *model.ReqProposalBlock, sourceId string) {
	blog := newBizLog("OMRPB")
	blog.log("hash %v", msg.Hash.ShortS())

	from :=  groupsig.ID{}
	from.SetHexString(sourceId)
	tlog := newHashTraceLog("OMRPB", msg.Hash, from)

	var s string
	defer func() {
		tlog.log("result:%v", s)
	}()

	pb := p.blockContexts.getProposed(msg.Hash)
	if pb == nil || pb.block == nil {
		s = fmt.Sprintf("block is nil")
		blog.log("block is nil hash=%v", msg.Hash.ShortS())
		return
	}

	if pb.maxResponseCount == 0 {
		gid := groupsig.DeserializeId(pb.block.Header.GroupId)
		group ,err:= p.globalGroups.GetGroupByID(gid)
		if err != nil {
			s = fmt.Sprintf("get group error")
			blog.log("block proposal response, GetGroupByID err= %v,  hash=%v", err , msg.Hash.ShortS())
			return
		}

		pb.maxResponseCount =uint(math.Ceil(float64(group.GetMemberCount())/3))
	}

	if pb.responseCount >= pb.maxResponseCount {
		s = fmt.Sprintf("response count exceed")
		blog.log("block proposal response count >= maxResponseCount(%v), not response, hash=%v", pb.maxResponseCount, msg.Hash.ShortS())
		return
	}

	pb.responseCount += 1

	s = fmt.Sprintf("response txs size %v", len(pb.block.Transactions))
	blog.log("block proposal response, count=%v, max count=%v, hash=%v", pb.responseCount,pb.maxResponseCount,msg.Hash.ShortS())

	m := &model.ResponseProposalBlock{
		Hash:         pb.block.Header.Hash,
		Transactions: pb.block.Transactions,
	}

	p.NetServer.ResponseProposalBlock(m, sourceId)
}

func (p *Processor) OnMessageResponseProposalBlock(msg *model.ResponseProposalBlock) {
	blog := newBizLog("OMRSPB")
	blog.log("hash %v", msg.Hash.ShortS())

	tlog := newHashTraceLog("OMRSPB", msg.Hash, groupsig.ID{})

	var s string
	defer func() {
		tlog.log("result:%v", s)
	}()

	if p.blockOnChain(msg.Hash) {
		s = "block onchain"
		return
	}
	vctx := p.blockContexts.getVctxByHash(msg.Hash)
	if vctx == nil {
		s = "vctx is nil"
		blog.log("verify context is nil, cache msg")
		return
	}
	slot := vctx.GetSlotByHash(msg.Hash)
	if slot == nil {
		s = "slot is nil"
		blog.log("slot is nil")
		return
	}
	block := types.Block{Header: slot.BH, Transactions: msg.Transactions}
	err := p.onBlockSignAggregation(&block, slot.gSignGenerator.GetGroupSign(), slot.rSignGenerator.GetGroupSign())
	if err != nil {
		blog.log("onBlockSignAggregation fail: %v", err)
		slot.setSlotStatus(slFailed)
		s = fmt.Sprintf("on block fail err=%v", err)
		return
	}
	s = "success"
}
