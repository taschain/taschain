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
)

type ConsensusHandler struct{
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

func memberExistIn(mems *[]model.PubKeyInfo, id groupsig.ID) bool {
	for _, member := range *mems {
		if member.ID.IsEqual(id) {
			return true
		}
	}
	return false
}

func (c *ConsensusHandler) Handle(sourceId string, msg network.Message)error{
	code := msg.Code
	body := msg.Body
	if !c.processor.Ready() {
		log.Printf("message ingored because processor not ready. code=%v\n", code)
		return fmt.Errorf("processor not ready yet")
	}
	switch code {
	case network.GroupInitMsg:
		m, e := unMarshalConsensusGroupRawMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusGroupRawMessage because of unmarshal error:%s", e.Error())
			return e
		}

		belongGroup := memberExistIn(&m.MEMS, c.processor.GetMinerID())

		var machines *StateMachines
		if belongGroup {
			machines = &GroupInsideMachines
		} else {
			machines = &GroupOutsideMachines
		}
		machines.GetMachine(m.GI.DummyID.GetHexString()).Transform(NewStateMsg(code, m, sourceId))
	case network.KeyPieceMsg:
		m, e := unMarshalConsensusSharePieceMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusSharePieceMessage because of unmarshal error:%s", e.Error())
			return e
		}
		GroupInsideMachines.GetMachine(m.DummyID.GetHexString()).Transform(NewStateMsg(code, m, sourceId))

	case network.SignPubkeyMsg:
		logger.Debugf("Receive SIGN_PUBKEY_MSG from:%s", sourceId)
		m, e := unMarshalConsensusSignPubKeyMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusSignPubKeyMessage because of unmarshal error:%s", e.Error())
			return  e
		}
		GroupInsideMachines.GetMachine(m.DummyID.GetHexString()).Transform(NewStateMsg(code, m, sourceId))

	case network.GroupInitDoneMsg:
		m, e := unMarshalConsensusGroupInitedMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusGroupInitedMessage because of unmarshal error%s", e.Error())
			return e
		}

		belongGroup := c.processor.ExistInDummyGroup(m.GI.GIS.DummyID)
		var machines *StateMachines
		if belongGroup {
			machines = &GroupInsideMachines
		} else {
			machines = &GroupOutsideMachines
		}
		machines.GetMachine(m.GI.GIS.DummyID.GetHexString()).Transform(NewStateMsg(code, m, sourceId))

	case network.CurrentGroupCastMsg:


	case network.CastVerifyMsg:
		m, e := unMarshalConsensusCastMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusCastMessage because of unmarshal error%s", e.Error())
			return e
		}
		c.processor.OnMessageCast(m)
	case network.VerifiedCastMsg:
		m, e := unMarshalConsensusVerifyMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusVerifyMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageVerify(m)

	case network.TransactionMsg, network.TransactionGotMsg:
		transactions, e := types.UnMarshalTransactions(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard TRANSACTION_GOT_MSG because of unmarshal error%s", e.Error())
			return e
		}
		var txHashes []common.Hash
		for _, tx := range transactions {
			txHashes = append(txHashes, tx.Hash)
		}
		c.processor.OnMessageNewTransactions(txHashes)
	case network.NewBlockMsg:
		m, e := unMarshalConsensusBlockMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusBlockMessage because of unmarshal error%s", e.Error())
			return  e
		}
		c.processor.OnMessageBlock(m)
		return nil
	case network.CreateGroupaRaw:
		m, e := unMarshalConsensusCreateGroupRawMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusCreateGroupRawMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageCreateGroupRaw(m)
		return nil
	case network.CreateGroupSign:
		m, e := unMarshalConsensusCreateGroupSignMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusCreateGroupSignMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageCreateGroupSign(m)
		return nil
	}
	return nil
}


//----------------------------------------------------------------------------------------------------------------------
func unMarshalConsensusGroupRawMessage(b []byte) (*model.ConsensusGroupRawMessage, error) {
	message := new(tas_middleware_pb.ConsensusGroupRawMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		network.Logger.Errorf("[handler]UnMarshalConsensusGroupRawMessage error:%s", e.Error())
		return nil, e
	}

	gi := pbToConsensusGroupInitSummary(message.ConsensusGroupInitSummary)

	sign := pbToSignData(message.Sign)

	var ids []model.PubKeyInfo
	for i := 0; i < len(message.Ids); i++ {
		pkInfo := pbToPubKeyInfo(message.Ids[i])
		ids = append(ids, *pkInfo)
	}

	base := model.BaseSignedMessage{SI: *sign}
	m := model.ConsensusGroupRawMessage{GI: *gi, MEMS: ids, BaseSignedMessage: base}
	return &m, nil
}

func unMarshalConsensusSharePieceMessage(b []byte) (*model.ConsensusSharePieceMessage, error) {
	m := new(tas_middleware_pb.ConsensusSharePieceMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("[handler]UnMarshalConsensusSharePieceMessage error:%s", e.Error())
		return nil, e
	}

	gisHash := common.BytesToHash(m.GISHash)
	var dummyId, dest groupsig.ID
	e1 := dummyId.Deserialize(m.DummyID)
	if e1 != nil {
		network.Logger.Errorf("[handler]groupsig.ID Deserialize error:%s", e1.Error())
		return nil, e1
	}

	e2 := dest.Deserialize(m.Dest)
	if e2 != nil {
		network.Logger.Errorf("[handler]groupsig.ID Deserialize error:%s", e2.Error())
		return nil, e2
	}

	share := pbToSharePiece(m.SharePiece)
	sign := pbToSignData(m.Sign)

	base := model.BaseSignedMessage{SI: *sign}
	message := model.ConsensusSharePieceMessage{GISHash: gisHash, DummyID: dummyId, Dest: dest, Share: *share, BaseSignedMessage: base}
	return &message, nil
}

func unMarshalConsensusSignPubKeyMessage(b []byte) (*model.ConsensusSignPubKeyMessage, error) {
	m := new(tas_middleware_pb.ConsensusSignPubKeyMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("[handler]unMarshalConsensusSignPubKeyMessage error:%s", e.Error())
		return nil, e
	}
	gisHash := common.BytesToHash(m.GISHash)
	var dummyId groupsig.ID
	e1 := dummyId.Deserialize(m.DummyID)
	if e1 != nil {
		network.Logger.Errorf("[handler]unMarshalConsensusSignPubKeyMessage error:%s", e1.Error())
		return nil, e1
	}

	var pubkey groupsig.Pubkey
	e2 := pubkey.Deserialize(m.SignPK)
	if e2 != nil {
		network.Logger.Errorf("[handler]unMarshalConsensusSignPubKeyMessage error:%s", e2.Error())
		return nil, e2
	}

	signData := pbToSignData(m.SignData)

	var sign groupsig.Signature
	e3 := sign.Deserialize(m.GISSign)
	if e3 != nil {
		network.Logger.Errorf("[handler]unMarshalConsensusSignPubKeyMessage error:%s", e3.Error())
		return nil, e3
	}

	base := model.BaseSignedMessage{SI: *signData}
	message := model.ConsensusSignPubKeyMessage{GISHash: gisHash, DummyID: dummyId, SignPK: pubkey, BaseSignedMessage: base, GISSign: sign}
	return &message, nil
}

func unMarshalConsensusGroupInitedMessage(b []byte) (*model.ConsensusGroupInitedMessage, error) {
	m := new(tas_middleware_pb.ConsensusGroupInitedMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("[handler]UnMarshalConsensusGroupInitedMessage error:%s", e.Error())
		return nil, e
	}

	gi := pbToStaticGroup(m.StaticGroupSummary)
	si := pbToSignData(m.Sign)

	base := model.BaseSignedMessage{SI: *si}
	message := model.ConsensusGroupInitedMessage{GI: *gi, BaseSignedMessage: base}
	return &message, nil
}

//--------------------------------------------组铸币--------------------------------------------------------------------
func unMarshalConsensusCurrentMessage(b []byte) (*model.ConsensusCurrentMessage, error) {
	m := new(tas_middleware_pb.ConsensusCurrentMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("[handler]UnMarshalConsensusCurrentMessage error:%s", e.Error())
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

func unMarshalConsensusCastMessage(b []byte) (*model.ConsensusCastMessage, error) {
	m := new(tas_middleware_pb.ConsensusBlockMessageBase)
	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("[handler]UnMarshalConsensusCastMessage error:%s", e.Error())
		return nil, e
	}

	bh := types.PbToBlockHeader(m.Bh)
	//var groupId groupsig.ID
	//e1 := groupId.Deserialize(m.GroupID)
	//if e1 != nil {
	//	logger.Errorf("groupsig.ID Deserialize error:%s", e1.Error())
	//	return nil, e1
	//}
	si := pbToSignData(m.Sign)

	base := model.BaseSignedMessage{SI: *si}
	message := model.ConsensusCastMessage{ConsensusBlockMessageBase: model.ConsensusBlockMessageBase{BH: *bh, BaseSignedMessage: base}}
	return &message, nil
}

func unMarshalConsensusVerifyMessage(b []byte) (*model.ConsensusVerifyMessage, error) {
	m := new(tas_middleware_pb.ConsensusBlockMessageBase)
	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("[handler]UnMarshalConsensusVerifyMessage error:%s", e.Error())
		return nil, e
	}

	bh := types.PbToBlockHeader(m.Bh)
	//var groupId groupsig.ID
	//e1 := groupId.Deserialize(m.GroupID)
	//if e1 != nil {
	//	logger.Errorf("groupsig.ID Deserialize error:%s", e1.Error())
	//	return nil, e1
	//}
	si := pbToSignData(m.Sign)
	base := model.BaseSignedMessage{SI: *si}
	message := model.ConsensusVerifyMessage{ConsensusBlockMessageBase: model.ConsensusBlockMessageBase{BH: *bh, BaseSignedMessage: base}}
	return &message, nil
}

func unMarshalConsensusBlockMessage(b []byte) (*model.ConsensusBlockMessage, error) {
	m := new(tas_middleware_pb.ConsensusBlockMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("[handler]unMarshalConsensusBlockMessage error:%s", e.Error())
		return nil, e
	}
	block := types.PbToBlock(m.Block)
	message := model.ConsensusBlockMessage{Block: *block}
	return &message, nil
}

func pbToConsensusGroupInitSummary(m *tas_middleware_pb.ConsensusGroupInitSummary) *model.ConsensusGroupInitSummary {
	var beginTime time.Time
	beginTime.UnmarshalBinary(m.BeginTime)

	name := [64]byte{}
	for i := 0; i < len(name); i++ {
		name[i] = m.Name[i]
	}

	var parentId groupsig.ID
	e1 := parentId.Deserialize(m.ParentID)

	if e1 != nil {
		network.Logger.Errorf("[handler]groupsig.ID Deserialize error:%s", e1.Error())
		return nil
	}

	var dummyID groupsig.ID
	e2 := dummyID.Deserialize(m.DummyID)

	if e1 != nil {
		network.Logger.Errorf("[handler]groupsig.ID Deserialize error:%s", e2.Error())
		return nil
	}

	var sign groupsig.Signature
	if err := sign.Deserialize(m.Signature); err != nil {
		network.Logger.Errorf("[handler]groupsig.Signature Deserialize error:%s", err.Error())
		return nil
	}

	mhash := common.Hash{}
	mhash.SetBytes(m.MemberHash)
	message := model.ConsensusGroupInitSummary{
		ParentID: parentId,
		Authority: *m.Authority,
		Name: name,
		DummyID: dummyID,
		BeginTime: beginTime,
		Members: *m.Members,
		MemberHash: mhash,
		Signature: sign,
		GetReadyHeight: *m.GetReadyHeight,
		BeginCastHeight: *m.BeginCastHeight,
		DismissHeight: *m.DismissHeight,
		TopHeight: *m.TopHeight,
		Extends:string(m.Extends),
	}
	return &message
}

func pbToSignData(s *tas_middleware_pb.SignData) *model.SignData {

	var sig groupsig.Signature
	e := sig.Deserialize(s.DataSign)
	if e != nil {
		network.Logger.Errorf("[handler]groupsig.Signature Deserialize error:%s", e.Error())
		return nil
	}

	id := groupsig.ID{}
	e1 := id.Deserialize(s.SignMember)
	if e1 != nil {
		network.Logger.Errorf("[handler]groupsig.ID Deserialize error:%s", e1.Error())
		return nil
	}
	sign := model.SignData{DataHash: common.BytesToHash(s.DataHash), DataSign: sig, SignMember: id}
	return &sign
}

func pbToSharePiece(s *tas_middleware_pb.SharePiece) *model.SharePiece {
	var share groupsig.Seckey
	var pub groupsig.Pubkey

	e1 := share.Deserialize(s.Seckey)
	if e1 != nil {
		network.Logger.Errorf("[handler]groupsig.Seckey Deserialize error:%s", e1.Error())
		return nil
	}

	e2 := pub.Deserialize(s.Pubkey)
	if e2 != nil {
		network.Logger.Errorf("[handler]groupsig.Pubkey Deserialize error:%s", e2.Error())
		return nil
	}

	sp := model.SharePiece{Share: share, Pub: pub}
	return &sp
}

func pbToStaticGroup(s *tas_middleware_pb.StaticGroupSummary) *model.StaticGroupSummary {
	var groupId groupsig.ID
	groupId.Deserialize(s.GroupID)

	var groupPk groupsig.Pubkey
	groupPk.Deserialize(s.GroupPK)

	gis := pbToConsensusGroupInitSummary(s.Gis)

	groupInfo := model.StaticGroupSummary{GroupID: groupId, GroupPK: groupPk, GIS: *gis}
	return &groupInfo
}

func pbToPubKeyInfo(p *tas_middleware_pb.PubKeyInfo) *model.PubKeyInfo {
	var id groupsig.ID
	var pk groupsig.Pubkey

	e1 := id.Deserialize(p.ID)
	if e1 != nil {
		network.Logger.Errorf("[handler]groupsig.ID Deserialize error:%s", e1.Error())
		return nil
	}

	e2 := pk.Deserialize(p.PublicKey)
	if e2 != nil {
		network.Logger.Errorf("[handler]groupsig.Pubkey Deserialize error:%s", e2.Error())
		return nil
	}

	pkInfo := model.NewPubKeyInfo(id, pk)
	return &pkInfo
}

func unMarshalConsensusCreateGroupRawMessage(b []byte) (*model.ConsensusCreateGroupRawMessage, error) {
	message := new(tas_middleware_pb.ConsensusCreateGroupRawMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		network.Logger.Errorf("[handler]UnMarshalConsensusCreateGroupRawMessage error:%s", e.Error())
		return nil, e
	}

	gi := pbToConsensusGroupInitSummary(message.ConsensusGroupInitSummary)

	sign := pbToSignData(message.Sign)

	ids := make([]groupsig.ID, 0)

	for _, idByte := range message.Ids {
		id := groupsig.DeserializeId(idByte)
		ids = append(ids, *id)
	}
	base := model.BaseSignedMessage{SI: *sign}
	m := model.ConsensusCreateGroupRawMessage{GI: *gi, IDs: ids, BaseSignedMessage: base}
	return &m, nil
}

func unMarshalConsensusCreateGroupSignMessage(b []byte) (*model.ConsensusCreateGroupSignMessage, error) {
	message := new(tas_middleware_pb.ConsensusCreateGroupSignMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		network.Logger.Errorf("[handler]UnMarshalConsensusCreateGroupSignMessage error:%s", e.Error())
		return nil, e
	}

	gi := pbToConsensusGroupInitSummary(message.ConsensusGroupInitSummary)

	sign := pbToSignData(message.Sign)
	base := model.BaseSignedMessage{SI: *sign}
	m := model.ConsensusCreateGroupSignMessage{GI: *gi, BaseSignedMessage: base}
	return &m, nil
}