package core

import (
	"testing"
	"common"

	c "vm/common"
	"fmt"
)

func TestBlockChain_AddBlock(t *testing.T) {

	Clear()
	initBlockChain()
	BlockChainImpl.transactionPool.Clear()
	//chain.Clear()

	// 查询创始块
	blockHeader := BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 0 != blockHeader.Height {
		t.Fatalf("clear data fail")
	}

	if BlockChainImpl.latestStateDB.GetBalance(c.BytesToAddress(genHash("1"))).Int64() != 100 {
		t.Fatalf("fail to init 1 balace to 100")
	}

	txpool := BlockChainImpl.GetTransactionPool()
	if nil == txpool {
		t.Fatalf("fail to get txpool")
	}

	// 交易1
	txpool.Add(genTestTx("jdai1", 12345, "1", "2", 0, 1))

	//交易2
	txpool.Add(genTestTx("jdai2", 123456, "2", "3", 0, 1))

	castor := new([]byte)
	groupid := new([]byte)

	// 铸块1
	block := BlockChainImpl.CastingBlock(1, 12, 0, *castor, *groupid)
	if nil == block {
		t.Fatalf("fail to cast new block")
	}

	// 上链
	if 0 != BlockChainImpl.AddBlockOnChain(block) {
		t.Fatalf("fail to add block")
	}

	//最新块是块1
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 1 != blockHeader.Height {
		t.Fatalf("add block1 failed")
	}

	if BlockChainImpl.latestStateDB.GetBalance(c.BytesToAddress(genHash("1"))).Int64() != 99 {
		t.Fatalf("fail to transfer 1 from 1  to 2")
	}

	// 池子中交易的数量为0
	if 0 != len(txpool.received) {
		t.Fatalf("fail to remove transactions after addBlock")
	}

	//交易3
	txpool.Add(genTestTx("jdai3", 1, "1", "2", 1, 10))

	// 铸块2
	block2 := BlockChainImpl.CastingBlock(2, 123, 0, *castor, *groupid)
	block2.Header.QueueNumber = 2
	if 0 != BlockChainImpl.AddBlockOnChain(block2) {
		t.Fatalf("fail to add empty block")
	}

	if BlockChainImpl.latestStateDB.GetBalance(c.BytesToAddress(genHash("1"))).Int64() != 89 {
		t.Fatalf("fail to transfer 10 from 1 to 2")
	}

	//最新块是块2
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 2 != blockHeader.Height || blockHeader.Hash != block2.Header.Hash || block.Header.Hash != block2.Header.PreHash {
		t.Fatalf("add block2 failed")
	}
	blockHeader = BlockChainImpl.QueryBlockByHash(block2.Header.Hash)
	if nil == blockHeader {
		t.Fatalf("fail to QueryBlockByHash, hash: %x ", block2.Header.Hash)
	}

	blockHeader = BlockChainImpl.QueryBlockByHeight(2)
	if nil == blockHeader {
		t.Fatalf("fail to QueryBlockByHeight, height: %d ", 2)
	}

	// 铸块3 空块
	block3 := BlockChainImpl.CastingBlock(3, 125, 0, *castor, *groupid)
	if 0 != BlockChainImpl.AddBlockOnChain(block3) {
		t.Fatalf("fail to add empty block")
	}
	//最新块是块3
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 3 != blockHeader.Height || blockHeader.Hash != block3.Header.Hash {
		t.Fatalf("add block3 failed")
	}

	// 铸块4 空块
	// 模拟分叉
	block4 := BlockChainImpl.CastingBlockAfter(block.Header, 2, 124, 0, *castor, *groupid)

	if 0 != BlockChainImpl.AddBlockOnChain(block4) {
		t.Fatalf("fail to add empty block")
	}
	//最新块是块4
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 2 != blockHeader.Height || blockHeader.Hash != block4.Header.Hash {
		t.Fatalf("add block4 failed")
	}
	blockHeader = BlockChainImpl.QueryBlockByHeight(3)
	if nil != blockHeader {
		t.Fatalf("failed to remove uncle blocks")
	}

	if BlockChainImpl.latestStateDB.GetBalance(c.BytesToAddress(genHash("1"))).Int64() != 99 {
		t.Fatalf("fail to switch to main chain")
	}

	BlockChainImpl.Close()

}

func TestBlockChain_CastingBlock(t *testing.T) {
	Clear()
	err := initBlockChain()
	if nil != err {
		panic(err)
	}
	BlockChainImpl.transactionPool.Clear()

	castor := []byte{1, 2}
	group := []byte{3, 4}
	block1 := BlockChainImpl.CastingBlock(1, 1, 1, castor, group)
	if nil == block1 {
		t.Fatalf("fail to cast block1")
	}

	BlockChainImpl.Close()
}

func TestBlockChain_GetBlockMessage(t *testing.T) {
	Clear()
	initBlockChain()
	BlockChainImpl.transactionPool.Clear()

	castor := new([]byte)
	groupid := new([]byte)
	block1 := BlockChainImpl.CastingBlock(1, 125, 0, *castor, *groupid)
	if 0 != BlockChainImpl.AddBlockOnChain(block1) {
		t.Fatalf("fail to add empty block")
	}

	block2 := BlockChainImpl.CastingBlock(2, 1256, 0, *castor, *groupid)
	if 0 != BlockChainImpl.AddBlockOnChain(block2) {
		t.Fatalf("fail to add empty block")
	}

	block3 := BlockChainImpl.CastingBlock(3, 1257, 0, *castor, *groupid)
	if 0 != BlockChainImpl.AddBlockOnChain(block3) {
		t.Fatalf("fail to add empty block")
	}

	if 3 != BlockChainImpl.Height() {
		t.Fatalf("fail to add 3 blocks")
	}

	header1 := BlockChainImpl.queryBlockHeaderByHeight(uint64(1), true)
	header2 := BlockChainImpl.queryBlockHeaderByHeight(uint64(2), false)
	header3 := BlockChainImpl.queryBlockHeaderByHeight(uint64(3), true)

	b1 := BlockChainImpl.queryBlockByHash(header1.Hash)
	b2 := BlockChainImpl.queryBlockByHash(header2.Hash)
	b3 := BlockChainImpl.queryBlockByHash(header3.Hash)

	fmt.Printf("1: %d\n", b1.Header.Nonce)
	fmt.Printf("2: %d\n", b2.Header.Nonce)
	fmt.Printf("3: %d\n", b3.Header.Nonce)

	message := BlockChainImpl.GetBlockMessage(1, common.Hash{})
	if nil == message || nil == message.Blocks {
		t.Fatalf("fail to get BlockMessage from 1")
	}

	blocks := message.Blocks
	if nil == blocks[0] || nil == blocks[1] || 1256 != blocks[0].Header.Nonce || 1257 != blocks[1].Header.Nonce {
		t.Fatalf("fail to get BlockMessage from 1")
	}

}

func TestBlockChain_GetTopBlocks(t *testing.T) {
	Clear()
	initBlockChain()
	BlockChainImpl.transactionPool.Clear()

	castor := new([]byte)
	groupid := new([]byte)

	var i uint64
	for i = 1; i < 2000; i++ {
		block := BlockChainImpl.CastingBlock(i, i, 0, *castor, *groupid)
		if 0 != BlockChainImpl.AddBlockOnChain(block) {
			t.Fatalf("fail to add empty block")
		}
	}

	if 1000 != BlockChainImpl.topBlocks.Len() {
		t.Fatalf("error for size:1000")
	}

	for i = BlockChainImpl.Height() - 999; i < 2000; i++ {
		lowest, ok := BlockChainImpl.topBlocks.Get(i)
		if !ok || nil == lowest {
			t.Fatalf("fail to get lowest block,%d", i)
		}

		lowestLDB := BlockChainImpl.queryBlockHeaderByHeight(i, false)
		if nil == lowestLDB {
			t.Fatalf("fail to get lowest block from ldb,%d", i)
		}

		bh := lowest.(*BlockHeader)
		if bh.Height != lowestLDB.Height {
			t.Fatalf("fail to check block from cache to ldb,%d", i)
		}
	}

	top := BlockChainImpl.GetTopBlocks()
	if top == nil || len(top) != 1000 {
		t.Fatalf("fail to get top blocks")
	}

	for i = 0; i < 1000; i++ {
		if top[i] == nil {

			t.Fatalf("fail to get top blocks, missing: %d", i)

		}

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
