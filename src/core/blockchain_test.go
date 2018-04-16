package core

import (
	"testing"
	"common"

	c "vm/common"
)

func TestBlockChain_AddBlock(t *testing.T) {

	Clear(DefaultBlockChainConfig())
	chain := InitBlockChain()
	//chain.Clear()

	// 查询创始块
	blockHeader := chain.QueryTopBlock()
	if nil == blockHeader || 0 != blockHeader.Height {
		t.Fatalf("clear data fail")
	}

	if chain.latestStateDB.GetBalance(c.BytesToAddress(genHash("1"))).Int64() != 100 {
		t.Fatalf("fail to init 1 balace to 100")
	}

	txpool := chain.GetTransactionPool()
	if nil == txpool {
		t.Fatalf("fail to get txpool")
	}

	// 交易1
	txpool.Add(genTestTx("jdai1", 12345, "1", "2", 0, 1))

	//交易2
	txpool.Add(genTestTx("jdai2", 123456, "2", "3", 0, 1))

	// 铸块1
	block := chain.CastingBlock()
	if nil == block {
		t.Fatalf("fail to cast new block")
	}

	// 上链
	if 0 != chain.AddBlockOnChain(block) {
		t.Fatalf("fail to add block")
	}

	//最新块是块1
	blockHeader = chain.QueryTopBlock()
	if nil == blockHeader || 1 != blockHeader.Height {
		t.Fatalf("add block1 failed")
	}

	if chain.latestStateDB.GetBalance(c.BytesToAddress(genHash("1"))).Int64() != 99 {
		t.Fatalf("fail to transfer 1 from 1  to 2")
	}

	// 池子中交易的数量为0
	if 0 != len(txpool.received) {
		t.Fatalf("fail to remove transactions after addBlock")
	}

	//交易3
	txpool.Add(genTestTx("jdai3", 1, "1", "2", 1, 10))

	// 铸块2
	block2 := chain.CastingBlock()
	block2.Header.QueueNumber = 2
	if 0 != chain.AddBlockOnChain(block2) {
		t.Fatalf("fail to add empty block")
	}

	if chain.latestStateDB.GetBalance(c.BytesToAddress(genHash("1"))).Int64() != 89 {
		t.Fatalf("fail to transfer 10 from 1 to 2")
	}

	//最新块是块2
	blockHeader = chain.QueryTopBlock()
	if nil == blockHeader || 2 != blockHeader.Height || blockHeader.Hash != block2.Header.Hash || block.Header.Hash != block2.Header.PreHash {
		t.Fatalf("add block2 failed")
	}
	blockHeader = chain.QueryBlockByHash(block2.Header.Hash)
	if nil == blockHeader {
		t.Fatalf("fail to QueryBlockByHash, hash: %x ", block2.Header.Hash)
	}

	blockHeader = chain.QueryBlockByHeight(2)
	if nil == blockHeader {
		t.Fatalf("fail to QueryBlockByHeight, height: %d ", 2)
	}

	// 铸块3 空块
	block3 := chain.CastingBlock()
	if 0 != chain.AddBlockOnChain(block3) {
		t.Fatalf("fail to add empty block")
	}
	//最新块是块3
	blockHeader = chain.QueryTopBlock()
	if nil == blockHeader || 3 != blockHeader.Height || blockHeader.Hash != block3.Header.Hash {
		t.Fatalf("add block3 failed")
	}

	// 铸块4 空块
	// 模拟分叉
	block4 := chain.CastingBlockAfter(block.Header)

	if 0 != chain.AddBlockOnChain(block4) {
		t.Fatalf("fail to add empty block")
	}
	//最新块是块4
	blockHeader = chain.QueryTopBlock()
	if nil == blockHeader || 2 != blockHeader.Height || blockHeader.Hash != block4.Header.Hash {
		t.Fatalf("add block4 failed")
	}
	blockHeader = chain.QueryBlockByHeight(3)
	if nil != blockHeader {
		t.Fatalf("failed to remove uncle blocks")
	}

	if chain.latestStateDB.GetBalance(c.BytesToAddress(genHash("1"))).Int64() != 99 {
		t.Fatalf("fail to switch to main chain")
	}

}

func genTestTx(hash string, price uint64, source string, target string, nonce uint64, value uint64) *Transaction {

	sourcebyte := common.BytesToAddress(genHash(source))
	targetbyte := common.BytesToAddress(genHash(target))

	return &Transaction{
		GasPrice: price,
		Hash:     common.BytesToHash(genHash(hash)),
		Source:   &sourcebyte,
		Target:   &targetbyte,
		Nonce:    nonce,
		Value:    value,
	}
}

func genHash(hash string) []byte {
	bytes3 := []byte(hash)
	return Sha256(bytes3)
}
