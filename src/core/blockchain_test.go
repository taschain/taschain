//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

import (
	"testing"
	"common"

	"fmt"
	"middleware/types"
	"network"
	"taslog"
	"os"
)

func TestBlockChain_AddBlock(t *testing.T) {
	common.InitConf(os.Getenv("HOME") + "/TasProject/work/1g3n/test1.ini")
	network.Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
	Clear()
	initBlockChain()
	BlockChainImpl.transactionPool.Clear()
	//chain.Clear()

	// 查询创始块
	blockHeader := BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 0 != blockHeader.Height {
		t.Fatalf("clear data fail")
	}

	if BlockChainImpl.latestStateDB.GetBalance(common.BytesToAddress(genHash("1"))).Int64() != 1000000 {
		t.Fatalf("fail to init 1 balace to 100")
	}

	txpool := BlockChainImpl.GetTransactionPool()
	if nil == txpool {
		t.Fatalf("fail to get txpool")
	}
	code := `
import account
def Test(a, b, c, d):
	print("hehe")
`
	// 交易1
	txpool.Add(genTestTx("jdai1", 12345, "1", "2", 0, 1))
	txpool.Add(genContractTx(123456, "1", "", 1, 0, []byte(code), nil, 0))
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine([]byte("1"), common.Uint64ToByte(0))))
	//交易2
	txpool.Add(genTestTx("jdai2", 123456, "2", "3", 0, 1))

	//交易3 执行失败的交易
	txpool.Add(genTestTx("jdaiddd", 123456, "2", "3", 0, 1))

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

	if BlockChainImpl.latestStateDB.GetBalance(common.BytesToAddress(genHash("1"))).Int64() != 999999 {
		t.Fatalf("fail to transfer 1 from 1  to 2")
	}

	// 池子中交易的数量为0
	if 0 != txpool.received.Len() {
		t.Fatalf("fail to remove transactions after addBlock")
	}

	//交易3
	txpool.Add(genTestTx("jdai3", 1, "1", "2", 2, 10))
	txpool.Add(genContractTx(123456, "1", contractAddr.GetHexString(), 3, 0, []byte(`{"FuncName": "Test", "Args": [10.123, "ten", [1, 2], {"key":"value", "key2":"value2"}]}`), nil, 0))
	fmt.Println(contractAddr.GetHexString())
	// 铸块2
	block2 := BlockChainImpl.CastingBlock(2, 123, 0, *castor, *groupid)
	block2.Header.QueueNumber = 2
	if 0 != BlockChainImpl.AddBlockOnChain(block2) {
		t.Fatalf("fail to add empty block")
	}

	if BlockChainImpl.latestStateDB.GetBalance(common.BytesToAddress(genHash("1"))).Int64() != 999989 {
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

	block4 := BlockChainImpl.CastingBlock(4, 126, 0, *castor, *groupid)
	if 0 != BlockChainImpl.AddBlockOnChain(block4) {
		t.Fatalf("fail to add empty block")
	}
	//最新块是块3
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 4 != blockHeader.Height || blockHeader.Hash != block4.Header.Hash {
		t.Fatalf("add block3 failed")
	}

	block5 := BlockChainImpl.CastingBlock(5, 127, 0, *castor, *groupid)
	if 0 != BlockChainImpl.AddBlockOnChain(block5) {
		t.Fatalf("fail to add empty block")
	}
	//最新块是块5
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 5 != blockHeader.Height || blockHeader.Hash != block5.Header.Hash {
		t.Fatalf("add block3 failed")
	}

	// 铸块4 空块
	// 模拟分叉
	//block4 := BlockChainImpl.CastingBlockAfter(block.Header, 2, 124, 0, *castor, *groupid)
	//
	//if 0 != BlockChainImpl.AddBlockOnChain(block4) {
	//	t.Fatalf("fail to add empty block")
	//}
	////最新块是块4
	//blockHeader = BlockChainImpl.QueryTopBlock()
	//if nil == blockHeader || 2 != blockHeader.Height || blockHeader.Hash != block4.Header.Hash {
	//	t.Fatalf("add block4 failed")
	//}
	//blockHeader = BlockChainImpl.QueryBlockByHeight(3)
	//if nil != blockHeader {
	//	t.Fatalf("failed to remove uncle blocks")
	//}
	//
	//if BlockChainImpl.latestStateDB.GetBalance(c.BytesToAddress(genHash("1"))).Int64() != 999999 {
	//	t.Fatalf("fail to switch to main chain. %d", BlockChainImpl.latestStateDB.GetBalance(c.BytesToAddress(genHash("1"))))
	//}

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

		lowestCache := BlockChainImpl.queryBlockHeaderByHeight(i, true)
		if nil == lowestCache {
			t.Fatalf("fail to get lowest block from cache,%d", i)
		}

		bh := lowest.(*types.BlockHeader)
		if bh.Height != lowestLDB.Height || bh.Height != lowestCache.Height || lowestLDB.Height != lowestCache.Height {
			t.Fatalf("fail to check block from cache to ldb,%d", i)
		}
	}
}

func TestBlockChain_StateTree(t *testing.T) {

	Clear()
	initBlockChain()
	BlockChainImpl.transactionPool.Clear()
	//chain.Clear()

	// 查询创始块
	blockHeader := BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 0 != blockHeader.Height {
		t.Fatalf("clear data fail")
	}

	if BlockChainImpl.latestStateDB.GetBalance(common.BytesToAddress(genHash("1"))).Int64() != 100 {
		t.Fatalf("fail to init 1 balace to 100")
	}

	txpool := BlockChainImpl.GetTransactionPool()
	if nil == txpool {
		t.Fatalf("fail to get txpool")
	}

	castor := new([]byte)
	groupid := new([]byte)

	block0 := BlockChainImpl.CastingBlock(1, 12, 0, *castor, *groupid)
	// 上链
	if 0 != BlockChainImpl.AddBlockOnChain(block0) {
		t.Fatalf("fail to add block0")
	}

	// 交易1
	txpool.Add(genTestTx("jdai1", 12345, "1", "2", 0, 1))

	//交易2
	txpool.Add(genTestTx("jdai2", 123456, "2", "3", 0, 2))

	// 交易3 失败的交易
	txpool.Add(genTestTx("jdai3", 123457, "1", "2", 0, 3))

	// 铸块1
	block := BlockChainImpl.CastingBlock(2, 12, 0, *castor, *groupid)
	if nil == block {
		t.Fatalf("fail to cast new block")
	}

	// 上链
	if 0 != BlockChainImpl.AddBlockOnChain(block) {
		t.Fatalf("fail to add block")
	}

	// 铸块2
	block2 := BlockChainImpl.CastingBlock(3, 12, 0, *castor, *groupid)
	if nil == block {
		t.Fatalf("fail to cast new block")
	}

	// 上链
	if 0 != BlockChainImpl.AddBlockOnChain(block2) {
		t.Fatalf("fail to add block")
	}
	fmt.Printf("state: %d\n", BlockChainImpl.latestBlock.StateTree)

	// 铸块3
	block3 := BlockChainImpl.CastingBlock(4, 12, 0, *castor, *groupid)
	if nil == block {
		t.Fatalf("fail to cast new block")
	}

	// 上链
	if 0 != BlockChainImpl.AddBlockOnChain(block3) {
		t.Fatalf("fail to add block")
	}
	fmt.Printf("state: %d\n", BlockChainImpl.latestBlock.StateTree)
}

func genTestTx(hash string, price uint64, source string, target string, nonce uint64, value uint64) *types.Transaction {

	sourcebyte := common.BytesToAddress(genHash(source))
	targetbyte := common.BytesToAddress(genHash(target))

	return &types.Transaction{
		GasPrice: price,
		Hash:     common.BytesToHash(genHash(hash)),
		Source:   &sourcebyte,
		Target:   &targetbyte,
		Nonce:    nonce,
		Value:    value,
	}
}

func genContractTx(price uint64, source string, target string, nonce uint64, value uint64, data []byte, extraData []byte, extraDataType int32) *types.Transaction {
	var sourceAddr, targetAddr *common.Address

	sourcebyte := common.BytesToAddress([]byte(source))
	sourceAddr = &sourcebyte
	if target == "" {
		targetAddr = nil
	} else {
		targetbyte := common.HexStringToAddress(target)
		targetAddr = &targetbyte
	}
	return &types.Transaction{
		Data:          data,
		GasPrice:      price,
		Source:        sourceAddr,
		Target:        targetAddr,
		Nonce:         nonce,
		Value:         value,
		ExtraData:     extraData,
		ExtraDataType: extraDataType,
	}
}

func genHash(hash string) []byte {
	bytes3 := []byte(hash)
	return common.Sha256(bytes3)
}
