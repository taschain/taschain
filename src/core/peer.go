package core

import (
	"taslog"
	"pb"
	"github.com/gogo/protobuf/proto"
	"network/p2p"
	"time"
	"common"
)

var logger = taslog.GetLogger(taslog.P2PConfig)

type TransactionRequestMessage struct {
	TransactionHashes []common.Hash

	SourceId string

	RequestTime time.Time
}

type EntityHeightMessage struct {
	Height   uint64
	SourceId string
}

type EntityRequestMessage struct {
	SourceHeight      uint64
	SourceCurrentHash common.Hash
	SourceId          string
}

type BlockMessage struct {
	Blocks []*Block

	Height      uint64 //起始高度，如果返回blocks，那就是BLOCKS的起始高度，如果返回blockHashes那就是HASH的起始高度
	BlockHashes []common.Hash
	BlockRatios []float32
}

type BlockArrivedMessage struct {
	BlockEntity BlockMessage
	SourceId    string
}

type GroupMessage struct {
	Groups []*Group
	Height uint64      //起始高度
	Hash   common.Hash //起始HASH
}

type GroupArrivedMessage struct {
	GroupEntity GroupMessage
	SourceId    string
}

//验证节点 交易集缺失，索要、特定交易 全网广播
func BroadcastTransactionRequest(m TransactionRequestMessage) {
	if m.SourceId == "" {
		m.SourceId = p2p.Server.SelfNetInfo.Id
	}

	body, e := marshalTransactionRequestMessage(&m)
	if e != nil {
		logger.Errorf("Discard MarshalTransactionRequestMessage because of marshal error:%s!", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.REQ_TRANSACTION_MSG, Body: body}

	conns := p2p.Server.Host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			p2p.Server.SendMessage(message, p2p.ConvertToID(id))
		}
	}
}

//本地查询到交易，返回请求方
func SendTransactions(txs []*Transaction, sourceId string) {
	body, e := marshalTransactions(txs)
	if e != nil {
		logger.Errorf("Discard MarshalTransactions because of marshal error:%s!", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.TRANSACTION_GOT_MSG, Body: body}
	p2p.Server.SendMessage(message, sourceId)
}

//收到交易 全网扩散
func BroadcastTransactions(txs []*Transaction) {

	defer func() {
		if r := recover(); r != nil {
			logger.Error("Runtime error caught: %v", r)
		}
	}()

	body, e := marshalTransactions(txs)
	if e != nil {
		logger.Errorf("Discard MarshalTransactions because of marshal error:%s", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.TRANSACTION_MSG, Body: body}

	conns := p2p.Server.Host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			p2p.Server.SendMessage(message, p2p.ConvertToID(id))
		}
	}
}

//--------------------------------------------------Transaction---------------------------------------------------------------
//func marshalTransaction(t *Transaction) ([]byte, error) {
//	transaction := transactionToPb(t)
//	return proto.Marshal(transaction)
//}

func marshalTransactions(txs []*Transaction) ([]byte, error) {
	transactions := transactionsToPb(txs)
	transactionSlice := tas_pb.TransactionSlice{Transactions: transactions}
	return proto.Marshal(&transactionSlice)
}

func marshalTransactionRequestMessage(m *TransactionRequestMessage) ([]byte, error) {
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

func transactionToPb(t *Transaction) *tas_pb.Transaction {
	var target []byte
	if t.Target != nil {
		target = t.Target.Bytes()
	}

	transaction := tas_pb.Transaction{Data: t.Data, Value: &t.Value, Nonce: &t.Nonce, Source: t.Source.Bytes(),
		Target: target, GasLimit: &t.GasLimit, GasPrice: &t.GasPrice, Hash: t.Hash.Bytes(),
		ExtraData: t.ExtraData, ExtraDataType: &t.ExtraDataType}
	return &transaction
}

func transactionsToPb(txs []*Transaction) []*tas_pb.Transaction {
	if txs == nil{
		return nil
	}
	transactions := make([]*tas_pb.Transaction, 0)
	for _, t := range txs {
		transaction := transactionToPb(t)
		transactions = append(transactions, transaction)
	}
	return transactions
}

//--------------------------------------------------Block---------------------------------------------------------------
func MarshalBlock(b *Block) ([]byte, error) {
	block := BlockToPb(b)
	return proto.Marshal(block)
}

func MarshalBlocks(bs []*Block) ([]byte, error) {
	blocks := make([]*tas_pb.Block, 0)
	for _, b := range bs {
		block := BlockToPb(b)
		blocks = append(blocks, block)
	}
	blockSlice := tas_pb.BlockSlice{Blocks: blocks}
	return proto.Marshal(&blockSlice)
}

func BlockHeaderToPb(h *BlockHeader) *tas_pb.BlockHeader {
	hashes := h.Transactions
	hashBytes := make([][]byte, 0)

	if hashes != nil {
		for _, hash := range hashes {
			hashBytes = append(hashBytes, hash.Bytes())
		}
	}
	txHashes := tas_pb.Hashes{Hashes:hashBytes}

	preTime, e1 := h.PreTime.MarshalBinary()
	if e1 != nil {
		logger.Errorf("BlockHeaderToPb marshal pre time error:%s\n", e1.Error())
		return nil
	}

	curTime, e2 := h.CurTime.MarshalBinary()
	if e2 != nil {
		logger.Errorf("BlockHeaderToPb marshal cur time error:%s", e2.Error())
		return nil
	}

	header := tas_pb.BlockHeader{Hash: h.Hash.Bytes(), Height: &h.Height, PreHash: h.PreHash.Bytes(), PreTime: preTime,
		QueueNumber: &h.QueueNumber, CurTime: curTime, Castor: h.Castor, GroupId: h.GroupId, Signature: h.Signature.Bytes(),
		Nonce: &h.Nonce, Transactions: &txHashes, TxTree: h.TxTree.Bytes(), ReceiptTree: h.ReceiptTree.Bytes(), StateTree: h.StateTree.Bytes(),
		ExtraData: h.ExtraData}
	return &header
}

func BlockToPb(b *Block) *tas_pb.Block {
	header := BlockHeaderToPb(b.Header)
	transactons := transactionsToPb(b.Transactions)
	block := tas_pb.Block{Header: header, Transactions: transactons}
	return &block
}

//--------------------------------------------------Group---------------------------------------------------------------

//func marshalMember(m *Member) ([]byte, error) {
//	member := memberToPb(m)
//	return proto.Marshal(member)
//}

func memberToPb(m *Member) *tas_pb.Member {
	member := tas_pb.Member{Id: m.Id, PubKey: m.PubKey}
	return &member
}

//func marshalGroup(g *Group) ([]byte, error) {
//	group := groupToPb(g)
//	return proto.Marshal(group)
//}

func GroupToPb(g *Group) *tas_pb.Group {
	members := make([]*tas_pb.Member, 0)
	for _, m := range g.Members {
		member := memberToPb(&m)
		members = append(members, member)
	}
	group := tas_pb.Group{Id: g.Id, Members: members, PubKey: g.PubKey, Parent: g.Parent, Dummy: g.Dummy, Signature: g.Signature}
	return &group
}
