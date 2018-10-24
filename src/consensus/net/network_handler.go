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
	"middleware/statistics"
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

func memberExistIn(mems *[]model.PubKeyInfo, id groupsig.ID) bool {
	for _, member := range *mems {
		if member.ID.IsEqual(id) {
			return true
		}
	}
	return false
}

func (c *ConsensusHandler) Handle(sourceId string, msg network.Message) error {
	code := msg.Code
	body := msg.Body
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
			logger.Errorf("[handler]Discard ConsensusSharePieceMessage because of unmarshal error:%s", e.Error())
			return e
		}
		GroupInsideMachines.GetMachine(m.DummyID.GetHexString()).Transform(NewStateMsg(code, m, sourceId))

	case network.SignPubkeyMsg:
		t := time.Now()
		m, e := unMarshalConsensusSignPubKeyMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusSignPubKeyMessage because of unmarshal error:%s", e.Error())
			return e
		}
		t1 := time.Since(t)
		GroupInsideMachines.GetMachine(m.DummyID.GetHexString()).Transform(NewStateMsg(code, m, sourceId))
		t2 := time.Since(t)
		logger.Infof("SignPubKeyMsg receive %v, unMarshal cost %v, transform cost %v", t.Format("2006-01-02/15:04:05.000"), t1.String(), t2.String())
	case network.GroupInitDoneMsg:
		m, e := unMarshalConsensusGroupInitedMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusGroupInitedMessage because of unmarshal error%s", e.Error())
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

	case network.TransactionMsg, network.TransactionGotMsg:
		transactions, e := types.UnMarshalTransactions(body)
		if e != nil {
			logger.Errorf("[handler]Discard TRANSACTION_GOT_MSG because of unmarshal error%s", e.Error())
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
			logger.Errorf("[handler]Discard ConsensusBlockMessage because of unmarshal error%s", e.Error())
			return e
		}
		logger.Debugf("Rcv new block hash:%v,height:%d,totalQn:%d", m.Block.Header.Hash, m.Block.Header.Height, m.Block.Header.TotalQN)
		statistics.AddBlockLog(common.BootId, statistics.RcvNewBlock, m.Block.Header.Height, 0, len(m.Block.Transactions), -1,
			time.Now().UnixNano(), "", "", common.InstanceIndex, m.Block.Header.CurTime.UnixNano())
		c.processor.OnMessageBlock(m)
		return nil
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
	}
	return nil
}

//----------------------------------------------------------------------------------------------------------------------
func unMarshalConsensusGroupRawMessage(b []byte) (*model.ConsensusGroupRawMessage, error) {
	message := new(tas_middleware_pb.ConsensusGroupRawMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusGroupRawMessage error:%s", e.Error())
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
		logger.Errorf("[handler]UnMarshalConsensusSharePieceMessage error:%s", e.Error())
		return nil, e
	}

	gisHash := common.BytesToHash(m.GISHash)
	var dummyId, dest groupsig.ID
	e1 := dummyId.Deserialize(m.DummyID)
	if e1 != nil {
		logger.Errorf("[handler]groupsig.ID Deserialize error:%s", e1.Error())
		return nil, e1
	}

	e2 := dest.Deserialize(m.Dest)
	if e2 != nil {
		logger.Errorf("[handler]groupsig.ID Deserialize error:%s", e2.Error())
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
		logger.Errorf("[handler]unMarshalConsensusSignPubKeyMessage error:%s", e.Error())
		return nil, e
	}
	gisHash := common.BytesToHash(m.GISHash)
	var dummyId groupsig.ID
	e1 := dummyId.Deserialize(m.DummyID)
	if e1 != nil {
		logger.Errorf("[handler]unMarshalConsensusSignPubKeyMessage error:%s", e1.Error())
		return nil, e1
	}

	var pubkey groupsig.Pubkey
	e2 := pubkey.Deserialize(m.SignPK)
	if e2 != nil {
		logger.Errorf("[handler]unMarshalConsensusSignPubKeyMessage error:%s", e2.Error())
		return nil, e2
	}

	signData := pbToSignData(m.SignData)

	var sign groupsig.Signature
	e3 := sign.Deserialize(m.GISSign)
	if e3 != nil {
		logger.Errorf("[handler]unMarshalConsensusSignPubKeyMessage error:%s", e3.Error())
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
		logger.Errorf("[handler]UnMarshalConsensusGroupInitedMessage error:%s", e.Error())
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
		BH: *bh,
		ProveHash:hashs,
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
	var beginTime time.Time
	beginTime.UnmarshalBinary(m.BeginTime)

	name := [64]byte{}
	for i := 0; i < len(name); i++ {
		name[i] = m.Name[i]
	}

	var parentId groupsig.ID
	e1 := parentId.Deserialize(m.ParentID)

	if e1 != nil {
		logger.Errorf("[handler]groupsig.ID Deserialize error:%s", e1.Error())
		return nil
	}

	var dummyID groupsig.ID
	e2 := dummyID.Deserialize(m.DummyID)

	if e1 != nil {
		logger.Errorf("[handler]groupsig.ID Deserialize error:%s", e2.Error())
		return nil
	}

	var sign groupsig.Signature
	if err := sign.Deserialize(m.Signature); err != nil {
		logger.Errorf("[handler]groupsig.Signature Deserialize error:%s", err.Error())
		return nil
	}

	mhash := common.Hash{}
	mhash.SetBytes(m.MemberHash)
	message := model.ConsensusGroupInitSummary{
		ParentID:        parentId,
		PrevGroupID:     groupsig.DeserializeId(m.PrevGroupID),
		Authority:       *m.Authority,
		Name:            name,
		DummyID:         dummyID,
		BeginTime:       beginTime,
		Members:         *m.Members,
		MemberHash:      mhash,
		Signature:       sign,
		GetReadyHeight:  *m.GetReadyHeight,
		BeginCastHeight: *m.BeginCastHeight,
		DismissHeight:   *m.DismissHeight,
		TopHeight:       *m.TopHeight,
		Extends:         string(m.Extends),
	}
	return &message
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
	sign := model.SignData{DataHash: common.BytesToHash(s.DataHash), DataSign: sig, SignMember: id}
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
		logger.Errorf("[handler]groupsig.ID Deserialize error:%s", e1.Error())
		return nil
	}

	e2 := pk.Deserialize(p.PublicKey)
	if e2 != nil {
		logger.Errorf("[handler]groupsig.Pubkey Deserialize error:%s", e2.Error())
		return nil
	}

	pkInfo := model.NewPubKeyInfo(id, pk)
	return &pkInfo
}

func unMarshalConsensusCreateGroupRawMessage(b []byte) (*model.ConsensusCreateGroupRawMessage, error) {
	message := new(tas_middleware_pb.ConsensusCreateGroupRawMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusCreateGroupRawMessage error:%s", e.Error())
		return nil, e
	}

	gi := pbToConsensusGroupInitSummary(message.ConsensusGroupInitSummary)

	sign := pbToSignData(message.Sign)

	ids := make([]groupsig.ID, 0)

	for _, idByte := range message.Ids {
		id := groupsig.DeserializeId(idByte)
		ids = append(ids, id)
	}
	base := model.BaseSignedMessage{SI: *sign}
	m := model.ConsensusCreateGroupRawMessage{GI: *gi, IDs: ids, BaseSignedMessage: base}
	return &m, nil
}

func unMarshalConsensusCreateGroupSignMessage(b []byte) (*model.ConsensusCreateGroupSignMessage, error) {
	message := new(tas_middleware_pb.ConsensusCreateGroupSignMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusCreateGroupSignMessage error:%s", e.Error())
		return nil, e
	}

	gi := pbToConsensusGroupInitSummary(message.ConsensusGroupInitSummary)

	sign := pbToSignData(message.Sign)
	base := model.BaseSignedMessage{SI: *sign}
	m := model.ConsensusCreateGroupSignMessage{GI: *gi, BaseSignedMessage: base}
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
