package core

import (
	"testing"
	"common"
)

func TestBlockChain_AddBlock(t *testing.T) {

	Clear(DefaultBlockChainConfig())
	chain := InitBlockChain()
	//chain.Clear()

	blockHeader := chain.QueryTopBlock()
	if nil == blockHeader || 0 != blockHeader.Height {
		t.Fatalf("clear data fail")
	}

	txpool := chain.GetTransactionPool()
	if nil == txpool {
		t.Fatalf("fail to get txpool")
	}

	bytes := []byte("jdai")
	hash := Sha256(bytes)
	transaction1 := &Transaction{
		Gasprice: 12345,
		Hash:     common.BytesToHash(hash),
	}
	txpool.Add(transaction1)

	bytes2 := []byte("jdai2")
	hash2 := Sha256(bytes2)
	transaction2 := &Transaction{
		Gasprice: 123456,
		Hash:     common.BytesToHash(hash2),
	}
	txpool.Add(transaction2)

	block := chain.CastingBlock()
	if nil == block {
		t.Fatalf("fail to cast new block")
	}

	if 0 != chain.AddBlockOnChain(block) {
		t.Fatalf("fail to add block")
	}
}
