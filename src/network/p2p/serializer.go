package p2p

import (
	"pb"
	"core"
	"github.com/golang/protobuf/proto"
	"common"
	"time"
	"consensus/logical"
	"consensus/groupsig"
	"network/biz"
)

//--------------------------------------Transactions-------------------------------------------------------------------
func MarshalTransaction(t *core.Transaction) ([]byte, error) {
	transaction := transactionToPb(t)
	return proto.Marshal(transaction)
}

func UnMarshalTransaction(b []byte) (*core.Transaction, error) {
	t := new(tas_pb.Transaction)
	error := proto.Unmarshal(b, t)
	if error != nil {
		logger.Errorf("Unmarshal transaction error:%s\n", error.Error())
		return &core.Transaction{}, error
	}
	transaction := pbToTransaction(t)
	return transaction, nil
}

func MarshalTransactions(txs []*core.Transaction) ([]byte, error) {
	transactions := transactionsToPb(txs)
	transactionSlice := tas_pb.TransactionSlice{Transactions: transactions}
	return proto.Marshal(&transactionSlice)
}

func UnMarshalTransactions(b []byte) ([]*core.Transaction, error) {
	ts := new(tas_pb.TransactionSlice)
	error := proto.Unmarshal(b, ts)
	if error != nil {
		logger.Errorf("Unmarshal transactions error:%s\n", error.Error())
		return nil, error
	}

	result := pbToTransactions(ts.Transactions)
	return result, nil
}

func MarshalTransactionRequestMessage(m *biz.TransactionRequestMessage) ([]byte, error) {
	txHashes := make([][]byte, 0)
	for _, txHash := range m.TransactionHashes {
		txHashes = append(txHashes, txHash.Bytes())
	}

	sourceId := []byte(m.SourceId)

	requestTime, e := m.RequestTime.MarshalBinary()
	if e != nil {
		logger.Error("TransactionRequestMessage request time marshal error:%s\n", e.Error())
	}
	message := tas_pb.TransactionRequestMessage{TransactionHashes: txHashes, SourceId: sourceId, RequestTime: requestTime}
	return proto.Marshal(&message)
}

func UnMarshalTransactionRequestMessage(b []byte) (*biz.TransactionRequestMessage, error) {
	m := new(tas_pb.TransactionRequestMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("UnMarshal TransactionRequestMessage error:%s\n", e.Error())
		return nil, e
	}

	txHashes := make([]common.Hash, 0)
	for _, txHash := range m.TransactionHashes {
		txHashes = append(txHashes, common.BytesToHash(txHash))
	}

	sourceId := string(m.SourceId)

	var requestTime time.Time
	e1 := requestTime.UnmarshalBinary(m.RequestTime)
	if e1 != nil {
		logger.Error("MarshalTransactionRequestMessage request time unmarshal error:%s\n", e1.Error())
	}
	message := biz.TransactionRequestMessage{TransactionHashes: txHashes, SourceId: sourceId, RequestTime: requestTime}
	return &message, nil
}

func transactionToPb(t *core.Transaction) *tas_pb.Transaction {
	transaction := tas_pb.Transaction{Data: t.Data, Value: &t.Value, Nonce: &t.Nonce, Source: t.Source.Bytes(),
		Target: t.Target.Bytes(), GasLimit: &t.GasLimit, GasPrice: &t.GasPrice, Hash: t.Hash.Bytes(), ExtraData: t.ExtraData}
	return &transaction
}

func pbToTransaction(t *tas_pb.Transaction) *core.Transaction {
	source := common.BytesToAddress(t.Source)
	target := common.BytesToAddress(t.Target)
	transaction := core.Transaction{Data: t.Data, Value: *t.Value, Nonce: *t.Nonce, Source: &source,
		Target: &target, GasLimit: *t.GasLimit, GasPrice: *t.GasPrice, Hash: common.BytesToHash(t.Hash), ExtraData: t.ExtraData}
	return &transaction
}

func transactionsToPb(txs []*core.Transaction) []*tas_pb.Transaction {
	transactions := make([]*tas_pb.Transaction,0)
	for _, t := range txs {
		transaction := transactionToPb(t)
		transactions = append(transactions, transaction)
	}
	return transactions
}

func pbToTransactions(txs []*tas_pb.Transaction) []*core.Transaction {
	result := make([]*core.Transaction, 0)
	for _, t := range txs {
		transaction := pbToTransaction(t)
		result = append(result, transaction)
	}
	return result
}

//--------------------------------------------------Block---------------------------------------------------------------
func MarshalBlock(b *core.Block) ([]byte, error) {
	block := blockToPb(b)
	return proto.Marshal(block)
}

func UnMarshalBlock(bytes []byte) (*core.Block, error) {
	b := new(tas_pb.Block)
	error := proto.Unmarshal(bytes, b)
	if error != nil {
		logger.Errorf("Unmarshal Block error:%s\n", error.Error())
		return nil, error
	}
	block := pbToBlock(b)
	return block, nil
}

func MarshalBlocks(bs []*core.Block) ([]byte, error) {
	blocks := make([]*tas_pb.Block, 0)
	for _, b := range bs {
		block := blockToPb(b)
		blocks = append(blocks, block)
	}
	blockSlice := tas_pb.BlockSlice{Blocks: blocks}
	return proto.Marshal(&blockSlice)
}

func UnMarshalBlocks(b []byte) ([]*core.Block, error) {
	blockSlice := new(tas_pb.BlockSlice)
	error := proto.Unmarshal(b, blockSlice)
	if error != nil {
		logger.Errorf("Unmarshal Blocks error:%s\n", error.Error())
		return nil, error
	}
	blocks := blockSlice.Blocks
	result := make([]*core.Block, 0)

	for _, b := range blocks {
		block := pbToBlock(b)
		result = append(result, block)
	}
	return result, nil
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

func pbToBlockHeader(h *tas_pb.BlockHeader) *core.BlockHeader {

	hashBytes := h.Transactions
	hashes := make([]common.Hash, 0)
	for _, hashByte := range hashBytes {
		hash := common.BytesToHash(hashByte)
		hashes = append(hashes, hash)
	}

	var preTime time.Time
	preTime.UnmarshalBinary(h.PreTime)
	var curTime time.Time
	curTime.UnmarshalBinary(h.CurTime)

	header := core.BlockHeader{Hash: common.BytesToHash(h.Hash), Height: *h.Height, PreHash: common.BytesToHash(h.PreHash), PreTime: preTime,
		BlockHeight: *h.BlockHeight, QueueNumber: *h.QueueNumber, CurTime: curTime, Castor: h.Castor, Signature: common.BytesToHash(h.Signature),
		Nonce: *h.Nonce, Transactions: hashes, TxTree: common.BytesToHash(h.TxTree), ReceiptTree: common.BytesToHash(h.ReceiptTree), StateTree: common.BytesToHash(h.StateTree),
		ExtraData: h.ExtraData}
	return &header
}

func blockToPb(b *core.Block) *tas_pb.Block {
	header := blockHeaderToPb(b.Header)
	transactons := transactionsToPb(b.Transactions)
	block := tas_pb.Block{Header: header, Transactions: transactons}
	return &block
}

func pbToBlock(b *tas_pb.Block) *core.Block {
	h := pbToBlockHeader(b.Header)
	txs := pbToTransactions(b.Transactions)
	block := core.Block{Header: h, Transactions: txs}
	return &block
}

//func MarshalBlockMap(blockMap map[uint64]core.Block) ([]byte, error) {
//	m := make(map[uint64]*tas_pb.Block, 0)
//	for id, block := range blockMap {
//		m[id] = blockToPb(&block)
//	}
//	bMap := tas_pb.BlockMap{Blocks: m}
//	return proto.Marshal(&bMap)
//}
//
//func UnMarshalBlockMap(b []byte) (map[uint64]*core.Block, error) {
//	blockMap := new(tas_pb.BlockMap)
//	e := proto.Unmarshal(b, blockMap)
//	if e != nil {
//		logger.Errorf("UnMarshalBlockMap error:%s\n", e.Error())
//		return nil, e
//	}
//	m := make(map[uint64]*core.Block, 0)
//	for id, block := range blockMap.Blocks {
//		m[id] = pbToBlock(block)
//	}
//	return m, nil
//}

//--------------------------------------------------Group---------------------------------------------------------------

func MarshalMember(m *core.Member) ([]byte, error) {
	member := memberToPb(m)
	return proto.Marshal(member)
}

func UnMarshalMember(b []byte) (*core.Member, error) {
	member := new(tas_pb.Member)
	e := proto.Unmarshal(b, member)
	if e != nil {
		logger.Errorf("UnMarshalMember error:%s\n", e.Error())
		return nil, e
	}
	m := pbToMember(member)
	return m, nil
}

func memberToPb(m *core.Member) *tas_pb.Member {
	member := tas_pb.Member{Id: m.Id, PubKey: m.PubKey}
	return &member
}

func pbToMember(m *tas_pb.Member) *core.Member {
	member := core.Member{Id: m.Id, PubKey: m.PubKey}
	return &member
}

func MarshalGroup(g *core.Group) ([]byte, error) {
	group := groupToPb(g)
	return proto.Marshal(group)
}

func UnMarshalGroup(b []byte) (*core.Group, error) {
	group := new(tas_pb.Group)
	e := proto.Unmarshal(b, group)
	if e != nil {
		logger.Errorf("UnMarshalGroup error:%s\n", e.Error())
		return nil, e
	}
	g := pbToGroup(group)
	return g, nil
}

func groupToPb(g *core.Group) *tas_pb.Group {
	members := make([]*tas_pb.Member, 0)
	for _, m := range g.Members {
		member := memberToPb(&m)
		members = append(members, member)
	}
	group := tas_pb.Group{Id: g.Id, Members: members, PubKey: g.PubKey, Parent: g.PubKey, Dummy: g.Dummy, Signature: g.Signature}
	return &group
}

func pbToGroup(g *tas_pb.Group) *core.Group {
	members := make([]core.Member, 0)
	for _, m := range g.Members {
		member := pbToMember(m)
		members = append(members, *member)
	}
	group := core.Group{Id: g.Id, Members: members, PubKey: g.PubKey, Parent: g.PubKey, Dummy: g.Dummy, Signature: g.Signature}
	return &group
}

func MarshalGroupMap(groupMap map[uint64]core.Group) ([]byte, error) {
	g := make(map[uint64]*tas_pb.Group,0)
	for height, group := range groupMap {
		g[height] = groupToPb(&group)
	}
	bMap := tas_pb.GroupMap{Groups: g}
	return proto.Marshal(&bMap)
}

func UnMarshalGroupMap(b []byte) (map[uint64]core.Group, error) {
	groupMap := new(tas_pb.GroupMap)
	e := proto.Unmarshal(b, groupMap)
	if e != nil {
		logger.Errorf("UnMarshalGroupMap error:%s\n", e.Error())
		return nil, e
	}
	g := make(map[uint64]core.Group, 0)
	for height, group := range groupMap.Groups {
		g[height] = *pbToGroup(group)
	}
	return g, nil
}

//----------------------------------------------组初始化---------------------------------------------------------------

func MarshalConsensusGroupRawMessage(m *logical.ConsensusGroupRawMessage) ([]byte, error) {
	gi := consensusGroupInitSummaryToPb(&m.GI)

	sign := signDataToPb(&m.SI)

	ids := make([]*tas_pb.PubKeyInfo, 0)
	for _, id := range m.MEMS {
		ids = append(ids, pubKeyInfoToPb(&id))
	}

	message := tas_pb.ConsensusGroupRawMessage{ConsensusGroupInitSummary: gi, Ids: ids, Sign: sign}
	return proto.Marshal(&message)
}

func UnMarshalConsensusGroupRawMessage(b []byte) (*logical.ConsensusGroupRawMessage, error) {
	message := new(tas_pb.ConsensusGroupRawMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("UnMarshalConsensusGroupRawMessage error:%s\n", e.Error())
		return nil, e
	}

	gi := pbToConsensusGroupInitSummary(message.ConsensusGroupInitSummary)

	sign := pbToSignData(message.Sign)

	ids := [5]logical.PubKeyInfo{}
	for i := 0; i < 5; i++ {
		pkInfo := pbToPubKeyInfo(message.Ids[i])
		ids[i] = *pkInfo
	}

	m := logical.ConsensusGroupRawMessage{GI: *gi, MEMS: ids, SI: *sign}
	return &m, nil
}

func MarshalConsensusSharePieceMessage(m *logical.ConsensusSharePieceMessage) ([]byte, error) {
	gisHash := m.GISHash.Bytes()
	dummyId := m.DummyID.Serialize()
	dest := m.Dest.Serialize()
	share := sharePieceToPb(&m.Share)
	sign := signDataToPb(&m.SI)

	message := tas_pb.ConsensusSharePieceMessage{GISHash: gisHash, DummyID: dummyId, Dest: dest, SharePiece: share, Sign: sign}
	return proto.Marshal(&message)
}

func UnMarshalConsensusSharePieceMessage(b []byte) (*logical.ConsensusSharePieceMessage, error) {
	m := new(tas_pb.ConsensusSharePieceMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("UnMarshalConsensusSharePieceMessage error:%s\n", e.Error())
		return nil, e
	}

	gisHash := common.BytesToHash(m.GISHash)
	var dummyId, dest groupsig.ID
	e1 := dummyId.Deserialize(m.DummyID)
	if e1 != nil {
		logger.Errorf("groupsig.ID Deserialize error:%s\n", e1.Error())
		return nil, e1
	}

	e2 := dest.Deserialize(m.Dest)
	if e2 != nil {
		logger.Errorf("groupsig.ID Deserialize error:%s\n", e2.Error())
		return nil, e2
	}

	share := pbToSharePiece(m.SharePiece)
	sign := pbToSignData(m.Sign)

	message := logical.ConsensusSharePieceMessage{GISHash: gisHash, DummyID: dummyId, Dest: dest, Share: *share, SI: *sign}
	return &message, nil
}

func MarshalConsensusGroupInitedMessage(m *logical.ConsensusGroupInitedMessage) ([]byte, error) {
	gi := staticGroupInfoToPb(&m.GI)
	si := signDataToPb(&m.SI)
	message := tas_pb.ConsensusGroupInitedMessage{StaticGroupInfo: gi, Sign: si}
	return proto.Marshal(&message)
}

func UnMarshalConsensusGroupInitedMessage(b []byte) (*logical.ConsensusGroupInitedMessage, error) {
	m := new(tas_pb.ConsensusGroupInitedMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("UnMarshalConsensusGroupInitedMessage error:%s\n", e.Error())
		return nil, e
	}

	gi := pbToStaticGroup(m.StaticGroupInfo)
	si := pbToSignData(m.Sign)
	message := logical.ConsensusGroupInitedMessage{GI: *gi, SI: *si}
	return &message, nil
}

//--------------------------------------------组铸币--------------------------------------------------------------------
func MarshalConsensusCurrentMessagee(m *logical.ConsensusCurrentMessage) ([]byte, error) {
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

func UnMarshalConsensusCurrentMessage(b []byte) (*logical.ConsensusCurrentMessage, error) {
	m := new(tas_pb.ConsensusCurrentMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("UnMarshalConsensusCurrentMessage error:%s\n", e.Error())
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

func MarshalConsensusCastMessage(m *logical.ConsensusCastMessage) ([]byte, error) {
	bh := blockHeaderToPb(&m.BH)
	groupId := m.GroupID.Serialize()
	si := signDataToPb(&m.SI)

	message := tas_pb.ConsensusBlockMessageBase{Bh: bh, GroupID: groupId, Sign: si}
	return proto.Marshal(&message)
}

func UnMarshalConsensusCastMessage(b []byte) (*logical.ConsensusCastMessage, error) {
	m := new(tas_pb.ConsensusBlockMessageBase)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("UnMarshalConsensusCastMessage error:%s\n", e.Error())
		return nil, e
	}

	bh := pbToBlockHeader(m.Bh)
	var groupId groupsig.ID
	e1 := groupId.Deserialize(m.GroupID)
	if e1 != nil {
		logger.Errorf("groupsig.ID Deserialize error:%s\n", e1.Error())
		return nil, e1
	}
	si := pbToSignData(m.Sign)
	message := logical.ConsensusCastMessage{BH: *bh, GroupID: groupId, SI: *si}
	return &message, nil
}

func MarshalConsensusVerifyMessage(m *logical.ConsensusVerifyMessage) ([]byte, error) {
	bh := blockHeaderToPb(&m.BH)
	groupId := m.GroupID.Serialize()
	si := signDataToPb(&m.SI)

	message := tas_pb.ConsensusBlockMessageBase{Bh: bh, GroupID: groupId, Sign: si}
	return proto.Marshal(&message)
}

func UnMarshalConsensusVerifyMessage(b []byte) (*logical.ConsensusVerifyMessage, error) {
	m := new(tas_pb.ConsensusBlockMessageBase)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("UnMarshalConsensusVerifyMessage error:%s\n", e.Error())
		return nil, e
	}

	bh := pbToBlockHeader(m.Bh)
	var groupId groupsig.ID
	e1 := groupId.Deserialize(m.GroupID)
	if e1 != nil {
		logger.Errorf("groupsig.ID Deserialize error:%s\n", e1.Error())
		return nil, e1
	}
	si := pbToSignData(m.Sign)
	message := logical.ConsensusVerifyMessage{BH: *bh, GroupID: groupId, SI: *si}
	return &message, nil
}

//----------------------------------------------------------------------------------------------------------------------

func consensusGroupInitSummaryToPb(m *logical.ConsensusGroupInitSummary) *tas_pb.ConsensusGroupInitSummary {
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
		logger.Errorf("groupsig.ID Deserialize error:%s\n", e1.Error())
		return nil
	}

	var dummyID groupsig.ID
	e2 := parentId.Deserialize(m.ParentID)

	if e1 != nil {
		logger.Errorf("groupsig.ID Deserialize error:%s\n", e2.Error())
		return nil
	}
	message := logical.ConsensusGroupInitSummary{ParentID: parentId, Authority: *m.Authority,
		Name: name, DummyID: dummyID, BeginTime: beginTime}
	return &message
}

func signDataToPb(s *logical.SignData) *tas_pb.SignData {
	sign := tas_pb.SignData{DataHash: s.DataHash.Bytes(), DataSign: s.DataSign.Serialize(), SignMember: s.SignMember.Serialize()}
	return &sign
}

func pbToSignData(s *tas_pb.SignData) *logical.SignData {

	var sig groupsig.Signature
	e := sig.Deserialize(s.DataSign)
	if e != nil {
		logger.Errorf("groupsig.Signature Deserialize error:%s\n", e.Error())
		return nil
	}

	id := groupsig.ID{}
	e1 := id.Deserialize(s.SignMember)
	if e1 != nil {
		logger.Errorf("groupsig.ID Deserialize error:%s\n", e1.Error())
		return nil
	}
	sign := logical.SignData{DataHash: common.BytesToHash(s.DataHash), DataSign: sig, SignMember: id}
	return &sign
}

func sharePieceToPb(s *logical.SharePiece) *tas_pb.SharePiece {
	share := tas_pb.SharePiece{Seckey: s.Share.Serialize(), Pubkey: s.Pub.Serialize()}
	return &share
}

func pbToSharePiece(s *tas_pb.SharePiece) *logical.SharePiece {
	var share groupsig.Seckey
	var pub groupsig.Pubkey

	e1 := share.Deserialize(s.Seckey)
	if e1 != nil {
		logger.Errorf("groupsig.Seckey Deserialize error:%s\n", e1.Error())
		return nil
	}

	e2 := pub.Deserialize(s.Pubkey)
	if e2 != nil {
		logger.Errorf("groupsig.Pubkey Deserialize error:%s\n", e2.Error())
		return nil
	}

	sp := logical.SharePiece{Share: share, Pub: pub}
	return &sp
}

func staticGroupInfoToPb(s *logical.StaticGroupInfo) *tas_pb.StaticGroupInfo {
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

func pubKeyInfoToPb(p *logical.PubKeyInfo) *tas_pb.PubKeyInfo {
	id := p.ID.Serialize()
	pk := p.PK.Serialize()

	pkInfo := tas_pb.PubKeyInfo{ID: id, PublicKey: pk}
	return &pkInfo
}

func pbToPubKeyInfo(p *tas_pb.PubKeyInfo) *logical.PubKeyInfo {
	var id groupsig.ID
	var pk groupsig.Pubkey

	e1 := id.Deserialize(p.ID)
	if e1 != nil {
		logger.Errorf("groupsig.ID Deserialize error:%s\n", e1.Error())
		return nil
	}

	e2 := pk.Deserialize(p.PublicKey)
	if e2 != nil {
		logger.Errorf("groupsig.Pubkey Deserialize error:%s\n", e2.Error())
		return nil
	}

	pkInfo := logical.PubKeyInfo{ID: id, PK: pk}
	return &pkInfo
}

//----------------------------------------------块同步------------------------------------------------------------------
func MarshalBlockOrGroupRequestEntity(e *BlockOrGroupRequestEntity) ([]byte, error) {
	return nil, nil
}

func UnMarshalBlockOrGroupRequestEntity(b []byte) (*BlockOrGroupRequestEntity, error) {
	return nil, nil
}


func MarshalBlockEntity(e *BlockEntity)([]byte, error){
	return nil, nil
}

func UnMarshalBlockEntity(b []byte)(*BlockEntity,error){
	return nil, nil
}