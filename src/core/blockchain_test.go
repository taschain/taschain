package core

import (
	"testing"
	"fmt"

)

func TestBlockChain_AddBlock(t *testing.T) {

	chain := InitBlockChain()
	chain.Clear()

	blockHeader := chain.QueryTopBlock()
	if nil != blockHeader {
		t.Fatalf("clear data fail")
	}

	txpool := chain.GetTransactionPool()
	if nil == txpool{
		t.Fatalf("fail to get txpool")
	}

	bytes := []byte("jdai")
	hash := chain.sha256(bytes)
	fmt.Printf("hash: %x\nlength: %d", hash, len(hash))
	//block := chain.CastingBlock()

}
