package net

import (
	"middleware/pb"
	"consensus/model"
	"github.com/gogo/protobuf/proto"
	"middleware/types"
)

/*
**  Creator: pxf
**  Date: 2019/2/19 下午2:43
**  Description: 
*/


func marshalGroupInfo(gInfo *model.ConsensusGroupInitInfo) *tas_middleware_pb.ConsensusGroupInitInfo {
	mems := make([][]byte, gInfo.MemberSize())
	for i, mem := range gInfo.Mems {
		mems[i] = mem.Serialize()
	}

	return &tas_middleware_pb.ConsensusGroupInitInfo{
		GI:	consensusGroupInitSummaryToPb(&gInfo.GI),
		Mems: mems,
	}
}

func marshalConsensusGroupRawMessage(m *model.ConsensusGroupRawMessage) ([]byte, error) {
	gi := marshalGroupInfo(&m.GInfo)

	sign := signDataToPb(&m.SI)

	message := tas_middleware_pb.ConsensusGroupRawMessage{
		GInfo: gi,
		Sign: sign,
	}
	return proto.Marshal(&message)
}

func marshalConsensusSharePieceMessage(m *model.ConsensusSharePieceMessage) ([]byte, error) {
	share := sharePieceToPb(&m.Share)
	sign := signDataToPb(&m.SI)

	message := tas_middleware_pb.ConsensusSharePieceMessage{
		GHash: m.GHash.Bytes(),
		Dest: m.Dest.Serialize(),
		SharePiece: share,
		Sign: sign,
		MemCnt: &m.MemCnt,
	}
	return proto.Marshal(&message)
}

func marshalConsensusSignPubKeyMessage(m *model.ConsensusSignPubKeyMessage) ([]byte, error) {
	signData := signDataToPb(&m.SI)

	message := tas_middleware_pb.ConsensusSignPubKeyMessage{
		GHash: m.GHash.Bytes(),
		SignPK: m.SignPK.Serialize(),
		SignData: signData,
		GroupID: m.GroupID.Serialize(),
		MemCnt: &m.MemCnt,
	}
	return proto.Marshal(&message)
}
func marshalConsensusGroupInitedMessage(m *model.ConsensusGroupInitedMessage) ([]byte, error) {
	si := signDataToPb(&m.SI)
	message := tas_middleware_pb.ConsensusGroupInitedMessage{
		GHash: m.GHash.Bytes(),
		GroupID: m.GroupID.Serialize(),
		GroupPK: m.GroupPK.Serialize(),
		CreateHeight: &m.CreateHeight,
		ParentSign: m.ParentSign.Serialize(),
		Sign: si,
		MemCnt: &m.MemCnt,
		MemMask: m.MemMask,
	}
	return proto.Marshal(&message)
}

func marshalConsensusSignPubKeyReqMessage(m *model.ConsensusSignPubkeyReqMessage) ([]byte, error) {
	signData := signDataToPb(&m.SI)

	message := tas_middleware_pb.ConsensusSignPubkeyReqMessage{
		GroupID: m.GroupID.Serialize(),
		SignData: signData,
	}
	return proto.Marshal(&message)
}

//--------------------------------------------组铸币--------------------------------------------------------------------

func marshalConsensusCastMessage(m *model.ConsensusCastMessage) ([]byte, error) {
	bh := types.BlockHeaderToPb(&m.BH)
	//groupId := m.GroupID.Serialize()
	si := signDataToPb(&m.SI)

	hashs := make([][]byte, len(m.ProveHash))
	for i, h := range m.ProveHash {
		hashs[i] = h.Bytes()
	}

	message := tas_middleware_pb.ConsensusCastMessage{Bh: bh, Sign: si, ProveHash: hashs}
	return proto.Marshal(&message)
}

func marshalConsensusVerifyMessage(m *model.ConsensusVerifyMessage) ([]byte, error) {
	message := &tas_middleware_pb.ConsensusVerifyMessage{
		BlockHash: m.BlockHash.Bytes(),
		RandomSign: m.RandomSign.Serialize(),
		Sign: signDataToPb(&m.SI),
	}
	return proto.Marshal(message)
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
	message := tas_middleware_pb.ConsensusGroupInitSummary{
		Header: 		types.GroupToPbHeader(m.GHeader),
		Signature:       m.Signature.Serialize(),
	}
	return &message
}

func signDataToPb(s *model.SignData) *tas_middleware_pb.SignData {
	sign := tas_middleware_pb.SignData{DataHash: s.DataHash.Bytes(), DataSign: s.DataSign.Serialize(), SignMember: s.SignMember.Serialize(), Version: &s.Version}
	return &sign
}

func sharePieceToPb(s *model.SharePiece) *tas_middleware_pb.SharePiece {
	share := tas_middleware_pb.SharePiece{Seckey: s.Share.Serialize(), Pubkey: s.Pub.Serialize()}
	return &share
}

//func staticGroupInfoToPb(s *model.StaticGroupSummary) *tas_middleware_pb.StaticGroupSummary {
//	groupId := s.GroupID.Serialize()
//	groupPk := s.GroupPK.Serialize()
//
//	gis := consensusGroupInitSummaryToPb(&s.GIS)
//
//	groupInfo := tas_middleware_pb.StaticGroupSummary{GroupID: groupId, GroupPK: groupPk, Gis: gis}
//	return &groupInfo
//}
//
//func pubKeyInfoToPb(p *model.PubKeyInfo) *tas_middleware_pb.PubKeyInfo {
//	id := p.ID.Serialize()
//	pk := p.PK.Serialize()
//
//	pkInfo := tas_middleware_pb.PubKeyInfo{ID: id, PublicKey: pk}
//	return &pkInfo
//}

func marshalConsensusCreateGroupRawMessage(msg *model.ConsensusCreateGroupRawMessage) ([]byte, error) {
	gi := marshalGroupInfo(&msg.GInfo)

	sign := signDataToPb(&msg.SI)

	message := tas_middleware_pb.ConsensusCreateGroupRawMessage{GInfo: gi, Sign: sign}
	return proto.Marshal(&message)
}

func marshalConsensusCreateGroupSignMessage(msg *model.ConsensusCreateGroupSignMessage) ([]byte, error) {
	sign := signDataToPb(&msg.SI)

	message := tas_middleware_pb.ConsensusCreateGroupSignMessage{GHash: msg.GHash.Bytes(), Sign: sign}
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

func marshalCreateGroupPingMessage(msg *model.CreateGroupPingMessage) ([]byte, error) {
	si := signDataToPb(&msg.SI)
	message := &tas_middleware_pb.CreateGroupPingMessage{
		Sign:      si,
		PingID:   &msg.PingID,
		FromGroupID: msg.FromGroupID.Serialize(),
	}
	return proto.Marshal(message)
}

func marshalCreateGroupPongMessage(msg *model.CreateGroupPongMessage) ([]byte, error) {
	si := signDataToPb(&msg.SI)
	tb, _ := msg.Ts.MarshalBinary()
	message := &tas_middleware_pb.CreateGroupPongMessage{
		Sign:      si,
		PingID:   &msg.PingID,
		Ts: 	tb,
	}
	return proto.Marshal(message)
}