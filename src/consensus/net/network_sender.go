package net

import (
	"consensus/groupsig"
	"github.com/gogo/protobuf/proto"
	"middleware/pb"
	"middleware/types"
	"network"
	"consensus/model"
	"time"
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

//----------------------------------------------------组初始化-----------------------------------------------------------

//广播 组初始化消息  全网广播
func (ns *NetworkServerImpl) SendGroupInitMessage(grm *model.ConsensusGroupRawMessage) {
	body, e := marshalConsensusGroupRawMessage(grm)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusGroupRawMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.GroupInitMsg, Body: body}
	//给自己发
	go MessageHandler.Handle(grm.SI.SignMember.String(), m)

	e = ns.net.Broadcast(m, -1)

	logger.Debugf("SendGroupInitMessage hash:%s,  dummyId %v", m.Hash(), grm.GI.DummyID.GetHexString())
}

//组内广播密钥   for each定向发送 组内广播
func (ns *NetworkServerImpl) SendKeySharePiece(spm *model.ConsensusSharePieceMessage) {

	body, e := marshalConsensusSharePieceMessage(spm)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusSharePieceMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.KeyPieceMsg, Body: body}
	if spm.SI.SignMember.IsEqual(spm.Dest) {
		go MessageHandler.Handle(spm.SI.SignMember.String(), m)
		return
	}

	begin := time.Now()
	ns.net.SendWithGroupRely(spm.Dest.String(), spm.DummyID.GetHexString(), m)
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
	go MessageHandler.Handle(spkm.SI.SignMember.String(), m)

	begin := time.Now()
	ns.net.Multicast(spkm.DummyID.GetHexString(), m)
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
	go MessageHandler.Handle(cgm.SI.SignMember.String(), m)

	ns.net.TransmitToNeighbor(m)
	logger.Debugf("Broadcast GROUP_INIT_DONE_MSG, hash:%s, dummyId:%v", m.Hash(), cgm.GI.GIS.DummyID.GetHexString())

}

//-----------------------------------------------------------------组铸币----------------------------------------------

//铸币节点完成铸币，将blockheader  签名后发送至组内其他节点进行验证。组内广播
func (ns *NetworkServerImpl) SendCastVerify(ccm *model.ConsensusCastMessage) {
	body, e := marshalConsensusCastMessage(ccm)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusCastMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CastVerifyMsg, Body: body}

	var groupId groupsig.ID
	e1 := groupId.Deserialize(ccm.BH.GroupId)
	if e1 != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusCurrentMessage because of Deserialize groupsig id error::%s", e.Error())
		return
	}
	logger.Debugf("[peer]send CAST_VERIFY_MSG,%d-%d,cost time:%v,hash:%s", ccm.BH.Height, ccm.BH.QueueNumber, time.Since(ccm.BH.CurTime), m.Hash())
	begin := time.Now()
	ns.net.Multicast(groupId.GetHexString(), m)
	logger.Debugf("[peer]send CAST_VERIFY_MSG,%d-%d,invoke Multicast cost time:%v,hash:%s", ccm.BH.Height, ccm.BH.QueueNumber, time.Since(begin), m.Hash())
}

//组内节点  验证通过后 自身签名 广播验证块 组内广播  验证不通过 保持静默
func (ns *NetworkServerImpl) SendVerifiedCast(cvm *model.ConsensusVerifyMessage) {
	body, e := marshalConsensusVerifyMessage(cvm)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusVerifyMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.VerifiedCastMsg, Body: body}
	var groupId groupsig.ID
	e1 := groupId.Deserialize(cvm.BH.GroupId)
	if e1 != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusCurrentMessage because of Deserialize groupsig id error::%s", e.Error())
		return
	}
	logger.Debugf("[peer]send VARIFIED_CAST_MSG,%d-%d,cost time:%v,hash:%s", cvm.BH.Height, cvm.BH.QueueNumber, time.Since(cvm.BH.CurTime), m.Hash())
	begin := time.Now()
	ns.net.Multicast(groupId.GetHexString(), m)
	logger.Debugf("[peer]send VARIFIED_CAST_MSG,%d-%d,invoke Multicast cost time:%v,hash:%s", cvm.BH.Height, cvm.BH.QueueNumber, time.Since(begin), m.Hash())
}

//对外广播经过组签名的block 全网广播
func (ns *NetworkServerImpl) BroadcastNewBlock(cbm *model.ConsensusBlockMessage, group *NextGroup) {
	network.Logger.Debugf("broad new block %d-%d ,tx count:%d,cast and verify cost %v", cbm.Block.Header.Height, cbm.Block.Header.QueueNumber, len(cbm.Block.Header.Transactions), time.Since(cbm.Block.Header.CurTime))
	body, e := marshalConsensusBlockMessage(cbm)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusBlockMessage because of marshal error:%s", e.Error())
		return
	}
	blockMsg := network.Message{Code: network.NewBlockMsg, Body: body}
	blockHash := cbm.Block.Header.Hash

	nextCastGroupId := group.Gid.GetHexString()
	groupMembers := id2String(group.MemIds)

	ns.net.SpreadOverGroup(nextCastGroupId,groupMembers,blockMsg,blockHash.Bytes())
	network.Logger.Debugf("spread block %d-%d over group:%s,body size %d", cbm.Block.Header.Height, cbm.Block.Header.QueueNumber,nextCastGroupId, len(body))



	body, e = types.MarshalBlockHeader(cbm.Block.Header)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusBlockMessage because of marshal error:%s", e.Error())
		return
	}
	headerMsg := network.Message{Code:network.NewBlockHeaderMsg,Body:body}
	ns.net.TransmitToNeighbor(headerMsg)
	network.Logger.Debugf("spread block %d-%d header over group:%s,header size %d", cbm.Block.Header.Height, cbm.Block.Header.QueueNumber,nextCastGroupId, len(body))
}

//====================================建组前共识=======================

//开始建组
func (ns *NetworkServerImpl) SendCreateGroupRawMessage(msg *model.ConsensusCreateGroupRawMessage) {
	body, e := marshalConsensusCreateGroupRawMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusCreateGroupRawMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CreateGroupaRaw, Body: body}

	var groupId = msg.GI.ParentID
	ns.net.Multicast(groupId.GetHexString(), m)
}

func (ns *NetworkServerImpl) SendCreateGroupSignMessage(msg *model.ConsensusCreateGroupSignMessage) {
	body, e := marshalConsensusCreateGroupSignMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusCreateGroupSignMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CreateGroupSign, Body: body}

	ns.net.SendWithGroupRely(msg.Launcher.String(), msg.GI.ParentID.GetHexString(), m)
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

func marshalConsensusCastMessage(m *model.ConsensusCastMessage) ([]byte, error) {
	bh := types.BlockHeaderToPb(&m.BH)
	//groupId := m.GroupID.Serialize()
	si := signDataToPb(&m.SI)

	message := tas_middleware_pb.ConsensusBlockMessageBase{Bh: bh, Sign: si}
	return proto.Marshal(&message)
}

func marshalConsensusVerifyMessage(m *model.ConsensusVerifyMessage) ([]byte, error) {
	bh := types.BlockHeaderToPb(&m.BH)
	//groupId := m.GroupID.Serialize()
	si := signDataToPb(&m.SI)

	message := tas_middleware_pb.ConsensusBlockMessageBase{Bh: bh, Sign: si}
	return proto.Marshal(&message)
}

func marshalConsensusBlockMessage(m *model.ConsensusBlockMessage) ([]byte, error) {
	block := types.BlockToPb(&m.Block)
	if block == nil {
		network.Logger.Errorf("[peer]Block is nil while marshalConsensusBlockMessage")
	}
	message := tas_middleware_pb.ConsensusBlockMessage{Block: block}
	return proto.Marshal(&message)
}

//----------------------------------------------------------------------------------------------------------------------
func consensusGroupInitSummaryToPb(m *model.ConsensusGroupInitSummary) *tas_middleware_pb.ConsensusGroupInitSummary {
	beginTime, e := m.BeginTime.MarshalBinary()
	if e != nil {
		network.Logger.Errorf("ConsensusGroupInitSummary marshal begin time error:%s", e.Error())
		return nil
	}

	name := make([]byte, 0)
	for _, b := range m.Name {
		name = append(name, b)
	}
	message := tas_middleware_pb.ConsensusGroupInitSummary{
		ParentID:        m.ParentID.Serialize(),
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
