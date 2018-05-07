package handler

import (
	"network/p2p"
	"consensus/mediator"
	"consensus/logical"
	"pb"
	"github.com/gogo/protobuf/proto"
	"consensus/groupsig"
	"common"
	"time"
	"core"
	"taslog"
	"core/net/handler"
)

var logger = taslog.GetLogger(taslog.P2PConfig)

type ConsensusHandler struct{}

func (c *ConsensusHandler) HandlerMessage(code uint32, body []byte, sourceId string) ([]byte, error) {
	switch code {
	case p2p.GROUP_MEMBER_MSG:
		m, e := unMarshalConsensusGroupRawMessage(body)
		if e != nil {
			logger.Errorf("Discard ConsensusGroupRawMessage because of unmarshal error:%s", e.Error())
			return nil, e
		}
		onGroupMemberReceived(*m)
	case p2p.GROUP_INIT_MSG:
		m, e := unMarshalConsensusGroupRawMessage(body)
		if e != nil {
			logger.Errorf("Discard ConsensusGroupRawMessage because of unmarshal error:%s", e.Error())
			return nil, e
		}
		mediator.Proc.OnMessageGroupInit(*m)
	case p2p.KEY_PIECE_MSG:
		m, e := unMarshalConsensusSharePieceMessage(body)
		if e != nil {
			logger.Errorf("Discard ConsensusSharePieceMessage because of unmarshal error:%s", e.Error())
			return nil, e
		}
		mediator.Proc.OnMessageSharePiece(*m)
	case p2p.SIGN_PUBKEY_MSG:
		m, e := unMarshalConsensusSignPubKeyMessage(body)
		if e != nil {
			logger.Errorf("Discard ConsensusSignPubKeyMessage because of unmarshal error:%s", e.Error())
			return nil, e
		}
		mediator.Proc.OnMessageSignPK(*m)
	case p2p.GROUP_INIT_DONE_MSG:
		m, e := unMarshalConsensusGroupInitedMessage(body)
		if e != nil {
			logger.Errorf("Discard ConsensusGroupInitedMessage because of unmarshal error%s", e.Error())
			return nil, e
		}
		mediator.Proc.OnMessageGroupInited(*m)

	case p2p.CURRENT_GROUP_CAST_MSG:
		m, e := unMarshalConsensusCurrentMessage(body)
		if e != nil {
			logger.Errorf("Discard ConsensusCurrentMessage because of unmarshal error%s", e.Error())
			return nil, e
		}
		mediator.Proc.OnMessageCurrent(*m)
	case p2p.CAST_VERIFY_MSG:
		m, e := unMarshalConsensusCastMessage(body)
		if e != nil {
			logger.Errorf("Discard ConsensusCastMessage because of unmarshal error%s", e.Error())
			return nil, e
		}
		mediator.Proc.OnMessageCast(*m)
	case p2p.VARIFIED_CAST_MSG:
		m, e := unMarshalConsensusVerifyMessage(body)
		if e != nil {
			logger.Errorf("Discard ConsensusVerifyMessage because of unmarshal error%s", e.Error())
			return nil, e
		}
		mediator.Proc.OnMessageVerify(*m)

	case p2p.TRANSACTION_GOT_MSG:
		transactions, e := handler.UnMarshalTransactions(body)
		if e != nil {
			logger.Errorf("Discard TRANSACTION_GOT_MSG because of unmarshal error%s", e.Error())
			return nil, e
		}
		var txHashes []common.Hash
		for _, tx := range transactions {
			txHashes = append(txHashes, tx.Hash)
		}
		mediator.Proc.OnMessageNewTransactions(txHashes)
	case p2p.NEW_BLOCK_MSG:
		m, e := unMarshalConsensusBlockMessage(body)
		if e != nil {
			logger.Errorf("Discard ConsensusBlockMessage because of unmarshal error%s", e.Error())
			return nil, e
		}
		b := mediator.Proc.OnMessageBlock(*m)
		bytes, e1 := core.MarshalBlock(b)
		if e1 != nil {
			logger.Errorf("Discard ConsensusBlockMessage because of marshal block error%s", e1.Error())
			return nil, e1
		}
		return bytes, nil
	}
	return nil, nil
}

//全网节点收到父亲节点广播的组信息，将组(没有组公钥的)上链
func onGroupMemberReceived(grm logical.ConsensusGroupRawMessage) {
	members := make([]core.Member, 0)
	for _, m := range grm.MEMS {
		mem := core.Member{Id: m.ID.Serialize(), PubKey: m.PK.Serialize()}
		members = append(members, mem)
	}
	group := core.Group{Dummy: grm.GI.DummyID.Serialize(), Members: members, Parent: grm.GI.ParentID.Serialize()}

	sender := grm.SI.SignMember.Serialize()
	signature := grm.SI.DataSign.Serialize()
	core.GroupChainImpl.AddGroup(&group, sender, signature)
}

//----------------------------------------------------------------------------------------------------------------------
func unMarshalConsensusGroupRawMessage(b []byte) (*logical.ConsensusGroupRawMessage, error) {
	message := new(tas_pb.ConsensusGroupRawMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("UnMarshalConsensusGroupRawMessage error:%s", e.Error())
		return nil, e
	}

	gi := pbToConsensusGroupInitSummary(message.ConsensusGroupInitSummary)

	sign := pbToSignData(message.Sign)

	ids := []logical.PubKeyInfo{}
	for i := 0; i < len(message.Ids); i++ {
		pkInfo := pbToPubKeyInfo(message.Ids[i])
		ids = append(ids, *pkInfo)
	}

	m := logical.ConsensusGroupRawMessage{GI: *gi, MEMS: ids, SI: *sign}
	return &m, nil
}

func unMarshalConsensusSharePieceMessage(b []byte) (*logical.ConsensusSharePieceMessage, error) {
	m := new(tas_pb.ConsensusSharePieceMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("UnMarshalConsensusSharePieceMessage error:%s", e.Error())
		return nil, e
	}

	gisHash := common.BytesToHash(m.GISHash)
	var dummyId, dest groupsig.ID
	e1 := dummyId.Deserialize(m.DummyID)
	if e1 != nil {
		logger.Errorf("groupsig.ID Deserialize error:%s", e1.Error())
		return nil, e1
	}

	e2 := dest.Deserialize(m.Dest)
	if e2 != nil {
		logger.Errorf("groupsig.ID Deserialize error:%s", e2.Error())
		return nil, e2
	}

	share := pbToSharePiece(m.SharePiece)
	sign := pbToSignData(m.Sign)

	message := logical.ConsensusSharePieceMessage{GISHash: gisHash, DummyID: dummyId, Dest: dest, Share: *share, SI: *sign}
	return &message, nil
}

func unMarshalConsensusSignPubKeyMessage(b []byte) (*logical.ConsensusSignPubKeyMessage, error) {
	m := new(tas_pb.ConsensusSignPubKeyMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("unMarshalConsensusSignPubKeyMessage error:%s", e.Error())
		return nil, e
	}
	gisHash := common.BytesToHash(m.GISHash)
	var dummyId groupsig.ID
	e1 := dummyId.Deserialize(m.DummyID)
	if e1 != nil {
		logger.Errorf("unMarshalConsensusSignPubKeyMessage error:%s", e1.Error())
		return nil, e1
	}

	var pubkey groupsig.Pubkey
	e2 := pubkey.Deserialize(m.SignPK)
	if e2 != nil {
		logger.Errorf("unMarshalConsensusSignPubKeyMessage error:%s", e2.Error())
		return nil, e1
	}

	signData := pbToSignData(m.SignData)

	message := logical.ConsensusSignPubKeyMessage{GISHash: gisHash, DummyID: dummyId, SignPK: pubkey, SI: *signData}
	return &message, nil
}

func unMarshalConsensusGroupInitedMessage(b []byte) (*logical.ConsensusGroupInitedMessage, error) {
	m := new(tas_pb.ConsensusGroupInitedMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("UnMarshalConsensusGroupInitedMessage error:%s", e.Error())
		return nil, e
	}

	gi := pbToStaticGroup(m.StaticGroupInfo)
	si := pbToSignData(m.Sign)
	message := logical.ConsensusGroupInitedMessage{GI: *gi, SI: *si}
	return &message, nil
}

//--------------------------------------------组铸币--------------------------------------------------------------------
func unMarshalConsensusCurrentMessage(b []byte) (*logical.ConsensusCurrentMessage, error) {
	m := new(tas_pb.ConsensusCurrentMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("UnMarshalConsensusCurrentMessage error:%s", e.Error())
		return nil, e
	}

	GroupID := m.GroupID
	PreHash := common.BytesToHash(m.PreHash)

	var PreTime time.Time
	PreTime.UnmarshalBinary(m.PreTime)

	BlockHeight := m.BlockHeight
	SI := pbToSignData(m.Sign)
	message := logical.ConsensusCurrentMessage{GroupID: GroupID, PreHash: PreHash, PreTime: PreTime, BlockHeight: *BlockHeight, SI: *SI}
	return &message, nil
}

func unMarshalConsensusCastMessage(b []byte) (*logical.ConsensusCastMessage, error) {
	m := new(tas_pb.ConsensusBlockMessageBase)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("UnMarshalConsensusCastMessage error:%s", e.Error())
		return nil, e
	}

	bh := handler.PbToBlockHeader(m.Bh)
	//var groupId groupsig.ID
	//e1 := groupId.Deserialize(m.GroupID)
	//if e1 != nil {
	//	logger.Errorf("groupsig.ID Deserialize error:%s", e1.Error())
	//	return nil, e1
	//}
	si := pbToSignData(m.Sign)
	message := logical.ConsensusCastMessage{ConsensusBlockMessageBase: logical.ConsensusBlockMessageBase{BH: *bh, SI: *si}}
	return &message, nil
}

func unMarshalConsensusVerifyMessage(b []byte) (*logical.ConsensusVerifyMessage, error) {
	m := new(tas_pb.ConsensusBlockMessageBase)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("UnMarshalConsensusVerifyMessage error:%s", e.Error())
		return nil, e
	}

	bh := handler.PbToBlockHeader(m.Bh)
	//var groupId groupsig.ID
	//e1 := groupId.Deserialize(m.GroupID)
	//if e1 != nil {
	//	logger.Errorf("groupsig.ID Deserialize error:%s", e1.Error())
	//	return nil, e1
	//}
	si := pbToSignData(m.Sign)
	message := logical.ConsensusVerifyMessage{ConsensusBlockMessageBase: logical.ConsensusBlockMessageBase{BH: *bh, SI: *si}}
	return &message, nil
}

func unMarshalConsensusBlockMessage(b []byte) (*logical.ConsensusBlockMessage, error) {
	m := new(tas_pb.ConsensusBlockMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("unMarshalConsensusBlockMessage error:%s", e.Error())
		return nil, e
	}
	block := handler.PbToBlock(m.Block)
	var groupId groupsig.ID
	e1 := groupId.Deserialize(m.GroupID)
	if e1 != nil {
		logger.Errorf("unMarshalConsensusBlockMessage error:%s", e1.Error())
		return nil, e
	}

	signData := pbToSignData(m.SignData)
	message := logical.ConsensusBlockMessage{Block: *block, GroupID: groupId, SI: *signData}
	return &message, nil
}

func pbToConsensusGroupInitSummary(m *tas_pb.ConsensusGroupInitSummary) *logical.ConsensusGroupInitSummary {
	var beginTime time.Time
	beginTime.UnmarshalBinary(m.BeginTime)

	name := [64]byte{}
	for i := 0; i < len(name); i++ {
		name[i] = m.Name[i]
	}

	var parentId groupsig.ID
	e1 := parentId.Deserialize(m.ParentID)

	if e1 != nil {
		logger.Errorf("groupsig.ID Deserialize error:%s", e1.Error())
		return nil
	}

	var dummyID groupsig.ID
	e2 := dummyID.Deserialize(m.DummyID)

	if e1 != nil {
		logger.Errorf("groupsig.ID Deserialize error:%s", e2.Error())
		return nil
	}
	message := logical.ConsensusGroupInitSummary{ParentID: parentId, Authority: *m.Authority,
		Name: name, DummyID: dummyID, BeginTime: beginTime}
	return &message
}

func pbToSignData(s *tas_pb.SignData) *logical.SignData {

	var sig groupsig.Signature
	e := sig.Deserialize(s.DataSign)
	if e != nil {
		logger.Errorf("groupsig.Signature Deserialize error:%s", e.Error())
		return nil
	}

	id := groupsig.ID{}
	e1 := id.Deserialize(s.SignMember)
	if e1 != nil {
		logger.Errorf("groupsig.ID Deserialize error:%s", e1.Error())
		return nil
	}
	sign := logical.SignData{DataHash: common.BytesToHash(s.DataHash), DataSign: sig, SignMember: id}
	return &sign
}

func pbToSharePiece(s *tas_pb.SharePiece) *logical.SharePiece {
	var share groupsig.Seckey
	var pub groupsig.Pubkey

	e1 := share.Deserialize(s.Seckey)
	if e1 != nil {
		logger.Errorf("groupsig.Seckey Deserialize error:%s", e1.Error())
		return nil
	}

	e2 := pub.Deserialize(s.Pubkey)
	if e2 != nil {
		logger.Errorf("groupsig.Pubkey Deserialize error:%s", e2.Error())
		return nil
	}

	sp := logical.SharePiece{Share: share, Pub: pub}
	return &sp
}

func pbToStaticGroup(s *tas_pb.StaticGroupInfo) *logical.StaticGroupInfo {
	var groupId groupsig.ID
	groupId.Deserialize(s.GroupID)

	var groupPk groupsig.Pubkey
	groupPk.Deserialize(s.GroupPK)

	members := make([]logical.PubKeyInfo, 0)
	for _, m := range s.Members {
		member := pbToPubKeyInfo(m)
		members = append(members, *member)
	}

	gis := pbToConsensusGroupInitSummary(s.Gis)

	groupInfo := logical.StaticGroupInfo{GroupID: groupId, GroupPK: groupPk, Members: members, GIS: *gis}
	return &groupInfo
}

func pbToPubKeyInfo(p *tas_pb.PubKeyInfo) *logical.PubKeyInfo {
	var id groupsig.ID
	var pk groupsig.Pubkey

	e1 := id.Deserialize(p.ID)
	if e1 != nil {
		logger.Errorf("groupsig.ID Deserialize error:%s", e1.Error())
		return nil
	}

	e2 := pk.Deserialize(p.PublicKey)
	if e2 != nil {
		logger.Errorf("groupsig.Pubkey Deserialize error:%s", e2.Error())
		return nil
	}

	pkInfo := logical.PubKeyInfo{ID: id, PK: pk}
	return &pkInfo
}
