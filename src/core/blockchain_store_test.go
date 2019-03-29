package core

import (
	"testing"
	"common"
	"middleware/types"
	"middleware"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2019/3/19 下午2:46
**  Description: 
*/

func TestFullBlockChain_HasBlock(t *testing.T) {
	common.InitConf("/Users/pxf/workspace/tas_develop/tas/deploy/daily/tas1.ini")
	types.InitMiddleware()
	middleware.InitMiddleware()
	initBlockChain(nil)

	hasBLock := BlockChainImpl.HasBlock(common.HexToHash("0x7f57774109cad543d9acfbcfa3630b30ca652d2310470341b78c62ee7463633b"))
	t.Log(hasBLock)
}

func TestFullBlockChain_QueryBlockFloor(t *testing.T) {
	common.InitConf("/Users/pxf/workspace/tas_develop/test9/tas9.ini")
	middleware.InitMiddleware()
	initBlockChain(nil)

	chain := BlockChainImpl.(*FullBlockChain)

	fmt.Println("=====")
	bh := chain.queryBlockHeaderByHeight(0)
	fmt.Println(bh, bh.Hash.String())
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