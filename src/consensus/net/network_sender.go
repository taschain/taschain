package net

import (
	"consensus/groupsig"
	"github.com/gogo/protobuf/proto"
	"middleware/pb"
	"middleware/types"
	"network"
	"consensus/model"
	"time"
	"middleware/statistics"
	"common"
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

func (ns *NetworkServerImpl) BuildGroupNet(gid groupsig.ID, mems []groupsig.ID) {
	memStrs := id2String(mems)
	ns.net.BuildGroupNet(gid.GetHexString(), memStrs)
}

func (ns *NetworkServerImpl) ReleaseGroupNet(gid groupsig.ID) {
	ns.net.DissolveGroupNet(gid.GetHexString())
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
	ns.send2Self(grm.SI.GetID(), m)

	e = ns.net.Broadcast(m)

	logger.Debugf("SendGroupInitMessage hash:%s,  dummyId %v", m.Hash(), grm.GI.DummyID.GetHexString())
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
		ns.send2Self(spm.SI.GetID(), m)
		return
	}

	begin := time.Now()
	go ns.net.SendWithGroupRelay(spm.Dest.String(), spm.DummyID.GetHexString(), m)
	logger.Debugf("SendKeySharePiece to id:%s,hash:%s, dummyId:%v, cost time:%v", spm.Dest.String(), m.Hash(), spm.DummyID.GetHexString(), time.Since(begin))
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
	go ns.net.SpreadAmongGroup(spkm.DummyID.GetHexString(), m)
	logger.Debugf("SendSignPubKey hash:%s, dummyId:%v, cost time:%v", m.Hash(), spkm.DummyID.GetHexString(), time.Since(begin))
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
	logger.Debugf("Broadcast GROUP_INIT_DONE_MSG, hash:%s, dummyId:%v", m.Hash(), cgm.GI.GIS.DummyID.GetHexString())

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
func (ns *NetworkServerImpl) SendVerifiedCast(cvm *model.ConsensusVerifyMessage) {
	body, e := marshalConsensusVerifyMessage(cvm)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusVerifyMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.VerifiedCastMsg, Body: body}
	var groupId groupsig.ID
	e1 := groupId.Deserialize(cvm.BH.GroupId)
	if e1 != nil {
		logger.Errorf("[peer]Discard send ConsensusCurrentMessage because of Deserialize groupsig id error::%s", e.Error())
		return
	}

	//验证消息需要给自己也发一份，否则自己的分片中将不包含自己的签名，导致分红没有
	ns.send2Self(cvm.SI.GetID(), m)

	timeFromCast := time.Since(cvm.BH.CurTime)
	begin := time.Now()
	go ns.net.SpreadAmongGroup(groupId.GetHexString(), m)
	logger.Debugf("[peer]send VARIFIED_CAST_MSG,%d-%d,invoke Multicast cost time:%v,time from cast:%v,hash:%s", cvm.BH.Height, cvm.BH.ProveValue, time.Since(begin), timeFromCast, m.Hash())
	statistics.AddBlockLog(common.BootId, statistics.SendVerified, cvm.BH.Height, cvm.BH.ProveValue.Uint64(), -1, -1,
		time.Now().UnixNano(), "", "", common.InstanceIndex, cvm.BH.CurTime.UnixNano())
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

//====================================建组前共识=======================

//开始建组
func (ns *NetworkServerImpl) SendCreateGroupRawMessage(msg *model.ConsensusCreateGroupRawMessage) {
	body, e := marshalConsensusCreateGroupRawMessage(msg)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusCreateGroupRawMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CreateGroupaRaw, Body: body}

	var groupId = msg.GI.ParentID
	go ns.net.SpreadAmongGroup(groupId.GetHexString(), m)
}

func (ns *NetworkServerImpl) SendCreateGroupSignMessage(msg *model.ConsensusCreateGroupSignMessage) {
	body, e := marshalConsensusCreateGroupSignMessage(msg)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusCreateGroupSignMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CreateGroupSign, Body: body}

	go ns.net.SendWithGroupRelay(msg.Launcher.String(), msg.GI.ParentID.GetHexString(), m)
}

func (ns *NetworkServerImpl) SendCastRewardSignReq(msg *model.CastRewardTransSignReqMessage) {
	body, e := marshalCastRewardTransSignReqMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send CastRewardTransSignReqMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CastRewardSignReq, Body: body}

	gid := groupsig.DeserializeId(msg.Reward.GroupId)

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

//----------------------------------------------组初始化---------------------------------------------------------------

func marshalConsensusGroupRawMessage(m *model.ConsensusGroupRawMessage) ([]byte, error) {
	gi := consensusGroupInitSummaryToPb(&m.GI)

	sign := signDataToPb(&m.SI)

	ids := make([]*tas_middleware_pb.PubKeyInfo, 0)
	for _, id := range m.MEMS {
		ids = append(ids, pubKeyInfoToPb(&id))
	}

	message := tas_middleware_pb.ConsensusGroupRawMessage{ConsensusGroupInitSummary: gi, Ids: ids, Sign: sign}
	return proto.Marshal(&message)
}

func marshalConsensusSharePieceMessage(m *model.ConsensusSharePieceMessage) ([]byte, error) {
	gisHash := m.GISHash.Bytes()
	dummyId := m.DummyID.Serialize()
	dest := m.Dest.Serialize()
	share := sharePieceToPb(&m.Share)
	sign := signDataToPb(&m.SI)

	message := tas_middleware_pb.ConsensusSharePieceMessage{GISHash: gisHash, DummyID: dummyId, Dest: dest, SharePiece: share, Sign: sign}
	return proto.Marshal(&message)
}

func marshalConsensusSignPubKeyMessage(m *model.ConsensusSignPubKeyMessage) ([]byte, error) {
	hash := m.GISHash.Bytes()
	dummyId := m.DummyID.Serialize()
	signPK := m.SignPK.Serialize()
	signData := signDataToPb(&m.SI)
	sign := m.GISSign.Serialize()

	message := tas_middleware_pb.ConsensusSignPubKeyMessage{GISHash: hash, DummyID: dummyId, SignPK: signPK, SignData: signData, GISSign: sign}
	return proto.Marshal(&message)
}
func marshalConsensusGroupInitedMessage(m *model.ConsensusGroupInitedMessage) ([]byte, error) {
	gi := staticGroupInfoToPb(&m.GI)
	si := signDataToPb(&m.SI)
	message := tas_middleware_pb.ConsensusGroupInitedMessage{StaticGroupSummary: gi, Sign: si}
	return proto.Marshal(&message)
}

//--------------------------------------------组铸币--------------------------------------------------------------------

func consensusBlockMessageBase2Pb(m *model.ConsensusBlockMessageBase) ([]byte, error) {
	bh := types.BlockHeaderToPb(&m.BH)
	//groupId := m.GroupID.Serialize()
	si := signDataToPb(&m.SI)

	hashs := make([][]byte, len(m.ProveHash))
	for i, h := range m.ProveHash {
		hashs[i] = h.Bytes()
	}

	message := tas_middleware_pb.ConsensusBlockMessageBase{Bh: bh, Sign: si, ProveHash: hashs}
	return proto.Marshal(&message)
}

func marshalConsensusCastMessage(m *model.ConsensusCastMessage) ([]byte, error) {
	return consensusBlockMessageBase2Pb(&m.ConsensusBlockMessageBase)
}

func marshalConsensusVerifyMessage(m *model.ConsensusVerifyMessage) ([]byte, error) {
	return consensusBlockMessageBase2Pb(&m.ConsensusBlockMessageBase)
}

func marshalConsensusBlockMessage(m *model.ConsensusBlockMessage) ([]byte, error) {
	block := types.BlockToPb(&m.Block)
	if block == nil {
		logger.Errorf("[peer]Block is nil while marshalConsensusBlockMessage")
	}
	message := tas_middleware_pb.ConsensusBlockMessage{Block: block}
	return proto.Marshal(&message)
}

//----------------------------------------------------------------------------------------------------------------------
func consensusGroupInitSummaryToPb(m *model.ConsensusGroupInitSummary) *tas_middleware_pb.ConsensusGroupInitSummary {
	beginTime, e := m.BeginTime.MarshalBinary()
	if e != nil {
		logger.Errorf("ConsensusGroupInitSummary marshal begin time error:%s", e.Error())
		return nil
	}

	name := make([]byte, 0)
	for _, b := range m.Name {
		name = append(name, b)
	}
	message := tas_middleware_pb.ConsensusGroupInitSummary{
		ParentID:        m.ParentID.Serialize(),
		PrevGroupID:     m.PrevGroupID.Serialize(),
		Authority:       &m.Authority,
		Name:            name,
		DummyID:         m.DummyID.Serialize(),
		BeginTime:       beginTime,
		Members:         &m.Members,
		MemberHash:      m.MemberHash.Bytes(),
		GetReadyHeight:  &m.GetReadyHeight,
		BeginCastHeight: &m.BeginCastHeight,
		DismissHeight:   &m.DismissHeight,
		Signature:       m.Signature.Serialize(),
		TopHeight:       &m.TopHeight,
		Extends:         []byte(m.Extends)}
	return &message
}

func signDataToPb(s *model.SignData) *tas_middleware_pb.SignData {
	sign := tas_middleware_pb.SignData{DataHash: s.DataHash.Bytes(), DataSign: s.DataSign.Serialize(), SignMember: s.SignMember.Serialize()}
	return &sign
}

func sharePieceToPb(s *model.SharePiece) *tas_middleware_pb.SharePiece {
	share := tas_middleware_pb.SharePiece{Seckey: s.Share.Serialize(), Pubkey: s.Pub.Serialize()}
	return &share
}

func staticGroupInfoToPb(s *model.StaticGroupSummary) *tas_middleware_pb.StaticGroupSummary {
	groupId := s.GroupID.Serialize()
	groupPk := s.GroupPK.Serialize()

	gis := consensusGroupInitSummaryToPb(&s.GIS)

	groupInfo := tas_middleware_pb.StaticGroupSummary{GroupID: groupId, GroupPK: groupPk, Gis: gis}
	return &groupInfo
}

func pubKeyInfoToPb(p *model.PubKeyInfo) *tas_middleware_pb.PubKeyInfo {
	id := p.ID.Serialize()
	pk := p.PK.Serialize()

	pkInfo := tas_middleware_pb.PubKeyInfo{ID: id, PublicKey: pk}
	return &pkInfo
}

func marshalConsensusCreateGroupRawMessage(msg *model.ConsensusCreateGroupRawMessage) ([]byte, error) {
	gi := consensusGroupInitSummaryToPb(&msg.GI)

	sign := signDataToPb(&msg.SI)

	ids := make([][]byte, 0)
	for _, id := range msg.IDs {
		ids = append(ids, id.Serialize())
	}

	message := tas_middleware_pb.ConsensusCreateGroupRawMessage{ConsensusGroupInitSummary: gi, Ids: ids, Sign: sign}
	return proto.Marshal(&message)
}

func marshalConsensusCreateGroupSignMessage(msg *model.ConsensusCreateGroupSignMessage) ([]byte, error) {
	gi := consensusGroupInitSummaryToPb(&msg.GI)

	sign := signDataToPb(&msg.SI)

	message := tas_middleware_pb.ConsensusCreateGroupSignMessage{ConsensusGroupInitSummary: gi, Sign: sign}
	return proto.Marshal(&message)
}

func bonusToPB(bonus *types.Bonus) *tas_middleware_pb.Bonus {
	return &tas_middleware_pb.Bonus{
		TxHash:     bonus.TxHash.Bytes(),
		TargetIds:  bonus.TargetIds,
		BlockHash:  bonus.BlockHash.Bytes(),
		GroupId:    bonus.GroupId,
		Sign:       bonus.Sign,
		TotalValue: &bonus.TotalValue,
	}
}

func marshalCastRewardTransSignReqMessage(msg *model.CastRewardTransSignReqMessage) ([]byte, error) {
	b := bonusToPB(&msg.Reward)
	si := signDataToPb(&msg.SI)
	pieces := make([][]byte, 0)
	for _, sp := range msg.SignedPieces {
		pieces = append(pieces, sp.Serialize())
	}
	message := &tas_middleware_pb.CastRewardTransSignReqMessage{
		Sign:         si,
		Reward:       b,
		SignedPieces: pieces,
	}
	return proto.Marshal(message)
}

func marshalCastRewardTransSignMessage(msg *model.CastRewardTransSignMessage) ([]byte, error) {
	si := signDataToPb(&msg.SI)
	message := &tas_middleware_pb.CastRewardTransSignMessage{
		Sign:      si,
		ReqHash:   msg.ReqHash.Bytes(),
		BlockHash: msg.BlockHash.Bytes(),
	}
	return proto.Marshal(message)
}
