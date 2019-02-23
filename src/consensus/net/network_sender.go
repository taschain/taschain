package net

import (
	"consensus/groupsig"
	"middleware/types"
	"network"
	"consensus/model"
	"time"
	"core"
)

type NetworkServerImpl struct {
	net network.Network
}



func NewNetworkServer() NetworkServer {
	return &NetworkServerImpl{
		net: network.GetNetInstance(),
	}
}

func id2String(ids []groupsig.ID) []string {
	idStrs := make([]string, len(ids))
	for idx, id := range ids {
		idStrs[idx] = id.String()
	}
	return idStrs
}

//------------------------------------组网络管理-----------------------

func (ns *NetworkServerImpl) BuildGroupNet(gid string, mems []groupsig.ID) {
	memStrs := id2String(mems)
	ns.net.BuildGroupNet(gid, memStrs)
}

func (ns *NetworkServerImpl) ReleaseGroupNet(gid string) {
	ns.net.DissolveGroupNet(gid)
}

func (ns *NetworkServerImpl) send2Self(self groupsig.ID, m network.Message) {
	go MessageHandler.Handle(self.String(), m)
}

//----------------------------------------------------组初始化-----------------------------------------------------------

//广播 组初始化消息  全网广播
func (ns *NetworkServerImpl) SendGroupInitMessage(grm *model.ConsensusGroupRawMessage) {
	body, e := marshalConsensusGroupRawMessage(grm)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusGroupRawMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.GroupInitMsg, Body: body}
	//给自己发
	//ns.send2Self(grm.SI.GetID(), m)
	//memIds := id2String(grm.GInfo.Mems)
	//e = ns.net.Broadcast(m)
	//e = ns.net.SpreadToGroup(grm.GInfo.GroupHash().Hex(), memIds, m, grm.GInfo.GroupHash().Bytes())
	//目标组还未建成，需要点对点发送
	for _, mem := range grm.GInfo.Mems {
		logger.Debugf("%v SendGroupInitMessage gHash %v to %v", grm.SI.GetID().String(), grm.GInfo.GroupHash().Hex(), mem.String())
		ns.net.Send(mem.String(), m)
	}
	//logger.Debugf("SendGroupInitMessage hash:%s,  gHash %v", m.Hash(), grm.GInfo.GroupHash().Hex())
}

//组内广播密钥   for each定向发送 组内广播
func (ns *NetworkServerImpl) SendKeySharePiece(spm *model.ConsensusSharePieceMessage) {

	body, e := marshalConsensusSharePieceMessage(spm)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusSharePieceMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.KeyPieceMsg, Body: body}
	if spm.SI.SignMember.IsEqual(spm.Dest) {
		go ns.send2Self(spm.SI.GetID(), m)
		return
	}

	begin := time.Now()
	go ns.net.SendWithGroupRelay(spm.Dest.String(), spm.GHash.Hex(), m)
	logger.Debugf("SendKeySharePiece to id:%s,hash:%s, gHash:%v, cost time:%v", spm.Dest.String(), m.Hash(), spm.GHash.Hex(), time.Since(begin))
}

//组内广播签名公钥
func (ns *NetworkServerImpl) SendSignPubKey(spkm *model.ConsensusSignPubKeyMessage) {
	body, e := marshalConsensusSignPubKeyMessage(spkm)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusSignPubKeyMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.SignPubkeyMsg, Body: body}
	//给自己发
	ns.send2Self(spkm.SI.GetID(), m)

	begin := time.Now()
	go ns.net.SpreadAmongGroup(spkm.GHash.Hex(), m)
	logger.Debugf("SendSignPubKey hash:%s, dummyId:%v, cost time:%v", m.Hash(), spkm.GHash.Hex(), time.Since(begin))
}

//组初始化完成 广播组信息 全网广播
func (ns *NetworkServerImpl) BroadcastGroupInfo(cgm *model.ConsensusGroupInitedMessage) {
	body, e := marshalConsensusGroupInitedMessage(cgm)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusGroupInitedMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.GroupInitDoneMsg, Body: body}
	//给自己发
	ns.send2Self(cgm.SI.GetID(), m)

	go ns.net.Broadcast(m)
	logger.Debugf("Broadcast GROUP_INIT_DONE_MSG, hash:%s, gHash:%v", m.Hash(), cgm.GHash.Hex())

}

//-----------------------------------------------------------------组铸币----------------------------------------------

//铸币节点完成铸币，将blockheader  签名后发送至组内其他节点进行验证。组内广播
func (ns *NetworkServerImpl) SendCastVerify(ccm *model.ConsensusCastMessage, group *GroupBrief, body []*types.Transaction) {

	//txs, e := types.MarshalTransactions(body)
	//if e != nil {
	//	logger.Errorf("[peer]Discard send cast verify because of MarshalTransactions error:%s", e.Error())
	//	return
	//}

	var groupId groupsig.ID
	e1 := groupId.Deserialize(ccm.BH.GroupId)
	if e1 != nil {
		logger.Errorf("[peer]Discard send ConsensusCurrentMessage because of Deserialize groupsig id error::%s", e1.Error())
		return
	}
	timeFromCast := time.Since(ccm.BH.CurTime)
	begin := time.Now()

	mems := id2String(group.MemIds)

	//txMsg := network.Message{Code: network.TransactionMsg, Body: txs}
	//go ns.net.SpreadToGroup(groupId.GetHexString(), mems, txMsg, ccm.BH.TxTree.Bytes())

	ccMsg, e := marshalConsensusCastMessage(ccm)
	if e != nil {
		logger.Errorf("[peer]Discard send cast verify because of marshalConsensusCastMessage error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CastVerifyMsg, Body: ccMsg}
	go ns.net.SpreadToGroup(groupId.GetHexString(), mems, m, ccm.BH.Hash.Bytes())
	logger.Debugf("send CAST_VERIFY_MSG,%d-%d to group:%s,invoke SpreadToGroup cost time:%v,time from cast:%v,hash:%s", ccm.BH.Height, ccm.BH.TotalQN, groupId.GetHexString(), time.Since(begin), timeFromCast,  ccm.BH.Hash.String())
}

//组内节点  验证通过后 自身签名 广播验证块 组内广播  验证不通过 保持静默
func (ns *NetworkServerImpl) SendVerifiedCast(cvm *model.ConsensusVerifyMessage, receiver groupsig.ID) {
	body, e := marshalConsensusVerifyMessage(cvm)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusVerifyMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.VerifiedCastMsg2, Body: body}

	//验证消息需要给自己也发一份，否则自己的分片中将不包含自己的签名，导致分红没有
	go ns.send2Self(cvm.SI.GetID(), m)

	go ns.net.SpreadAmongGroup(receiver.GetHexString(), m)
	logger.Debugf("[peer]send VARIFIED_CAST_MSG,hash:%s", cvm.BlockHash.String())
	//statistics.AddBlockLog(common.BootId, statistics.SendVerified, cvm.BH.Height, cvm.BH.ProveValue.Uint64(), -1, -1,
	//	time.Now().UnixNano(), "", "", common.InstanceIndex, cvm.BH.CurTime.UnixNano())
}

//对外广播经过组签名的block 全网广播
func (ns *NetworkServerImpl) BroadcastNewBlock(cbm *model.ConsensusBlockMessage, group *GroupBrief) {
	timeFromCast := time.Since(cbm.Block.Header.CurTime)
	body, e := types.MarshalBlock(&cbm.Block)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusBlockMessage because of marshal error:%s", e.Error())
		return
	}
	blockMsg := network.Message{Code: network.NewBlockMsg, Body: body}
	//blockHash := cbm.Block.Header.Hash

	nextVerifyGroupId := group.Gid.GetHexString()
	groupMembers := id2String(group.MemIds)

	//广播给重节点的虚拟组
	heavyMinerMembers := core.MinerManagerImpl.GetHeavyMiners()
	go ns.net.SpreadToGroup(network.FULL_NODE_VIRTUAL_GROUP_ID, heavyMinerMembers, blockMsg, []byte(blockMsg.Hash()))
	//广播给轻节点的下一个组
	go ns.net.SpreadToGroup(nextVerifyGroupId, groupMembers, blockMsg, []byte(blockMsg.Hash()))

	core.Logger.Debugf("Broad new block %d-%d,hash:%v,tx count:%d,msg size:%d, time from cast:%v,spread over group:%s", cbm.Block.Header.Height, cbm.Block.Header.TotalQN, cbm.Block.Header.Hash.Hex(), len(cbm.Block.Header.Transactions), len(blockMsg.Body), timeFromCast, nextVerifyGroupId)
	//statistics.AddBlockLog(common.BootId, statistics.BroadBlock, cbm.Block.Header.Height, cbm.Block.Header.ProveValue.Uint64(), len(cbm.Block.Transactions), len(body),
	//	time.Now().UnixNano(), "", "", common.InstanceIndex, cbm.Block.Header.CurTime.UnixNano())
}


func (ns *NetworkServerImpl) AnswerSignPkMessage(msg *model.ConsensusSignPubKeyMessage, receiver groupsig.ID) {
	body, e := marshalConsensusSignPubKeyMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusSignPubKeyMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.AnswerSignPkMsg, Body: body}

	begin := time.Now()
	go ns.net.Send(receiver.GetHexString(), m)
	logger.Debugf("AnswerSignPkMessage %v, hash:%s, dummyId:%v, cost time:%v", receiver.GetHexString(), m.Hash(), msg.GHash.Hex(), time.Since(begin))
}

func (ns *NetworkServerImpl) AskSignPkMessage(msg *model.ConsensusSignPubkeyReqMessage, receiver groupsig.ID) {
	body, e := marshalConsensusSignPubKeyReqMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusSignPubkeyReqMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.AskSignPkMsg, Body: body}

	begin := time.Now()
	go ns.net.Send(receiver.GetHexString(), m)
	logger.Debugf("AskSignPkMessage %v, hash:%s, cost time:%v", receiver.GetHexString(), m.Hash(), time.Since(begin))
}

//====================================建组前共识=======================

//开始建组
func (ns *NetworkServerImpl) SendCreateGroupRawMessage(msg *model.ConsensusCreateGroupRawMessage) {
	body, e := marshalConsensusCreateGroupRawMessage(msg)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusCreateGroupRawMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CreateGroupaRaw, Body: body}

	var groupId = msg.GInfo.GI.ParentID()
	go ns.net.SpreadAmongGroup(groupId.GetHexString(), m)
}

func (ns *NetworkServerImpl) SendCreateGroupSignMessage(msg *model.ConsensusCreateGroupSignMessage, parentGid groupsig.ID) {
	body, e := marshalConsensusCreateGroupSignMessage(msg)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusCreateGroupSignMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CreateGroupSign, Body: body}

	go ns.net.SendWithGroupRelay(msg.Launcher.String(), parentGid.GetHexString(), m)
}

func (ns *NetworkServerImpl) SendCastRewardSignReq(msg *model.CastRewardTransSignReqMessage) {
	body, e := marshalCastRewardTransSignReqMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send CastRewardTransSignReqMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CastRewardSignReq, Body: body}

	gid := groupsig.DeserializeId(msg.Reward.GroupId)

	network.Logger.Debugf("send SendCastRewardSignReq to %v", gid.GetHexString())

	ns.net.SpreadAmongGroup(gid.GetHexString(), m)
}

func (ns *NetworkServerImpl) SendCastRewardSign(msg *model.CastRewardTransSignMessage) {
	body, e := marshalCastRewardTransSignMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send CastRewardTransSignMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CastRewardSignGot, Body: body}

	ns.net.SendWithGroupRelay(msg.Launcher.String(), msg.GroupID.GetHexString(), m)
}

func (ns *NetworkServerImpl) SendGroupPingMessage(msg *model.CreateGroupPingMessage, receiver groupsig.ID) {
	body, e := marshalCreateGroupPingMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send SendGroupPingMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.GroupPing, Body: body}

	ns.net.Send(receiver.String(), m)
}

func (ns *NetworkServerImpl) SendGroupPongMessage(msg *model.CreateGroupPongMessage, group *GroupBrief) {
	body, e := marshalCreateGroupPongMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send SendGroupPongMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.GroupPong, Body: body}

	mems := id2String(group.MemIds)

	ns.net.SpreadToGroup(group.Gid.GetHexString(), mems, m, msg.SI.DataHash.Bytes())
}

func (ns *NetworkServerImpl) ReqSharePiece(msg *model.ReqSharePieceMessage, receiver groupsig.ID) {
	body, e := marshalSharePieceReqMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send marshalSharePieceReqMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.ReqSharePiece, Body: body}

	ns.net.Send(receiver.String(), m)
}

func (ns *NetworkServerImpl) ResponseSharePiece(msg *model.ResponseSharePieceMessage, receiver groupsig.ID) {
	body, e := marshalSharePieceResponseMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send marshalSharePieceResponseMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.ResponseSharePiece, Body: body}

	ns.net.Send(receiver.String(), m)
}