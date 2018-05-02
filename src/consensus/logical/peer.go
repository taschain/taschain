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

//全网广播组成员信息
func BroadcastMembersInfo(grm ConsensusGroupRawMessage) {
	body, e := marshalConsensusGroupRawMessage(&grm)
	if e != nil {
		logger.Errorf("Discard BroadcastMembersInfo because of marshal error:%s", e.Error())
		return
	}
	m := p2p.Message{Code: p2p.GROUP_MEMBER_MSG, Body: body}

	conns := p2p.Server.Host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			p2p.Server.SendMessage(m, p2p.ConvertToID(id))
		}
	}
}

//广播 组初始化消息  组内广播
func SendGroupInitMessage(grm ConsensusGroupRawMessage) {
	body, e := marshalConsensusGroupRawMessage(&grm)
	if e != nil {
		logger.Errorf("Discard send ConsensusGroupRawMessage because of marshal error:%s", e.Error())
		return
	}
	m := p2p.Message{Code: p2p.GROUP_INIT_MSG, Body: body}
	for _, member := range grm.MEMS {
		if member.ID.GetString() != "" {
			p2p.Server.SendMessage(m, member.ID.GetString())
		}
	}
}

//组内广播密钥   for each定向发送 组内广播
func SendKeySharePiece(spm ConsensusSharePieceMessage) {
	body, e := marshalConsensusSharePieceMessage(&spm)
	if e != nil {
		logger.Errorf("Discard send ConsensusSharePieceMessage because of marshal error:%s", e.Error())
		return
	}
	id := spm.Dest.GetString()
	m := p2p.Message{Code: p2p.KEY_PIECE_MSG, Body: body}
	p2p.Server.SendMessage(m, id)
}

//组内广播签名公钥
func SendSignPubKey(spkm ConsensusSignPubKeyMessage) {
	body, e := marshalConsensusSignPubKeyMessage(&spkm)
	if e != nil {
		logger.Errorf("Discard send ConsensusSignPubKeyMessage because of marshal error:%s", e.Error())
		return
	}
	m := p2p.Message{Code: p2p.SIGN_PUBKEY_MSG, Body: body}
	groupBroadcast(m, spkm.DummyID)
}

//组初始化完成 广播组信息 全网广播
func BroadcastGroupInfo(cgm ConsensusGroupInitedMessage) {
	body, e := marshalConsensusGroupInitedMessage(&cgm)
	if e != nil {
		logger.Errorf("Discard send ConsensusGroupInitedMessage because of marshal error:%s", e.Error())
		return
	}
	m := p2p.Message{Code: p2p.GROUP_INIT_DONE_MSG, Body: body}

	conns := p2p.Server.Host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			p2p.Server.SendMessage(m, p2p.ConvertToID(id))
		}
	}
}

//-----------------------------------------------------------------组铸币----------------------------------------------
//组内成员发现自己所在组成为铸币组 发消息通知全组 组内广播
//param: 组信息
//      SignData

func SendCurrentGroupCast(ccm *ConsensusCurrentMessage) {
	body, e := marshalConsensusCurrentMessagee(ccm)
	if e != nil {
		logger.Errorf("Discard send ConsensusCurrentMessage because of marshal error::%s", e.Error())
		return
	}
	m := p2p.Message{Code: p2p.CURRENT_GROUP_CAST_MSG, Body: body}
	var groupId groupsig.ID
	e1 := groupId.Deserialize(ccm.GroupID)
	if e1 != nil {
		logger.Errorf("Discard send ConsensusCurrentMessage because of Deserialize groupsig id error::%s", e.Error())
		return
	}
	groupBroadcast(m, groupId)
}

//铸币节点完成铸币，将blockheader  签名后发送至组内其他节点进行验证。组内广播
func SendCastVerify(ccm *ConsensusCastMessage) {
	body, e := marshalConsensusCastMessage(ccm)
	if e != nil {
		logger.Errorf("Discard send ConsensusCastMessage because of marshal error:%s", e.Error())
		return
	}
	m := p2p.Message{Code: p2p.CAST_VERIFY_MSG, Body: body}
	groupBroadcast(m, ccm.GroupID)
}

//组内节点  验证通过后 自身签名 广播验证块 组内广播  验证不通过 保持静默
func SendVerifiedCast(cvm *ConsensusVerifyMessage) {
	body, e := marshalConsensusVerifyMessage(cvm)
	if e != nil {
		logger.Errorf("Discard send ConsensusVerifyMessage because of marshal error:%s", e.Error())
		return
	}
	m := p2p.Message{Code: p2p.VARIFIED_CAST_MSG, Body: body}
	groupBroadcast(m, cvm.GroupID)
}

//对外广播经过组签名的block 全网广播
func BroadcastNewBlock(cbm *ConsensusBlockMessage) {
	body, e := marshalConsensusBlockMessage(cbm)
	if e != nil {
		logger.Errorf("Discard send ConsensusBlockMessage because of marshal error:%s", e.Error())
		return
	}
	m := p2p.Message{Code: p2p.NEW_BLOCK_MSG, Body: body}

	conns := p2p.Server.Host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			p2p.Server.SendMessage(m, p2p.ConvertToID(id))
		}
	}
}

//组内广播
func groupBroadcast(m p2p.Message, groupId groupsig.ID) {
	group := core.GroupChainImpl.GetGroupById(groupId.Serialize())
	if group == nil {
		logger.Errorf("Get nil group by id:%s\n", groupId.GetString())
		return
	}
	for _, member := range group.Members {
		var id groupsig.ID
		e := id.Deserialize(member.Id)
		if e != nil {
			logger.Errorf("Discard send ConsensusSignPubKeyMessage because of groupsig id deserialize error:%s", e.Error())
			return
		}
		p2p.Server.SendMessage(m, id.GetString())
	}
}

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

func marshalConsensusSignPubKeyMessage(m *ConsensusSignPubKeyMessage) ([]byte, error) {
	hash := m.GISHash.Bytes()
	dummyId := m.DummyID.Serialize()
	signPK := m.SignPK.Serialize()
	signData := signDataToPb(&m.SI)

	message := tas_pb.ConsensusSignPubKeyMessage{GISHash: hash, DummyID: dummyId, SignPK: signPK, SignData: signData}
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
		logger.Errorf("MarshalConsensusCurrentMessagee marshal PreTime error:%s", e.Error())
		return nil, e
	}

	BlockHeight := m.BlockHeight
	SI := signDataToPb(&m.SI)
	message := tas_pb.ConsensusCurrentMessage{GroupID: GroupID, PreHash: PreHash, PreTime: PreTime, BlockHeight: &BlockHeight, Sign: SI}
	return proto.Marshal(&message)
}

func marshalConsensusCastMessage(m *ConsensusCastMessage) ([]byte, error) {
	bh := core.BlockHeaderToPb(&m.BH)
	groupId := m.GroupID.Serialize()
	si := signDataToPb(&m.SI)

	message := tas_pb.ConsensusBlockMessageBase{Bh: bh, GroupID: groupId, Sign: si}
	return proto.Marshal(&message)
}

func marshalConsensusVerifyMessage(m *ConsensusVerifyMessage) ([]byte, error) {
	bh := core.BlockHeaderToPb(&m.BH)
	groupId := m.GroupID.Serialize()
	si := signDataToPb(&m.SI)

	message := tas_pb.ConsensusBlockMessageBase{Bh: bh, GroupID: groupId, Sign: si}
	return proto.Marshal(&message)
}

func marshalConsensusBlockMessage(m *ConsensusBlockMessage) ([]byte, error) {
	block := core.BlockToPb(&m.Block)
	id := m.GroupID.Serialize()
	sign := signDataToPb(&m.SI)
	message := tas_pb.ConsensusBlockMessage{Block: block, GroupID: id, SignData: sign}
	return proto.Marshal(&message)
}

//----------------------------------------------------------------------------------------------------------------------
func consensusGroupInitSummaryToPb(m *ConsensusGroupInitSummary) *tas_pb.ConsensusGroupInitSummary {
	beginTime, e := m.BeginTime.MarshalBinary()
	if e != nil {
		logger.Errorf("ConsensusGroupInitSummary marshal begin time error:%s", e.Error())
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
