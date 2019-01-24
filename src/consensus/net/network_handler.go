package net

import (
	"network"
	"github.com/gogo/protobuf/proto"
	"consensus/groupsig"
	"common"
	"time"
	"middleware/types"
	"middleware/pb"
	"log"
	"fmt"
	"consensus/model"
	"runtime/debug"
)

type ConsensusHandler struct {
	processor MessageProcessor
}

var MessageHandler = new(ConsensusHandler)

func (c *ConsensusHandler) Init(proc MessageProcessor) {
	c.processor = proc
	InitStateMachines()
}

func (c *ConsensusHandler) Processor() MessageProcessor {
	return c.processor
}

func (c *ConsensusHandler) ready() bool {
	return c.processor != nil && c.processor.Ready()
}

func (c *ConsensusHandler) Handle(sourceId string, msg network.Message) error {
	code := msg.Code
	body := msg.Body

	defer func() {
		if r := recover(); r != nil {
			common.DefaultLogger.Errorf("error：%v\n", r)
			s := debug.Stack()
			common.DefaultLogger.Errorf(string(s))
		}
	}()

	if !c.ready() {
		log.Printf("message ingored because processor not ready. code=%v\n", code)
		return fmt.Errorf("processor not ready yet")
	}
	switch code {
	case network.GroupInitMsg:
		m, e := unMarshalConsensusGroupRawMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusGroupRawMessage because of unmarshal error:%s", e.Error())
			return e
		}

		//belongGroup := m.GInfo.MemberExists(c.processor.GetMinerID())

		//var machines *StateMachines
		//if belongGroup {
		//	machines = &GroupInsideMachines
		//} else {
		//	machines = &GroupOutsideMachines
		//}
		GroupInsideMachines.GetMachine(m.GInfo.GI.GetHash().Hex()).Transform(NewStateMsg(code, m, sourceId))
	case network.KeyPieceMsg:
		m, e := unMarshalConsensusSharePieceMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusSharePieceMessage because of unmarshal error:%s", e.Error())
			return e
		}
		GroupInsideMachines.GetMachine(m.GHash.Hex()).Transform(NewStateMsg(code, m, sourceId))
		logger.Infof("SharepieceMsg receive from:%v, gHash:%v",sourceId, m.GHash.Hex())
	case network.SignPubkeyMsg:
		m, e := unMarshalConsensusSignPubKeyMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusSignPubKeyMessage because of unmarshal error:%s", e.Error())
			return e
		}
		GroupInsideMachines.GetMachine(m.GHash.Hex()).Transform(NewStateMsg(code, m, sourceId))
		logger.Infof("SignPubKeyMsg receive from:%v, gHash:%v, groupId:%v",sourceId, m.GHash.Hex(), m.GroupID.GetHexString())
	case network.GroupInitDoneMsg:
		m, e := unMarshalConsensusGroupInitedMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusGroupInitedMessage because of unmarshal error%s", e.Error())
			return e
		}
		logger.Infof("Rcv GroupInitDoneMsg from:%s,gHash:%s, groupId:%v", sourceId, m.GHash.Hex(), m.GroupID.GetHexString())

		//belongGroup := c.processor.ExistInGroup(m.GHash)
		//var machines *StateMachines
		//if belongGroup {
		//	machines = &GroupInsideMachines
		//} else {
		//	machines = &GroupOutsideMachines
		//}
		GroupInsideMachines.GetMachine(m.GHash.Hex()).Transform(NewStateMsg(code, m, sourceId))

	case network.CurrentGroupCastMsg:

	case network.CastVerifyMsg:
		m, e := unMarshalConsensusCastMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusCastMessage because of unmarshal error%s", e.Error())
			return e
		}
		c.processor.OnMessageCast(m)
	case network.VerifiedCastMsg:
		m, e := unMarshalConsensusVerifyMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusVerifyMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageVerify(m)

	case network.CreateGroupaRaw:
		m, e := unMarshalConsensusCreateGroupRawMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusCreateGroupRawMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageCreateGroupRaw(m)
		return nil
	case network.CreateGroupSign:
		m, e := unMarshalConsensusCreateGroupSignMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusCreateGroupSignMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageCreateGroupSign(m)
		return nil
	case network.CastRewardSignReq:
		m, e := unMarshalCastRewardReqMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard CastRewardSignReqMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageCastRewardSignReq(m)
	case network.CastRewardSignGot:
		m, e := unMarshalCastRewardSignMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard CastRewardSignMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageCastRewardSign(m)
	case network.AskSignPkMsg:
		m, e := unMarshalConsensusSignPKReqMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalConsensusSignPKReqMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageSignPKReq(m)
	case network.AnswerSignPkMsg:
		m, e := unMarshalConsensusSignPubKeyMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusSignPubKeyMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageSignPK(m)
	}

	return nil
}

//----------------------------------------------------------------------------------------------------------------------

func baseMessage(sign *tas_middleware_pb.SignData) *model.BaseSignedMessage {
	return &model.BaseSignedMessage{SI: *pbToSignData(sign)}
}

func pbToGroupInfo(gi *tas_middleware_pb.ConsensusGroupInitInfo) *model.ConsensusGroupInitInfo {
	gis := pbToConsensusGroupInitSummary(gi.GI)
	mems := make([]groupsig.ID, len(gi.Mems))
	for idx, mem := range gi.Mems {
		mems[idx] = groupsig.DeserializeId(mem)
	}
	return &model.ConsensusGroupInitInfo{
		GI:   *gis,
		Mems: mems,
	}
}

func unMarshalConsensusGroupRawMessage(b []byte) (*model.ConsensusGroupRawMessage, error) {
	message := new(tas_middleware_pb.ConsensusGroupRawMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusGroupRawMessage error:%s", e.Error())
		return nil, e
	}

	m := model.ConsensusGroupRawMessage{
		GInfo:             *pbToGroupInfo(message.GInfo),
		BaseSignedMessage: *baseMessage(message.Sign),
	}
	return &m, nil
}

func unMarshalConsensusSharePieceMessage(b []byte) (*model.ConsensusSharePieceMessage, error) {
	m := new(tas_middleware_pb.ConsensusSharePieceMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusSharePieceMessage error:%s", e.Error())
		return nil, e
	}

	gHash := common.BytesToHash(m.GHash)

	dest := groupsig.DeserializeId(m.Dest)

	share := pbToSharePiece(m.SharePiece)
	message := model.ConsensusSharePieceMessage{
		GHash:             gHash,
		Dest:              dest,
		Share:             *share,
		BaseSignedMessage: *baseMessage(m.Sign),
	}
	return &message, nil
}

func unMarshalConsensusSignPubKeyMessage(b []byte) (*model.ConsensusSignPubKeyMessage, error) {
	m := new(tas_middleware_pb.ConsensusSignPubKeyMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]unMarshalConsensusSignPubKeyMessage error:%s", e.Error())
		return nil, e
	}
	gisHash := common.BytesToHash(m.GHash)

	pk := groupsig.DeserializePubkeyBytes(m.SignPK)

	base := baseMessage(m.SignData)
	message := model.ConsensusSignPubKeyMessage{
		GHash:             gisHash,
		SignPK:            pk,
		GroupID:          groupsig.DeserializeId(m.GroupID),
		BaseSignedMessage: *base,
	}
	return &message, nil
}

func unMarshalConsensusGroupInitedMessage(b []byte) (*model.ConsensusGroupInitedMessage, error) {
	m := new(tas_middleware_pb.ConsensusGroupInitedMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusGroupInitedMessage error:%s", e.Error())
		return nil, e
	}

	ch := uint64(0)
	if m.CreateHeight != nil {
		ch = *m.CreateHeight
	}
	var sign groupsig.Signature
	if len(m.ParentSign) > 0 {
		sign.Deserialize(m.ParentSign)
	}
	message := model.ConsensusGroupInitedMessage{
		GHash:             common.BytesToHash(m.GHash),
		GroupID:           groupsig.DeserializeId(m.GroupID),
		GroupPK:           groupsig.DeserializePubkeyBytes(m.GroupPK),
		CreateHeight: 		ch,
		ParentSign: 		sign,
		BaseSignedMessage: *baseMessage(m.Sign),
	}
	return &message, nil
}

func unMarshalConsensusSignPKReqMessage(b []byte) (*model.ConsensusSignPubkeyReqMessage, error) {
	m := new(tas_middleware_pb.ConsensusSignPubkeyReqMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]unMarshalConsensusSignPKReqMessage error: %v", e.Error())
		return nil, e
	}
	message := &model.ConsensusSignPubkeyReqMessage{
		GroupID: groupsig.DeserializeId(m.GroupID),
		BaseSignedMessage: *baseMessage(m.SignData),
	}
	return message, nil
}

//--------------------------------------------组铸币--------------------------------------------------------------------
func unMarshalConsensusCurrentMessage(b []byte) (*model.ConsensusCurrentMessage, error) {
	m := new(tas_middleware_pb.ConsensusCurrentMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusCurrentMessage error:%s", e.Error())
		return nil, e
	}

	GroupID := m.GroupID
	PreHash := common.BytesToHash(m.PreHash)

	var PreTime time.Time
	PreTime.UnmarshalBinary(m.PreTime)

	BlockHeight := m.BlockHeight
	si := pbToSignData(m.Sign)
	base := model.BaseSignedMessage{SI: *si}
	message := model.ConsensusCurrentMessage{GroupID: GroupID, PreHash: PreHash, PreTime: PreTime, BlockHeight: *BlockHeight, BaseSignedMessage: base}
	return &message, nil
}

func pb2ConsensusBlockMessageBase(b []byte) (*model.ConsensusBlockMessageBase, error) {
	m := new(tas_middleware_pb.ConsensusBlockMessageBase)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]pb2ConsensusBlockMessageBase error:%s", e.Error())
		return nil, e
	}

	bh := types.PbToBlockHeader(m.Bh)

	si := pbToSignData(m.Sign)

	hashs := make([]common.Hash, len(m.ProveHash))
	for i, h := range m.ProveHash {
		hashs[i] = common.BytesToHash(h)
	}

	base := model.BaseSignedMessage{SI: *si}
	return &model.ConsensusBlockMessageBase{
		BH:                *bh,
		ProveHash:         hashs,
		BaseSignedMessage: base,
	}, nil
}
func unMarshalConsensusCastMessage(b []byte) (*model.ConsensusCastMessage, error) {
	base, err := pb2ConsensusBlockMessageBase(b)
	if err != nil {
		return nil, err
	}
	message := model.ConsensusCastMessage{ConsensusBlockMessageBase: *base}
	return &message, nil
}

func unMarshalConsensusVerifyMessage(b []byte) (*model.ConsensusVerifyMessage, error) {
	base, err := pb2ConsensusBlockMessageBase(b)
	if err != nil {
		return nil, err
	}
	message := model.ConsensusVerifyMessage{ConsensusBlockMessageBase: *base}
	return &message, nil
}

func unMarshalConsensusBlockMessage(b []byte) (*model.ConsensusBlockMessage, error) {
	m := new(tas_middleware_pb.ConsensusBlockMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]unMarshalConsensusBlockMessage error:%s", e.Error())
		return nil, e
	}
	block := types.PbToBlock(m.Block)
	message := model.ConsensusBlockMessage{Block: *block}
	return &message, nil
}

func pbToConsensusGroupInitSummary(m *tas_middleware_pb.ConsensusGroupInitSummary) *model.ConsensusGroupInitSummary {
	gh := types.PbToGroupHeader(m.Header)
	return &model.ConsensusGroupInitSummary{
		GHeader:   gh,
		Signature: *groupsig.DeserializeSign(m.Signature),
	}
}

func pbToSignData(s *tas_middleware_pb.SignData) *model.SignData {

	var sig groupsig.Signature
	e := sig.Deserialize(s.DataSign)
	if e != nil {
		logger.Errorf("[handler]groupsig.Signature Deserialize error:%s", e.Error())
		return nil
	}

	id := groupsig.ID{}
	e1 := id.Deserialize(s.SignMember)
	if e1 != nil {
		logger.Errorf("[handler]groupsig.ID Deserialize error:%s", e1.Error())
		return nil
	}

	v := int32(0)
	if s.Version != nil {
		v = *s.Version
	}
	sign := model.SignData{DataHash: common.BytesToHash(s.DataHash), DataSign: sig, SignMember: id, Version: v,}
	return &sign
}

func pbToSharePiece(s *tas_middleware_pb.SharePiece) *model.SharePiece {
	var share groupsig.Seckey
	var pub groupsig.Pubkey

	e1 := share.Deserialize(s.Seckey)
	if e1 != nil {
		logger.Errorf("[handler]groupsig.Seckey Deserialize error:%s", e1.Error())
		return nil
	}

	e2 := pub.Deserialize(s.Pubkey)
	if e2 != nil {
		logger.Errorf("[handler]groupsig.Pubkey Deserialize error:%s", e2.Error())
		return nil
	}

	sp := model.SharePiece{Share: share, Pub: pub}
	return &sp
}

//
//func pbToStaticGroup(s *tas_middleware_pb.StaticGroupSummary) *model.StaticGroupSummary {
//	var groupId groupsig.ID
//	groupId.Deserialize(s.GroupID)
//
//	var groupPk groupsig.Pubkey
//	groupPk.Deserialize(s.GroupPK)
//
//	gis := pbToConsensusGroupInitSummary(s.Gis)
//
//	groupInfo := model.StaticGroupSummary{GroupID: groupId, GroupPK: groupPk, GIS: *gis}
//	return &groupInfo
//}
//
//func pbToPubKeyInfo(p *tas_middleware_pb.PubKeyInfo) *model.PubKeyInfo {
//	var id groupsig.ID
//	var pk groupsig.Pubkey
//
//	e1 := id.Deserialize(p.ID)
//	if e1 != nil {
//		logger.Errorf("[handler]groupsig.ID Deserialize error:%s", e1.Error())
//		return nil
//	}
//
//	e2 := pk.Deserialize(p.PublicKey)
//	if e2 != nil {
//		logger.Errorf("[handler]groupsig.Pubkey Deserialize error:%s", e2.Error())
//		return nil
//	}
//
//	pkInfo := model.NewPubKeyInfo(id, pk)
//	return &pkInfo
//}

func unMarshalConsensusCreateGroupRawMessage(b []byte) (*model.ConsensusCreateGroupRawMessage, error) {
	message := new(tas_middleware_pb.ConsensusCreateGroupRawMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusCreateGroupRawMessage error:%s", e.Error())
		return nil, e
	}

	gi := pbToGroupInfo(message.GInfo)

	m := model.ConsensusCreateGroupRawMessage{
		GInfo:             *gi,
		BaseSignedMessage: *baseMessage(message.Sign),
	}
	return &m, nil
}

func unMarshalConsensusCreateGroupSignMessage(b []byte) (*model.ConsensusCreateGroupSignMessage, error) {
	message := new(tas_middleware_pb.ConsensusCreateGroupSignMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusCreateGroupSignMessage error:%s", e.Error())
		return nil, e
	}

	m := model.ConsensusCreateGroupSignMessage{
		GHash:             common.BytesToHash(message.GHash),
		BaseSignedMessage: *baseMessage(message.Sign),
	}
	return &m, nil
}

func pbToBonus(b *tas_middleware_pb.Bonus) *types.Bonus {
	return &types.Bonus{
		TxHash:     common.BytesToHash(b.TxHash),
		TargetIds:  b.TargetIds,
		BlockHash:  common.BytesToHash(b.BlockHash),
		GroupId:    b.GroupId,
		Sign:       b.Sign,
		TotalValue: *b.TotalValue,
	}
}

func unMarshalCastRewardReqMessage(b []byte) (*model.CastRewardTransSignReqMessage, error) {
	message := new(tas_middleware_pb.CastRewardTransSignReqMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		network.Logger.Errorf("[handler]unMarshalCastRewardReqMessage error:%s", e.Error())
		return nil, e
	}

	rw := pbToBonus(message.Reward)
	sign := pbToSignData(message.Sign)
	base := model.BaseSignedMessage{SI: *sign}

	signPieces := make([]groupsig.Signature, len(message.SignedPieces))
	for idx, sp := range message.SignedPieces {
		signPieces[idx] = *groupsig.DeserializeSign(sp)
	}

	m := &model.CastRewardTransSignReqMessage{
		BaseSignedMessage: base,
		Reward:            *rw,
		SignedPieces:      signPieces,
	}
	return m, nil
}

func unMarshalCastRewardSignMessage(b []byte) (*model.CastRewardTransSignMessage, error) {
	message := new(tas_middleware_pb.CastRewardTransSignMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		network.Logger.Errorf("[handler]unMarshalCastRewardSignMessage error:%s", e.Error())
		return nil, e
	}

	sign := pbToSignData(message.Sign)
	base := model.BaseSignedMessage{SI: *sign}

	m := &model.CastRewardTransSignMessage{
		BaseSignedMessage: base,
		ReqHash:           common.BytesToHash(message.ReqHash),
		BlockHash:         common.BytesToHash(message.BlockHash),
	}
	return m, nil
}
