package p2p

import (
	"pb"
	"core"
	"github.com/golang/protobuf/proto"
	"common"
	"taslog"
	"time"
)

func MarshalTransaction(t *core.Transaction) ([]byte, error) {
	transaction := transactionToPb(t)
	return proto.Marshal(transaction)
}

func UnMarshalTransaction(b []byte) (*core.Transaction, error) {
	t := new(tas_pb.Transaction)
	error := proto.Unmarshal(b, t)
	if error != nil {
		taslog.P2pLogger.Errorf("Unmarshal transaction error:%s\n", error.Error())
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
		taslog.P2pLogger.Errorf("Unmarshal transactions error:%s\n", error.Error())
		return nil, error
	}

	result := pbToTransactions(ts.Transactions)
	return result, nil
}

func transactionToPb(t *core.Transaction) *tas_pb.Transaction {
	transaction := tas_pb.Transaction{Id: t.Id.Bytes(), Data: t.Data, Value: &t.Value, Nonce: &t.Nonce, Source: t.Source.Bytes(),
		Target: t.Target.Bytes(), GasLimit: &t.GasLimit, GasPrice: &t.GasPrice, Hash: t.Hash.Bytes(), ExtraData: t.ExtraData}
	return &transaction
}

func pbToTransaction(t *tas_pb.Transaction) *core.Transaction {
	source := common.BytesToAddress(t.Source)
	target := common.BytesToAddress(t.Target)
	transaction := core.Transaction{Id: common.BytesToAddress(t.Id), Data: t.Data, Value: *t.Value, Nonce: *t.Nonce, Source: &source,
		Target: &target, GasLimit: *t.GasLimit, GasPrice: *t.GasPrice, Hash: common.BytesToHash(t.Hash), ExtraData: t.ExtraData}
	return &transaction
}

func transactionsToPb(txs []*core.Transaction) []*tas_pb.Transaction {
	transactions := make([]*tas_pb.Transaction, len(txs))
	for _, t := range txs {
		transaction := transactionToPb(t)
		transactions = append(transactions, transaction)
	}
	return transactions
}

func pbToTransactions(txs []*tas_pb.Transaction) []*core.Transaction {
	result := make([]*core.Transaction, len(txs))
	for _, t := range txs {
		transaction := pbToTransaction(t)
		result = append(result, transaction)
	}
	return result
}

func MarshalBlock(b *core.Block) ([]byte, error) {
	block := blockToPb(b)
	return proto.Marshal(block)
}

func UnMarshalBlock(bytes []byte) (*core.Block, error) {
	b := new(tas_pb.Block)
	error := proto.Unmarshal(bytes, b)
	if error != nil {
		taslog.P2pLogger.Errorf("Unmarshal Block error:%s\n", error.Error())
		return nil, error
	}
	block := pbToBlock(b)
	return block, nil
}

func MarshalBlocks(bs []*core.Block) ([]byte, error) {
	blocks := make([]*tas_pb.Block, len(bs))
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
		taslog.P2pLogger.Errorf("Unmarshal Blocks error:%s\n", error.Error())
		return nil, error
	}
	blocks := blockSlice.Blocks
	result := make([]*core.Block, len(blocks))

	for _, b := range blocks {
		block := pbToBlock(b)
		result = append(result, block)
	}
	return result, nil
}

func blockHeaderToPb(h *core.BlockHeader) *tas_pb.BlockHeader {
	hashes := h.Transactions
	hashBytes := make([][]byte, len(hashes))
	for _, hash := range hashes {
		hashBytes = append(hashBytes, hash.Bytes())
	}
	preTime, e1 := h.PreTime.MarshalBinary()
	if e1 != nil {
		taslog.P2pLogger.Errorf("BlockHeaderToPb marshal pre time error:%s\n", e1.Error())
		return nil
	}

	curTime, e2 := h.CurTime.MarshalBinary()
	if e2 != nil {
		taslog.P2pLogger.Errorf("BlockHeaderToPb marshal cur time error:%s\n", e2.Error())
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
	hashes := make([]common.Hash, len(hashBytes))
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

func MarshalBlockMap(blockMap map[uint64]core.Block) ([]byte, error) {
	m := make(map[uint64]*tas_pb.Block, len(blockMap))
	for id, block := range blockMap {
		m[id] = blockToPb(&block)
	}
	bMap := tas_pb.BlockMap{Blocks: m}
	return proto.Marshal(&bMap)
}

func UnMarshalBlockMap(b []byte) (map[uint64]*core.Block, error) {
	blockMap := new(tas_pb.BlockMap)
	e := proto.Unmarshal(b, blockMap)
	if e!=nil{
		taslog.P2pLogger.Errorf("UnMarshalBlockMap error:%s\n", e.Error())
		return nil,e
	}
	m := make(map[uint64]*core.Block,len(blockMap.Blocks))
	for id,block := range blockMap.Blocks{
		m[id] = pbToBlock(block)
	}
	return m,nil
}
