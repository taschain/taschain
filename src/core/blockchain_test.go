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
	"common"
	"math/rand"
	"testing"
	"time"

	"fmt"
	"github.com/gin-gonic/gin/json"
	"middleware"
	"middleware/types"
	"network"
	"taslog"
	"tvm"
)

func init() {
	middleware.InitMiddleware()
}


func OnChainFunc(code string, source string) {
	common.InitConf("d:/test1.ini")
	//common.InitConf(os.Getenv("HOME") + "/tas/code/tas/taschain/taschain/deploy/tvm/test1.ini")
	network.Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
	//Clear()
	initBlockChain()
	BlockChainImpl.transactionPool.Clear()
	txpool := BlockChainImpl.GetTransactionPool()
	index := uint64(time.Now().Unix())
	fmt.Println(index)
	txpool.Add(genContractTx(123456, source, "", index, 0, []byte(code), nil, 0))
	fmt.Println("nonce:", BlockChainImpl.GetNonce(common.HexStringToAddress(source)))
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(common.HexStringToAddress(source).Bytes(), common.Uint64ToByte(BlockChainImpl.GetNonce(common.HexStringToAddress(source))))))
	castor := new([]byte)
	groupid := new([]byte)
	// 铸块1
	block := BlockChainImpl.CastingBlock(BlockChainImpl.Height() + 1, 12, 0, *castor, *groupid)
	if nil == block {
		fmt.Println("fail to cast new block")
	}
	// 上链
	if 0 != BlockChainImpl.AddBlockOnChain(block) {
		fmt.Println("fail to add block")
	}
	fmt.Println(contractAddr.GetHexString())
}

func CallContract(address, abi string) {
	CallContract2(address, abi, "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
}

func CallContract2(address, abi string, source string) {
	common.InitConf("d:/test1.ini")
	//common.InitConf(os.Getenv("HOME") + "/tas/code/tas/taschain/taschain/deploy/tvm/test1.ini")
	network.Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
	initBlockChain()
	BlockChainImpl.transactionPool.Clear()
	castor := new([]byte)
	groupid := new([]byte)
	contractAddr := common.HexStringToAddress(address)
	code := BlockChainImpl.latestStateDB.GetCode(contractAddr)
	fmt.Println(string(code))
	txpool := BlockChainImpl.GetTransactionPool()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	txpool.Add(genContractTx(123456, source, contractAddr.GetHexString(), r.Uint64(), 44, []byte(abi), nil, 0))
	block2 := BlockChainImpl.CastingBlock(BlockChainImpl.Height() + 1, 123, 0, *castor, *groupid)
	block2.Header.QueueNumber = 2
	if 0 != BlockChainImpl.AddBlockOnChain(block2) {
		fmt.Println("fail to add empty block")
	}
}

func TestVmTest(t *testing.T)  {
	Clear()
	code := tvm.Read0("/Users/guangyujing/workspace/tas/src/tvm/py/token/contract_token_tas.py")

	contract := tvm.Contract{code, "MyAdvancedToken", nil}
	jsonString, _ := json.Marshal(contract)
	fmt.Println(string(jsonString))
	contractAddress := common.HexToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
	OnChainFunc(string(jsonString), contractAddress.GetHexString())
}

func VmTest1(code string)  {
	contractAddr := "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
	abi := code
	sourceAddr := common.HexToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
	CallContract2(contractAddr, abi, sourceAddr.GetHexString())
}

func TestVmTest2(t *testing.T)  {
	code := tvm.Read0("/Users/guangyujing/workspace/tas/src/tvm/py/recharge/recharge.py")

	contract := tvm.Contract{code, "Recharge", nil}
	jsonString, _ := json.Marshal(contract)
	fmt.Println(string(jsonString))
	contractAddress := common.HexToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
	OnChainFunc(string(jsonString), contractAddress.GetHexString())
}

func VmTest2(code string)  {
	contractAddr := "0x1ed70a8b95d348573aaa5414d6cd9b1cccc22831"
	abi := code
	contractAddress := common.HexToAddress("0x00000001")
	CallContract2(contractAddr, abi, contractAddress.GetHexString())
}

func TestVmTest3(t *testing.T)  {

	VmTest1(`{"FuncName": "transfer", "Args": ["0x0000000300000000000000000000000000000000", 1000]}`)
	//VmTest1(`{"FuncName": "set_prices", "Args": [100, 100]}`)
	//VmTest1(`{"FuncName": "burn", "Args": [2500]}`)
	//VmTest1(`{"FuncName": "mint_token", "Args": ["0x0000000100000000000000000000000000000000", 5000]}`)

	//VmTest1(`{"FuncName": "approveAndCall", "Args": ["0xe4d60f63188f69980e762cb38aad8727ceb86bbe", 50, "13968999999"]}`)
}









func Test_Deploy_Contract1(t *testing.T)  {

	code := `
import account
class A():
    def __init__(self):
        self.a = 10

    def deploy(self):
        print("deploy")

    @register.public(int,str,list,dict)
    def test(self,aa,bb,cc,dd):
        print(aa)
        print(bb)
        print(cc)
        print(dd)
        #account.transfer("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b", 50)
        pass

    
    def test2(self):
        print("test2")
        #account.transfer("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b", 50)
        pass
`
    contract := tvm.Contract{code, "A", nil}
    jsonString, _ := json.Marshal(contract)
    fmt.Println(string(jsonString))
    OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")

	//code := tvm.Read0("/Users/mike/tas/code/tas/taschain/taschain/src/tvm/py/test/contract_becalled.py")
	//contract := tvm.Contract{code, "ContractBeCalled", nil}
	//jsonString, _ := json.Marshal(contract)
	////fmt.Println(string(jsonString))
	//OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
}

func Test_Deploy_Contract2(t *testing.T)  {
	code := tvm.Read0("/Users/mike/tas/code/tas/taschain/taschain/src/tvm/py/test/contract_game.py")
	contract := tvm.Contract{code, "ContractGame", nil}
	jsonString, _ := json.Marshal(contract)
	//fmt.Println(string(jsonString))
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
}


func TestCallConstract(t *testing.T)  {
	contractAddr := "0xf5f946643f8847e48cfb6e1dbca803246500613e"
	abi := `{"FuncName": "test", "Args": [10,"fffff",["aa","bb",33],{"name":"xxxx","value":777}]}`
	CallContract(contractAddr, abi)
}


func Test_Clear(t *testing.T){
	Clear()
}



















func TestCallConstract2(t *testing.T)  {
	contractAddr := "0xf744049b3381ca85b36c50ed3cced8c17bb5ea28"
	abi := `{"FuncName": "test2", "Args": []}`
	CallContract(contractAddr, abi)
}

func TestBlockChain_AddBlock(t *testing.T) {
	common.InitConf("d:/test1.ini")
	//common.InitConf(os.Getenv("HOME") + "/tas/code/tas/taschain/taschain/deploy/tvm/test1.ini")
	network.Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
	Clear()
	initBlockChain()
	middleware.InitMiddleware()
	BlockChainImpl.transactionPool.Clear()
	//BlockChainImpl.Clear()

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
	txpool.Add(genTestTx("jdai1", 12345, "100", "2", 0, 1))
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

	sourcebyte := common.HexStringToAddress(source)
	sourceAddr = &sourcebyte
	if target == "" {
		targetAddr = nil
	} else {
		targetbyte := common.HexStringToAddress(target)
		targetAddr = &targetbyte
	}
	return &types.Transaction{
		Hash: common.BytesToHash([]byte{byte(nonce)}),
		Data:          data,
		GasLimit:		2000000000,
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

func TestMinerOnChain(t *testing.T)  {
	Clear()
	code := tvm.Read0("/Users/guangyujing/workspace/tas/src/tvm/py/miner/miner.py")

	contract := tvm.Contract{code, "miner", nil}
	jsonString, _ := json.Marshal(contract)
	fmt.Println(string(jsonString))
	contractAddress := common.HexToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
	OnChainFunc(string(jsonString), contractAddress.GetHexString())
}

func TestMinerCall(t *testing.T)  {
	//VmTest1(`{"FuncName": "register", "Args": ["0x0000000300000000000000000000000000000000", 0]}`)
	//VmTest1(`{"FuncName": "test_print", "Args": []}`)

	//VmTest1(`{"FuncName": "deregister", "Args": ["0x0000000300000000000000000000000000000000"]}`)
	//VmTest1(`{"FuncName": "test_print", "Args": []}`)

	//VmTest1(`{"FuncName": "withdraw", "Args": ["0x0000000300000000000000000000000000000000"]}`)
	//VmTest1(`{"FuncName": "test_print", "Args": []}`)
}