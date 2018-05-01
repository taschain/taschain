package logical

import (
	"consensus/groupsig"
	"network/p2p"
	"taslog"

	"github.com/gogo/protobuf/proto"
	"pb"
	"core"
)

var logger = taslog.GetLogger(taslog.P2PConfig)

//----------------------------------------------------组初始化-----------------------------------------------------------
//广播 组初始化消息  组内广播
func SendGroupInitMessage(grm ConsensusGroupRawMessage) {
	body, e := marshalConsensusGroupRawMessage(&grm)
	if e != nil {
		logger.Error("Discard ConsensusGroupRawMessage because of marshal error!\n")
		return
	}
	m := p2p.Message{Code: p2p.GROUP_INIT_MSG, Body: body}
	for _, member := range grm.MEMS {
		p2p.Server.SendMessage(m, member.ID.GetHexString())
	}
}

//组内广播密钥   for each定向发送 组内广播
func SendKeySharePiece(spm ConsensusSharePieceMessage) {
	body, e := marshalConsensusSharePieceMessage(&spm)
	if e != nil {
		logger.Error("Discard ConsensusSharePieceMessage because of marshal error!\n")
		return
	}
	id := spm.Dest.GetHexString()
	m := p2p.Message{Code: p2p.KEY_PIECE_MSG, Body: body}
	p2p.Server.SendMessage(m, id)

}

//组初始化完成 广播组信息 全网广播
func BroadcastGroupInfo(cgm ConsensusGroupInitedMessage) {

	body, e := marshalConsensusGroupInitedMessage(&cgm)
	if e != nil {
		logger.Error("Discard ConsensusGroupInitedMessage because of marshal error!\n")
		return
	}
	m := p2p.Message{Code: p2p.GROUP_INIT_DONE_MSG, Body: body}

	conns := p2p.Server.Host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			p2p.Server.SendMessage(m, string(id))
		}
	}
}

//-----------------------------------------------------------------组铸币----------------------------------------------
//组内成员发现自己所在组成为铸币组 发消息通知全组 组内广播
//param: 组信息
//      SignData

func  SendCurrentGroupCast(ccm *ConsensusCurrentMessage) {
	//groupId := ccm.GroupID
	var memberIds []groupsig.ID
	//todo 从鸠兹获得
	body, e := marshalConsensusCurrentMessagee(ccm)
	if e != nil {
		logger.Error("Discard ConsensusCurrentMessage because of marshal error!\n")
		return
	}
	m := p2p.Message{Code: p2p.CURRENT_GROUP_CAST_MSG, Body: body}
	for _, memberId := range memberIds {
		p2p.Server.SendMessage(m, memberId.GetHexString())
	}
}

//铸币节点完成铸币，将blockheader  签名后发送至组内其他节点进行验证。组内广播
func  SendCastVerify(ccm *ConsensusCastMessage) {
	//groupId := ccm.GroupID
	var memberIds []groupsig.ID
	//todo 从鸠兹获得

	body, e := marshalConsensusCastMessage(ccm)
	if e != nil {
		logger.Error("Discard ConsensusCastMessage because of marshal error!\n")
		return
	}
	m := p2p.Message{Code: p2p.CAST_VERIFY_MSG, Body: body}
	for _, memberId := range memberIds {
		p2p.Server.SendMessage(m, memberId.GetHexString())
	}
}

//组内节点  验证通过后 自身签名 广播验证块 组内广播  验证不通过 保持静默
func  SendVerifiedCast(cvm *ConsensusVerifyMessage) {
	//groupId := ccm.GroupID
	var memberIds []groupsig.ID
	//todo 从鸠兹获得

	body, e := marshalConsensusVerifyMessage(cvm)
	if e != nil {
		logger.Error("Discard ConsensusVerifyMessage because of marshal error!\n")
		return
	}
	m := p2p.Message{Code: p2p.VARIFIED_CAST_MSG, Body: body}
	for _, memberId := range memberIds {
		p2p.Server.SendMessage(m, memberId.GetHexString())
	}
}

//对外广播经过组签名的block 全网广播
//todo 此处参数留空 等班德构造
func BroadcastNewBlock() {}

//----------------------------------------------组初始化---------------------------------------------------------------

func marshalConsensusGroupRawMessage(m *ConsensusGroupRawMessage) ([]byte, error) {
	gi := consensusGroupInitSummaryToPb(&m.GI)

	sign := signDataToPb(&m.SI)

	ids := make([]*tas_pb.PubKeyInfo, 0)
	for _, id := range m.MEMS {
		ids = append(ids, pubKeyInfoToPb(&id))
	}

	message := tas_pb.ConsensusGroupRawMessage{ConsensusGroupInitSummary: gi, Ids: ids, Sign: sign}
	return proto.Marshal(&message)
}


func marshalConsensusSharePieceMessage(m *ConsensusSharePieceMessage) ([]byte, error) {
	gisHash := m.GISHash.Bytes()
	dummyId := m.DummyID.Serialize()
	dest := m.Dest.Serialize()
	share := sharePieceToPb(&m.Share)
	sign := signDataToPb(&m.SI)

	message := tas_pb.ConsensusSharePieceMessage{GISHash: gisHash, DummyID: dummyId, Dest: dest, SharePiece: share, Sign: sign}
	return proto.Marshal(&message)
}


func marshalConsensusGroupInitedMessage(m *ConsensusGroupInitedMessage) ([]byte, error) {
	gi := staticGroupInfoToPb(&m.GI)
	si := signDataToPb(&m.SI)
	message := tas_pb.ConsensusGroupInitedMessage{StaticGroupInfo: gi, Sign: si}
	return proto.Marshal(&message)
}

//--------------------------------------------组铸币--------------------------------------------------------------------
func marshalConsensusCurrentMessagee(m *ConsensusCurrentMessage) ([]byte, error) {
	GroupID := m.GroupID
	PreHash := m.PreHash.Bytes()
	PreTime, e := m.PreTime.MarshalBinary()
	if e != nil {
		logger.Errorf("MarshalConsensusCurrentMessagee marshal PreTime error:%s\n", e.Error())
		return nil, e
	}

	BlockHeight := m.BlockHeight
	SI := signDataToPb(&m.SI)
	message := tas_pb.ConsensusCurrentMessage{GroupID: GroupID, PreHash: PreHash, PreTime: PreTime, BlockHeight: &BlockHeight, Sign: SI}
	return proto.Marshal(&message)
}

func marshalConsensusCastMessage(m *ConsensusCastMessage) ([]byte, error) {
	bh := blockHeaderToPb(&m.BH)
	groupId := m.GroupID.Serialize()
	si := signDataToPb(&m.SI)

	message := tas_pb.ConsensusBlockMessageBase{Bh: bh, GroupID: groupId, Sign: si}
	return proto.Marshal(&message)
}


func marshalConsensusVerifyMessage(m *ConsensusVerifyMessage) ([]byte, error) {
	bh := blockHeaderToPb(&m.BH)
	groupId := m.GroupID.Serialize()
	si := signDataToPb(&m.SI)

	message := tas_pb.ConsensusBlockMessageBase{Bh: bh, GroupID: groupId, Sign: si}
	return proto.Marshal(&message)
}

//----------------------------------------------------------------------------------------------------------------------
func consensusGroupInitSummaryToPb(m *ConsensusGroupInitSummary) *tas_pb.ConsensusGroupInitSummary {
	beginTime, e := m.BeginTime.MarshalBinary()
	if e != nil {
		logger.Errorf("ConsensusGroupInitSummary marshal begin time error:%s\n", e.Error())
		return nil
	}

	name := []byte{}
	for _, b := range m.Name {
		name = append(name, b)
	}
	message := tas_pb.ConsensusGroupInitSummary{ParentID: m.ParentID.Serialize(), Authority: &m.Authority,
		Name: name, DummyID: m.DummyID.Serialize(), BeginTime: beginTime}
	return &message
}


func signDataToPb(s *SignData) *tas_pb.SignData {
	sign := tas_pb.SignData{DataHash: s.DataHash.Bytes(), DataSign: s.DataSign.Serialize(), SignMember: s.SignMember.Serialize()}
	return &sign
}

func sharePieceToPb(s *SharePiece) *tas_pb.SharePiece {
	share := tas_pb.SharePiece{Seckey: s.Share.Serialize(), Pubkey: s.Pub.Serialize()}
	return &share
}

func staticGroupInfoToPb(s *StaticGroupInfo) *tas_pb.StaticGroupInfo {
	groupId := s.GroupID.Serialize()
	groupPk := s.GroupPK.Serialize()
	members := make([]*tas_pb.PubKeyInfo, 0)
	for _, m := range s.Members {
		member := pubKeyInfoToPb(&m)
		members = append(members, member)
	}
	gis := consensusGroupInitSummaryToPb(&s.GIS)

	groupInfo := tas_pb.StaticGroupInfo{GroupID: groupId, GroupPK: groupPk, Members: members, Gis: gis}
	return &groupInfo
}

func pubKeyInfoToPb(p *PubKeyInfo) *tas_pb.PubKeyInfo {
	id := p.ID.Serialize()
	pk := p.PK.Serialize()

	pkInfo := tas_pb.PubKeyInfo{ID: id, PublicKey: pk}
	return &pkInfo
}

func blockHeaderToPb(h *core.BlockHeader) *tas_pb.BlockHeader {
	hashes := h.Transactions
	hashBytes := make([][]byte, 0)
	for _, hash := range hashes {
		hashBytes = append(hashBytes, hash.Bytes())
	}
	preTime, e1 := h.PreTime.MarshalBinary()
	if e1 != nil {
		logger.Errorf("BlockHeaderToPb marshal pre time error:%s\n", e1.Error())
		return nil
	}

	curTime, e2 := h.CurTime.MarshalBinary()
	if e2 != nil {
		logger.Errorf("BlockHeaderToPb marshal cur time error:%s\n", e2.Error())
		return nil
	}

	header := tas_pb.BlockHeader{Hash: h.Hash.Bytes(), Height: &h.Height, PreHash: h.PreHash.Bytes(), PreTime: preTime,
		BlockHeight: &h.BlockHeight, QueueNumber: &h.QueueNumber, CurTime: curTime, Castor: h.Castor, Signature: h.Signature.Bytes(),
		Nonce: &h.Nonce, Transactions: hashBytes, TxTree: h.TxTree.Bytes(), ReceiptTree: h.ReceiptTree.Bytes(), StateTree: h.StateTree.Bytes(),
		ExtraData: h.ExtraData}
	return &header
}
