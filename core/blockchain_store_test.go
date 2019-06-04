package core

import (
	"fmt"
	"github.com/taschain/taschain/common"
	"testing"
)

func TestFullBlockChain_HasBlock(t *testing.T) {
	initContext4Test()
	defer clear()
	hasBLock := BlockChainImpl.HasBlock(common.HexToHash("0x7f57774109cad543d9acfbcfa3630b30ca652d2310470341b78c62ee7463633b"))
	t.Log(hasBLock)
}

func TestFullBlockChain_QueryBlockFloor(t *testing.T) {
	initContext4Test()
	defer clear()
	chain := BlockChainImpl.(*FullBlockChain)

	fmt.Println("=====")
	bh := chain.queryBlockHeaderByHeight(0)
	fmt.Println(bh, bh.Hash.Hex())
	//top := gchain.latestBlock
	//t.Log(top.Height, top.Hash.String())
	//
	//for h := uint64(4460); h <= 4480; h++ {
	//	bh := gchain.queryBlockHeaderByHeightFloor(h)
	//	t.Log(bh.Height, bh.Hash.String())
	//}

	bh = chain.queryBlockHeaderByHeightFloor(0)
	fmt.Println(bh)
}
