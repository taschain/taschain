package core

import (
	"taslog"
	"pb"
	"github.com/gogo/protobuf/proto"
	"network/p2p"
	"time"
	"common"
	"log"
)

var logger = taslog.GetLogger(taslog.P2PConfig)

type TransactionRequestMessage struct {
	TransactionHashes []common.Hash
	SourceId          string
	RequestTime       time.Time
}

type EntityTotalQNMessage struct {
	TotalQN  uint64
	SourceId string
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
	Blocks      []*Block
	BlockHashes []*ChainBlockHash
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
		log.Printf("[peer]Discard MarshalTransactionRequestMessage because of marshal error:%s!", e.Error())
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
		log.Printf("[peer]Discard MarshalTransactions because of marshal error:%s!", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.TRANSACTION_GOT_MSG, Body: body}
	p2p.Server.SendMessage(message, sourceId)
}

//收到交易 全网扩散
func BroadcastTransactions(txs []*Transaction) {

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[peer]Runtime error caught: %v", r)
		}
	}()

	body, e := marshalTransactions(txs)
	if e != nil {
		log.Printf("[peer]Discard MarshalTransactions because of marshal error:%s", e.Error())
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

//向某一节点请求Block
func RequestBlockByHeight(id string, localHeight uint64, currentHash common.Hash) {
	m := EntityRequestMessage{SourceHeight: localHeight, SourceCurrentHash: currentHash, SourceId: ""}
	body, e := MarshalEntityRequestMessage(&m)
	if e != nil {
		log.Printf("[peer]requestBlockByHeight marshal EntityRequestMessage error:%s", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.REQ_BLOCK_MSG, Body: body}
	p2p.Server.SendMessage(message, id)
}

type ChainBlockHashesReq struct {
	Height uint64 //起始高度
	Length uint64 //从起始高度开始的非空长度
}

type ChainBlockHash struct {
	Height uint64 //所在链高度
	Hash   common.Hash
}

//向目标结点索要 block hash
func RequestBlockChainHashes(targetNode string, cbhr ChainBlockHashesReq) {
	body, e := marshalChainBlockHashesReq(&cbhr)
	if e != nil {
		log.Printf("[peer]Discard RequestBlockChainHashes because of marshal error:%s!", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.BLOCK_CHAIN_HASHES_REQ, Body: body}
	p2p.Server.SendMessage(message, targetNode)
}

//向目标结点发送 block hash
func SendChainBlockHashes(targetNode string, cbh []*ChainBlockHash) {
	body, e := marshalChainBlockHashes(cbh)
	if e != nil {
		log.Printf("[peer]Discard sendChainBlockHashes because of marshal error:%s!", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.BLOCK_CHAIN_HASHES, Body: body}
	p2p.Server.SendMessage(message, targetNode)
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
		log.Printf("[peer]TransactionRequestMessage request time marshal error:%s\n", e.Error())
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
	if txs == nil {
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
	if block == nil {
		return nil, nil
	}
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
	txHashes := tas_pb.Hashes{Hashes: hashBytes}

	preTime, e1 := h.PreTime.MarshalBinary()
	if e1 != nil {
		log.Printf("[peer]BlockHeaderToPb marshal pre time error:%s\n", e1.Error())
		return nil
	}

	curTime, e2 := h.CurTime.MarshalBinary()
	if e2 != nil {
		log.Printf("[peer]BlockHeaderToPb marshal cur time error:%s", e2.Error())
		return nil
	}

	eTxs := h.EvictedTxs
	eBytes := make([][]byte, 0)

	if eTxs != nil {
		for _, etx := range eTxs {
			eBytes = append(eBytes, etx.Bytes())
		}
	}
	evictedTxs := tas_pb.Hashes{Hashes: eBytes}

	header := tas_pb.BlockHeader{Hash: h.Hash.Bytes(), Height: &h.Height, PreHash: h.PreHash.Bytes(), PreTime: preTime,
		QueueNumber: &h.QueueNumber, CurTime: curTime, Castor: h.Castor, GroupId: h.GroupId, Signature: h.Signature,
		Nonce: &h.Nonce, Transactions: &txHashes, TxTree: h.TxTree.Bytes(), ReceiptTree: h.ReceiptTree.Bytes(), StateTree: h.StateTree.Bytes(),
		ExtraData: h.ExtraData, EvictedTxs: &evictedTxs, TotalQN: &h.TotalQN}
	return &header
}

func BlockToPb(b *Block) *tas_pb.Block {
	if b == nil {
		log.Printf("[peer]Block is nil!")
		return nil
	}
	header := BlockHeaderToPb(b.Header)
	transactons := transactionsToPb(b.Transactions)
	block := tas_pb.Block{Header: header, Transactions: transactons}
	return &block
}

func marshalChainBlockHashesReq(req *ChainBlockHashesReq) ([]byte, error) {
	if req == nil {
		return nil, nil
	}
	cbhr := chainBlockHashesReqToPb(req)
	return proto.Marshal(cbhr)
}

func chainBlockHashesReqToPb(req *ChainBlockHashesReq) *tas_pb.ChainBlockHashesReq {
	if req == nil {
		return nil
	}
	cbhr := tas_pb.ChainBlockHashesReq{Height: &req.Height, Length: &req.Length}
	return &cbhr
}
func marshalChainBlockHashes(cbh []*ChainBlockHash) ([]byte, error) {
	if cbh == nil {
		return nil, nil
	}

	chainBlockHashes := make([]*tas_pb.ChainBlockHash, 0)
	for _, c := range cbh {
		chainBlockHashes = append(chainBlockHashes, ChainBlockHashToPb(c))
	}
	r := tas_pb.ChainBlockHashSlice{ChainBlockHashes: chainBlockHashes}
	return proto.Marshal(&r)
}

func ChainBlockHashToPb(cbh *ChainBlockHash) *tas_pb.ChainBlockHash {
	if cbh == nil {
		return nil
	}

	r := tas_pb.ChainBlockHash{Hash: cbh.Hash.Bytes(), Height: &cbh.Height}
	return &r
}

func MarshalEntityRequestMessage(e *EntityRequestMessage) ([]byte, error) {
	sourceHeight := e.SourceHeight
	currentHash := e.SourceCurrentHash.Bytes()
	sourceId := []byte(e.SourceId)

	m := tas_pb.EntityRequestMessage{SourceHeight: &sourceHeight, SourceCurrentHash: currentHash, SourceId: sourceId}
	return proto.Marshal(&m)
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
