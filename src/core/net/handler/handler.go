package handler

import (
	"time"
	"network/p2p"
	"pb"
	"github.com/gogo/protobuf/proto"
	"common"
	"core"
	"taslog"
	"core/net/sync"
	"utility"
	"fmt"
)

var logger = taslog.GetLogger(taslog.P2PConfig)

const MAX_TRANSACTION_REQUEST_INTERVAL = 20 * time.Second

type ChainHandler struct{}

func (c *ChainHandler) HandlerMessage(code uint32, body []byte, sourceId string) error {
	switch code {
	case p2p.REQ_TRANSACTION_MSG:
		m, e := unMarshalTransactionRequestMessage(body)
		if e != nil {
			logger.Error("Discard TransactionRequestMessage because of unmarshal error!\n")
			return nil
		}
		OnTransactionRequest(m)
	case p2p.TRANSACTION_GOT_MSG, p2p.TRANSACTION_MSG:
		m, e := unMarshalTransactions(body)
		if e != nil {
			logger.Error("Discard TRANSACTION_MSG because of unmarshal error!\n")
			return nil
		}
		return OnMessageTransaction(m)
	case p2p.NEW_BLOCK_MSG:
		//todo

	case p2p.REQ_BLOCK_CHAIN_HEIGHT_MSG:
		sync.BlockSyncer.HeightRequestCh <- sourceId
	case p2p.BLOCK_CHAIN_HEIGHT_MSG:
		height := utility.ByteToUInt64(body)
		s := core.EntityHeightMessage{Height: height, SourceId: sourceId}
		sync.BlockSyncer.HeightCh <- s
	case p2p.REQ_BLOCK_MSG:
		m, e := unMarshalEntityRequestMessage(body)
		if e != nil {
			logger.Error("Discard REQ_BLOCK_MSG because of unmarshal error!\n")
			return e
		}
		s := core.EntityRequestMessage{SourceHeight: m.SourceHeight, SourceCurrentHash: m.SourceCurrentHash, SourceId: sourceId}
		sync.BlockSyncer.BlockRequestCh <- s
	case p2p.BLOCK_MSG:
		m, e := unMarshalBlockMessage(body)
		if e != nil {
			logger.Error("Discard BLOCK_MSG because of unmarshal error!\n")
			return e
		}
		s := core.BlockArrivedMessage{BlockEntity: *m, SourceId: sourceId}
		sync.BlockSyncer.BlockArrivedCh <- s

	case p2p.REQ_GROUP_CHAIN_HEIGHT_MSG:
		sync.GroupSyncer.HeightRequestCh <- sourceId
	case p2p.GROUP_CHAIN_HEIGHT_MSG:
		height := utility.ByteToUInt64(body)
		s := core.EntityHeightMessage{Height: height, SourceId: sourceId}
		sync.GroupSyncer.HeightCh <- s
	case p2p.REQ_GROUP_MSG:
		m, e := unMarshalEntityRequestMessage(body)
		if e != nil {
			logger.Error("Discard REQ_BLOCK_MSG because of unmarshal error!\n")
			return e
		}
		s := core.EntityRequestMessage{SourceHeight: m.SourceHeight, SourceCurrentHash: m.SourceCurrentHash, SourceId: sourceId}
		sync.GroupSyncer.GroupRequestCh <- s
	case p2p.GROUP_MSG:
		m, e := unMarshalGroupMessage(body)
		if e != nil {
			logger.Error("Discard BLOCK_MSG because of unmarshal error!\n")
			return e
		}
		s := core.GroupArrivedMessage{GroupEntity: *m, SourceId: sourceId}
		sync.GroupSyncer.GroupArrivedCh <- s
	}
	return nil
}

//-----------------------------------------------铸币-------------------------------------------------------------------

//接收索要交易请求 查询自身是否有该交易 有的话返回, 没有的话自己广播该请求
func OnTransactionRequest(m *core.TransactionRequestMessage) error {
	requestTime := m.RequestTime
	now := time.Now()
	interval := now.Sub(requestTime)
	if interval > MAX_TRANSACTION_REQUEST_INTERVAL {
		return nil
	}

	//本地查询transaction
	if nil == core.BlockChainImpl {
		return nil
	}
	transactions, need, e := core.BlockChainImpl.GetTransactionPool().GetTransactions(m.TransactionHashes)
	if e == core.ErrNil {
		logger.Error("Local do not have transaction,broadcast this message!:%s", e.Error())
		m.TransactionHashes = need
		core.BroadcastTransactionRequest(*m)
	}

	if nil != transactions && 0 != len(transactions) {
		core.SendTransactions(transactions, m.SourceId)
	}

	return nil
}

func mockTxs() []*core.Transaction {
	//source byte: 138,170,12,235,193,42,59,204,152,26,146,154,213,207,129,10,9,14,17,174
	//target byte: 93,174,34,35,176,3,97,163,150,23,122,156,180,16,255,97,242,0,21,173
	//hash : 112,155,85,189,61,160,245,168,56,18,91,208,238,32,197,191,221,124,171,161,115,145,45,66,129,202,232,22,183,154,32,27
	t1 := genTestTx("tx1", 123, "111", "abc", 0, 1)
	t2 := genTestTx("tx1", 456, "222", "ddd", 0, 1)
	s := []*core.Transaction{t1, t2}
	return s
}

func genTestTx(hash string, price uint64, source string, target string, nonce uint64, value uint64) *core.Transaction {

	sourcebyte := common.BytesToAddress(core.Sha256([]byte(source)))
	targetbyte := common.BytesToAddress(core.Sha256([]byte(target)))

	//byte: 84,104,105,115,32,105,115,32,97,32,116,114,97,110,115,97,99,116,105,111,110
	data := []byte("This is a transaction")
	return &core.Transaction{
		Data:     data,
		Value:    value,
		Nonce:    nonce,
		Source:   &sourcebyte,
		Target:   &targetbyte,
		GasPrice: price,
		GasLimit: 3,
		Hash:     common.BytesToHash(core.Sha256([]byte(hash))),
	}
}

//验证节点接收交易 或者接收来自客户端广播的交易
func OnMessageTransaction(txs []*core.Transaction) error {
	//验证节点接收交易 加入交易池
	if nil == txs {
		logger.Error("received nil txs")
	} else {
		logger.Errorf("received: %d", len(txs))
	}

	if nil == core.BlockChainImpl {
		return nil
	}
	e := core.BlockChainImpl.GetTransactionPool().AddTransactions(txs)
	if e != nil {
		logger.Errorf("OnMessageTransaction notify block error:%s \n", e.Error())
		return e
	}
	return nil
}

//全网其他节点 接收block 进行验证
func OnMessageNewBlock(b *core.Block) error {
	//接收到新的块 本地上链
	if nil == core.BlockChainImpl {
		return nil
	}
	if core.BlockChainImpl.AddBlockOnChain(b) == -1 {
		logger.Errorf("Add new block to chain error \n")
		return fmt.Errorf("fail to add block")
	}
	return nil
}

//----------------------------------------------Transaction-------------------------------------------------------------

func unMarshalTransaction(b []byte) (*core.Transaction, error) {
	t := new(tas_pb.Transaction)
	error := proto.Unmarshal(b, t)
	if error != nil {
		logger.Errorf("Unmarshal transaction error:%s\n", error.Error())
		return &core.Transaction{}, error
	}
	transaction := pbToTransaction(t)
	return transaction, nil
}

func unMarshalTransactions(b []byte) ([]*core.Transaction, error) {
	ts := new(tas_pb.TransactionSlice)
	error := proto.Unmarshal(b, ts)
	if error != nil {
		logger.Errorf("Unmarshal transactions error:%s\n", error.Error())
		return nil, error
	}

	result := pbToTransactions(ts.Transactions)
	return result, nil
}

func unMarshalTransactionRequestMessage(b []byte) (*core.TransactionRequestMessage, error) {
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
	message := core.TransactionRequestMessage{TransactionHashes: txHashes, SourceId: sourceId, RequestTime: requestTime}
	return &message, nil
}

func pbToTransaction(t *tas_pb.Transaction) *core.Transaction {
	source := common.BytesToAddress(t.Source)
	target := common.BytesToAddress(t.Target)
	transaction := core.Transaction{Data: t.Data, Value: *t.Value, Nonce: *t.Nonce, Source: &source,
		Target: &target, GasLimit: *t.GasLimit, GasPrice: *t.GasPrice, Hash: common.BytesToHash(t.Hash),
		ExtraData: t.ExtraData, ExtraDataType: *t.ExtraDataType}
	return &transaction
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
//func unMarshalBlock(bytes []byte) (*core.Block, error) {
//	b := new(tas_pb.Block)
//	error := proto.Unmarshal(bytes, b)
//	if error != nil {
//		logger.Errorf("Unmarshal Block error:%s\n", error.Error())
//		return nil, error
//	}
//	block := pbToBlock(b)
//	return block, nil
//}
//
//func unMarshalBlocks(b []byte) ([]*core.Block, error) {
//	blockSlice := new(tas_pb.BlockSlice)
//	error := proto.Unmarshal(b, blockSlice)
//	if error != nil {
//		logger.Errorf("Unmarshal Blocks error:%s\n", error.Error())
//		return nil, error
//	}
//	blocks := blockSlice.Blocks
//	result := make([]*core.Block, 0)
//
//	for _, b := range blocks {
//		block := pbToBlock(b)
//		result = append(result, block)
//	}
//	return result, nil
//}

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

func pbToBlock(b *tas_pb.Block) *core.Block {
	h := pbToBlockHeader(b.Header)
	txs := pbToTransactions(b.Transactions)
	block := core.Block{Header: h, Transactions: txs}
	return &block
}

//--------------------------------------------------Group---------------------------------------------------------------
//func unMarshalMember(b []byte) (*core.Member, error) {
//	member := new(tas_pb.Member)
//	e := proto.Unmarshal(b, member)
//	if e != nil {
//		logger.Errorf("UnMarshalMember error:%s\n", e.Error())
//		return nil, e
//	}
//	m := pbToMember(member)
//	return m, nil
//}

func pbToMember(m *tas_pb.Member) *core.Member {
	member := core.Member{Id: m.Id, PubKey: m.PubKey}
	return &member
}

//
//func unMarshalGroup(b []byte) (*core.Group, error) {
//	group := new(tas_pb.Group)
//	e := proto.Unmarshal(b, group)
//	if e != nil {
//		logger.Errorf("UnMarshalGroup error:%s\n", e.Error())
//		return nil, e
//	}
//	g := pbToGroup(group)
//	return g, nil
//}

func pbToGroup(g *tas_pb.Group) *core.Group {
	members := make([]core.Member, 0)
	for _, m := range g.Members {
		member := pbToMember(m)
		members = append(members, *member)
	}
	group := core.Group{Id: g.Id, Members: members, PubKey: g.PubKey, Parent: g.PubKey, Dummy: g.Dummy, Signature: g.Signature}
	return &group
}

//----------------------------------------------块同步------------------------------------------------------------------

func unMarshalEntityRequestMessage(b []byte) (*core.EntityRequestMessage, error) {
	m := new(tas_pb.EntityRequestMessage)

	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("Unmarshal EntityRequestMessage error:%s\n", e.Error())
		return nil, e
	}

	sourceHeight := m.SourceHeight
	sourceCurrentHash := common.BytesToHash(m.SourceCurrentHash)
	sourceId := string(m.SourceId)
	message := core.EntityRequestMessage{SourceHeight: *sourceHeight, SourceCurrentHash: sourceCurrentHash, SourceId: sourceId}
	return &message, nil
}

func unMarshalBlockMessage(b []byte) (*core.BlockMessage, error) {
	message := new(tas_pb.BlockMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("Unmarshal BlockMessage error:%s\n", e.Error())
		return nil, e
	}

	blocks := make([]*core.Block, 0)
	if message.Blocks.Blocks != nil {
		for _, b := range message.Blocks.Blocks {
			blocks = append(blocks, pbToBlock(b))
		}
	}

	height := *message.Height

	hashes := make([]common.Hash, 0)
	if message.Hashes.Hashes != nil {
		for _, h := range message.Hashes.Hashes {
			hash := common.BytesToHash(h)
			hashes = append(hashes, hash)
		}
	}

	ratios := message.Ratios.Ratios

	m := core.BlockMessage{Blocks: blocks, Height: height, BlockHashes: hashes, BlockRatios: ratios}
	return &m, nil
}

//----------------------------------------------组同步------------------------------------------------------------------
func unMarshalGroupMessage(b []byte) (*core.GroupMessage, error) {
	message := new(tas_pb.GroupMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("Unmarshal GroupMessage error:%s\n", e.Error())
		return nil, e
	}

	groups := make([]*core.Group, 0)
	if message.Groups.Groups != nil {
		for _, g := range message.Groups.Groups {
			groups = append(groups, pbToGroup(g))
		}
	}

	height := *message.Height

	hashes := make([]common.Hash, 0)
	if message.Hashes.Hashes != nil {
		for _, h := range message.Hashes.Hashes {
			hash := common.BytesToHash(h)
			hashes = append(hashes, hash)
		}
	}

	ratios := message.Ratios.Ratios

	m := core.GroupMessage{Groups: groups, Height: height, GroupHashes: hashes, GroupRatios: ratios}
	return &m, nil
}
