package handler

import (
	"network/p2p"
	"consensus/mediator"
	"consensus/logical"
	"github.com/gogo/protobuf/proto"
	"consensus/groupsig"
	"common"
	"time"
	"core"
	"consensus/net"
	"network"
	"middleware/types"
	"middleware/pb"
	"log"
	"fmt"
)

type ConsensusHandler struct{}

func memberExistIn(mems *[]logical.PubKeyInfo, id groupsig.ID) bool {
	for _, member := range *mems {
		if member.ID.IsEqual(id) {
			return true
		}
	}
	return false
}

func (c *ConsensusHandler) HandlerMessage(code uint32, body []byte, sourceId string) ([]byte, error) {
	if !mediator.Proc.Ready() {
		log.Printf("message ingored because processor not ready. code=%v\n", code)
		return nil, fmt.Errorf("processor not ready yet")
	}
	switch code {
	case p2p.GROUP_MEMBER_MSG:
		m, e := unMarshalConsensusGroupRawMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusGroupRawMessage because of unmarshal error:%s", e.Error())
			return nil, e
		}
		onGroupMemberReceived(*m)
	case p2p.GROUP_INIT_MSG:
		m, e := unMarshalConsensusGroupRawMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusGroupRawMessage because of unmarshal error:%s", e.Error())
			return nil, e
		}

		belongGroup := memberExistIn(&m.MEMS, mediator.Proc.GetMinerID())

		var machine net.StateMachineTransform
		if belongGroup {
			machine = net.TimeSeq.GetInsideGroupStateMachine(m.GI.DummyID.GetHexString())
		} else {
			machine = net.TimeSeq.GetOutsideGroupStateMachine(m.GI.DummyID.GetHexString())
		}
		machine.Transform(net.NewStateMsg(code, m, sourceId, ""), func(msg interface{}) {
			mediator.Proc.OnMessageGroupInit(*msg.(*logical.ConsensusGroupRawMessage))
		})
		//mediator.Proc.OnMessageGroupInit(*m)
	case p2p.KEY_PIECE_MSG:
		m, e := unMarshalConsensusSharePieceMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusSharePieceMessage because of unmarshal error:%s", e.Error())
			return nil, e
		}
		machine := net.TimeSeq.GetInsideGroupStateMachine(m.DummyID.GetHexString())
		machine.Transform(net.NewStateMsg(code, m, sourceId, ""), func(msg interface{}) {
			mediator.Proc.OnMessageSharePiece(*msg.(*logical.ConsensusSharePieceMessage))
		})
		//mediator.Proc.OnMessageSharePiece(*m)
	case p2p.SIGN_PUBKEY_MSG:
		m, e := unMarshalConsensusSignPubKeyMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusSignPubKeyMessage because of unmarshal error:%s", e.Error())
			return nil, e
		}
		machine := net.TimeSeq.GetInsideGroupStateMachine(m.DummyID.GetHexString())
		machine.Transform(net.NewStateMsg(code, m, sourceId, ""), func(msg interface{}) {
			mediator.Proc.OnMessageSignPK(*msg.(*logical.ConsensusSignPubKeyMessage))
		})
		//mediator.Proc.OnMessageSignPK(*m)
	case p2p.GROUP_INIT_DONE_MSG:
		m, e := unMarshalConsensusGroupInitedMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusGroupInitedMessage because of unmarshal error%s", e.Error())
			return nil, e
		}

		belongGroup := memberExistIn(&m.GI.Members, mediator.Proc.GetMinerID())
		var machine net.StateMachineTransform
		if belongGroup { //组内状态机
			machine = net.TimeSeq.GetInsideGroupStateMachine(m.GI.GIS.DummyID.GetHexString())
		} else { //组外状态机
			machine = net.TimeSeq.GetOutsideGroupStateMachine(m.GI.GIS.DummyID.GetHexString())
		}

		machine.Transform(net.NewStateMsg(code, m, sourceId, ""), func(msg interface{}) {
			mediator.Proc.OnMessageGroupInited(*msg.(*logical.ConsensusGroupInitedMessage))
		})

	case p2p.CURRENT_GROUP_CAST_MSG:
		//m, e := unMarshalConsensusCurrentMessage(body)
		//if e != nil {
		//	logger.Errorf("Discard ConsensusCurrentMessage because of unmarshal error%s", e.Error())
		//	return nil, e
		//}
		//
		////machine := net.TimeSeq.GetBlockStateMachine(m.GroupID, m.BlockHeight)
		////stateMsg := net.NewStateMsg(code, m, sourceId, "")
		////machine.Transform(stateMsg, func(msg interface{}) {
		////	mediator.Proc.OnMessageCurrent(*msg.(*logical.ConsensusCurrentMessage))
		////})
		//
		//mediator.Proc.OnMessageCurrent(*m)
	case p2p.CAST_VERIFY_MSG:
		m, e := unMarshalConsensusCastMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusCastMessage because of unmarshal error%s", e.Error())
			return nil, e
		}
		//fmt.Println("Get CAST_VERIFY_MSG!")
		//machine := net.TimeSeq.GetBlockStateMachine(m.BH.GroupId, m.BH.Height)
		//key := net.GenerateBlockMachineKey(m.BH.GroupId, m.BH.Height, m.BH.Castor)
		//machine.Transform(net.NewStateMsg(code, m, sourceId, key), func(msg interface{}) {
		//	mediator.Proc.OnMessageCast(*msg.(*logical.ConsensusCastMessage))
		//})

		mediator.Proc.OnMessageCast(*m)
	case p2p.VARIFIED_CAST_MSG:
		m, e := unMarshalConsensusVerifyMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusVerifyMessage because of unmarshal error%s", e.Error())
			return nil, e
		}

		//machine := net.TimeSeq.GetBlockStateMachine(m.BH.GroupId, m.BH.Height)
		//key := net.GenerateBlockMachineKey(m.BH.GroupId, m.BH.Height, m.BH.Castor)
		//machine.Transform(net.NewStateMsg(code, m, sourceId, key), func(msg interface{}) {
		//	mediator.Proc.OnMessageVerify(*msg.(*logical.ConsensusVerifyMessage))
		//})

		mediator.Proc.OnMessageVerify(*m)

	case p2p.TRANSACTION_MSG,p2p.TRANSACTION_GOT_MSG:
		transactions, e := types.UnMarshalTransactions(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard TRANSACTION_GOT_MSG because of unmarshal error%s", e.Error())
			return nil, e
		}
		var txHashes []common.Hash
		for _, tx := range transactions {
			txHashes = append(txHashes, tx.Hash)
		}
		if code == p2p.TRANSACTION_GOT_MSG{
			core.Logger.Debugf("[BlockChain]consensus handler TRANSACTION_GOT_MSG from %s,count:%d,time:%v",sourceId,len(txHashes),time.Now())
		}
		mediator.Proc.OnMessageNewTransactions(txHashes)
	case p2p.NEW_BLOCK_MSG:
		m, e := unMarshalConsensusBlockMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard ConsensusBlockMessage because of unmarshal error%s", e.Error())
			return nil, e
		}

		//todo 此处为啥需要返回b, 接口显得不统一, 不好处理
		//b := &m.Block
		//belongGroup := mediator.Proc.IsMinerGroup(m.GroupID)
		//if belongGroup {
		//	machine := net.TimeSeq.GetBlockStateMachine(m.Block.Header.GroupId, m.Block.Header.Height)
		//	key := net.GenerateBlockMachineKey(m.Block.Header.GroupId, m.Block.Header.Height, m.Block.Header.Castor)
		//	machine.Transform(net.NewStateMsg(code, m, sourceId, key), func(msg interface{}) {
		//		mediator.Proc.OnMessageBlock(*msg.(*logical.ConsensusBlockMessage))
		//	})
		//} else {
		mediator.Proc.OnMessageBlock(*m)
		//}

		//b := mediator.Proc.OnMessageBlock(*m)
		//bytes, e1 := core.MarshalBlock(b)
		//if e1 != nil {
		//	logger.Errorf("Discard ConsensusBlockMessage because of marshal block error%s", e1.Error())
		//	return nil, e1
		//}
		return nil, nil
	}
	return nil, nil
}

//全网节点收到父亲节点广播的组信息，将组(没有组公钥的)上链
func onGroupMemberReceived(grm logical.ConsensusGroupRawMessage) {
	members := make([]types.Member, 0)
	for _, m := range grm.MEMS {
		mem := types.Member{Id: m.ID.Serialize(), PubKey: m.PK.Serialize()}
		members = append(members, mem)
	}
	group := types.Group{Dummy: grm.GI.DummyID.Serialize(), Members: members, Parent: grm.GI.ParentID.Serialize()}

	sender := grm.SI.SignMember.Serialize()
	signature := grm.SI.DataSign.Serialize()
	core.GroupChainImpl.AddGroup(&group, sender, signature)
}

//----------------------------------------------------------------------------------------------------------------------
func unMarshalConsensusGroupRawMessage(b []byte) (*logical.ConsensusGroupRawMessage, error) {
	message := new(tas_middleware_pb.ConsensusGroupRawMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		network.Logger.Errorf("[handler]UnMarshalConsensusGroupRawMessage error:%s", e.Error())
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

	message := logical.ConsensusSharePieceMessage{GISHash: gisHash, DummyID: dummyId, Dest: dest, Share: *share, SI: *sign}
	return &message, nil
}

func unMarshalConsensusSignPubKeyMessage(b []byte) (*logical.ConsensusSignPubKeyMessage, error) {
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

	message := logical.ConsensusSignPubKeyMessage{GISHash: gisHash, DummyID: dummyId, SignPK: pubkey, SI: *signData, GISSign: sign}
	return &message, nil
}

func unMarshalConsensusGroupInitedMessage(b []byte) (*logical.ConsensusGroupInitedMessage, error) {
	m := new(tas_middleware_pb.ConsensusGroupInitedMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("[handler]UnMarshalConsensusGroupInitedMessage error:%s", e.Error())
		return nil, e
	}

	gi := pbToStaticGroup(m.StaticGroupInfo)
	si := pbToSignData(m.Sign)
	message := logical.ConsensusGroupInitedMessage{GI: *gi, SI: *si}
	return &message, nil
}

//--------------------------------------------组铸币--------------------------------------------------------------------
func unMarshalConsensusCurrentMessage(b []byte) (*logical.ConsensusCurrentMessage, error) {
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
	SI := pbToSignData(m.Sign)
	message := logical.ConsensusCurrentMessage{GroupID: GroupID, PreHash: PreHash, PreTime: PreTime, BlockHeight: *BlockHeight, SI: *SI}
	return &message, nil
}

func unMarshalConsensusCastMessage(b []byte) (*logical.ConsensusCastMessage, error) {
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
	message := logical.ConsensusCastMessage{ConsensusBlockMessageBase: logical.ConsensusBlockMessageBase{BH: *bh, SI: *si}}
	return &message, nil
}

func unMarshalConsensusVerifyMessage(b []byte) (*logical.ConsensusVerifyMessage, error) {
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
	message := logical.ConsensusVerifyMessage{ConsensusBlockMessageBase: logical.ConsensusBlockMessageBase{BH: *bh, SI: *si}}
	return &message, nil
}

func unMarshalConsensusBlockMessage(b []byte) (*logical.ConsensusBlockMessage, error) {
	m := new(tas_middleware_pb.ConsensusBlockMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("[handler]unMarshalConsensusBlockMessage error:%s", e.Error())
		return nil, e
	}
	block := types.PbToBlock(m.Block)
	var groupId groupsig.ID
	e1 := groupId.Deserialize(m.GroupID)
	if e1 != nil {
		network.Logger.Errorf("[handler]unMarshalConsensusBlockMessage error:%s", e1.Error())
		return nil, e
	}

	signData := pbToSignData(m.SignData)
	message := logical.ConsensusBlockMessage{Block: *block, GroupID: groupId, SI: *signData}
	return &message, nil
}

func pbToConsensusGroupInitSummary(m *tas_middleware_pb.ConsensusGroupInitSummary) *logical.ConsensusGroupInitSummary {
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
	message := logical.ConsensusGroupInitSummary{ParentID: parentId, Authority: *m.Authority,
		Name: name, DummyID: dummyID, BeginTime: beginTime}
	return &message
}

func pbToSignData(s *tas_middleware_pb.SignData) *logical.SignData {

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
	sign := logical.SignData{DataHash: common.BytesToHash(s.DataHash), DataSign: sig, SignMember: id}
	return &sign
}

func pbToSharePiece(s *tas_middleware_pb.SharePiece) *logical.SharePiece {
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

	sp := logical.SharePiece{Share: share, Pub: pub}
	return &sp
}

func pbToStaticGroup(s *tas_middleware_pb.StaticGroupInfo) *logical.StaticGroupInfo {
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

	beginHeight := *s.BeginHeight
	groupInfo := logical.StaticGroupInfo{GroupID: groupId, GroupPK: groupPk, Members: members, GIS: *gis, BeginHeight: beginHeight}
	return &groupInfo
}

func pbToPubKeyInfo(p *tas_middleware_pb.PubKeyInfo) *logical.PubKeyInfo {
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

	pkInfo := logical.PubKeyInfo{ID: id, PK: pk}
	return &pkInfo
}
