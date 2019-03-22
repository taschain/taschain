package core

import (
	"testing"
	"common"
	"middleware/types"
	"middleware"
	"utility"
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
	//top := gchain.latestBlock
	//t.Log(top.Height, top.Hash.String())
	//
	//for h := uint64(4460); h <= 4480; h++ {
	//	bh := gchain.queryBlockHeaderByHeightFloor(h)
	//	t.Log(bh.Height, bh.Hash.String())
	//}

	iter := chain.blockHeight.NewIterator()
	defer iter.Release()
	height := uint64(1320)

	if iter.Seek(utility.UInt64ToByte(height)) {

	}
	for iter.Next() {
		h := utility.ByteToUInt64(iter.Key()[2:])
		t.Log(h, iter.Value())
	}

}