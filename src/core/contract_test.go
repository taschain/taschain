package core

import (
	"common"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"middleware"
	"middleware/types"
	"network"
	"taslog"
	"testing"
	"time"
	"tvm"
	types2 "storage/core/types"
)

func init() {
	middleware.InitMiddleware()
}


func ChainInit() {
	Clear()
	common.InitConf("../../deploy/tvm/test1.ini")
	network.Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
	initBlockChain()
	BlockChainImpl.transactionPool.Clear()
}

func genContractTx(price uint64, gaslimit uint64, source string, target string, nonce uint64, value uint64, data []byte, extraData []byte, extraDataType int32) *types.Transaction {
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
		GasLimit:		gaslimit,
		GasPrice:      price,
		Source:        sourceAddr,
		Target:        targetAddr,
		Nonce:         nonce,
		Value:         value,
		ExtraData:     extraData,
		ExtraDataType: extraDataType,
	}
}

func DeployContract(code string, source string, gaslimit uint64, value uint64) common.Address{
	txpool := BlockChainImpl.GetTransactionPool()
	index := uint64(time.Now().Unix())
	txpool.Add(genContractTx(1, gaslimit, source, "", index, value, []byte(code), nil, 0))
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(common.HexStringToAddress(source).Bytes(), common.Uint64ToByte(BlockChainImpl.GetNonce(common.HexStringToAddress(source))))))
	castor := new([]byte)
	groupid := new([]byte)
	block := BlockChainImpl.CastingBlock(BlockChainImpl.Height() + 1, 12, 0, *castor, *groupid)
	if nil == block {
		fmt.Println("fail to cast new block")
	}
	// 上链
	if 0 != BlockChainImpl.AddBlockOnChain(block) {
		fmt.Println("fail to add block")
	}
	return contractAddr
}

func ExecuteContract(address, abi string, source string, gaslimit uint64) common.Hash{
	contractAddr := common.HexStringToAddress(address)
	code := BlockChainImpl.latestStateDB.GetCode(contractAddr)
	fmt.Println(string(code))
	txpool := BlockChainImpl.GetTransactionPool()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	trans := genContractTx(1, gaslimit, source, contractAddr.GetHexString(), r.Uint64(), 44, []byte(abi), nil, 0)
	txpool.Add(trans)
	castor := new([]byte)
	groupid := new([]byte)
	block2 := BlockChainImpl.CastingBlock(BlockChainImpl.Height() + 1, 123, 0, *castor, *groupid)
	block2.Header.QueueNumber = 2
	if 0 != BlockChainImpl.AddBlockOnChain(block2) {
		fmt.Println("fail to add empty block")
	}
	return trans.Hash
}

func TestGasUse(t *testing.T) {
	ChainInit()
	source := "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
	balance0 := BlockChainImpl.GetBalance(common.HexStringToAddress(source))
	code := `
import account
class A():
    def __init__(self):
        self.a = 10

    def deploy(self):
        print("deploy")

    @register.public(int)
    def test(self, i):
		for i in range(i):
			pass
`
	contract := tvm.Contract{code, "A", nil}
	jsonString, _ := json.Marshal(contract)
	// test deploy
	contractAddr := DeployContract(string(jsonString), source, 200000, 0)
	balance1 := BlockChainImpl.GetBalance(common.HexStringToAddress(source))
	tmp := big.NewInt(0).Sub(balance0, balance1)
	if tmp.Int64() != 9389 {
		t.Errorf("deploy gas used: wannted %d, got %d",9389, tmp.Int64())
	}
	// test call "test" function
	ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": [10]}`, source, 2000000)
	balance2 := BlockChainImpl.GetBalance(common.HexStringToAddress(source))
	tmp = big.NewInt(0).Sub(balance1, balance2)
	if tmp.Int64() != 10183 {
		t.Errorf("call 'test' function gas used: wannted %d, got %d",10183, tmp.Int64())
	}
	// test call "test" function
	ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": [20]}`, source, 2000000)
	balance3 := BlockChainImpl.GetBalance(common.HexStringToAddress(source))
	tmp = big.NewInt(0).Sub(balance2, balance3)
	if tmp.Int64() != 10343 {
		t.Errorf("call 'test' function gas used: wannted %d, got %d",10343, tmp.Int64())
	}
	// test call "test" function
	ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": [30]}`, source, 2000000)
	balance4 := BlockChainImpl.GetBalance(common.HexStringToAddress(source))
	tmp = big.NewInt(0).Sub(balance3, balance4)
	if tmp.Int64() != 10503 {
		t.Errorf("call 'test' function gas used: wannted %d, got %d",10503, tmp.Int64())
	}
	// test run out gas
	//ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": [456]}`, source, 5000)
	//balance4 := BlockChainImpl.GetBalance(common.HexStringToAddress(source))
	//tmp = big.NewInt(0).Sub(balance3, balance4)
	//if tmp.Int64() != 5000 {
	//	t.Errorf("call 'test' function gas used: wannted %d, got %d",5000, tmp.Int64())
	//}
}

func TestAccessControl(t *testing.T) {
	ChainInit()
	source := "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
	code := `
class A():
    def __init__(self):
        pass

    def deploy(self):
        pass

    @register.public(int, bool, str, list, dict)
    def test(self, aa, bb, cc, dd, ee):
        assert isinstance(aa, int)
        assert isinstance(bb, bool)
        assert isinstance(cc, str)
        assert isinstance(dd, list)
        assert isinstance(ee, dict)

    def test2(self):
        print("test2")
`
	contract := tvm.Contract{code, "A", nil}
	jsonString, _ := json.Marshal(contract)
	contractAddr := DeployContract(string(jsonString), source, 200000, 0)
	// 测试正常数据调用
	hash := ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": [10, true, "a", [11], {"key": "value"}]}`, source, 2000000)
	receipt := BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	if receipt.Receipt.Status != types2.ReceiptStatusSuccessful {
		t.Errorf("execute: failed, wanted succeed")
	}
	// 测试错误数据调用
	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": ["", true, "a", [11], {"key": "value"}]}`, source, 2000000)
	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	if receipt.Receipt.Status != types2.ReceiptStatusFailed {
		t.Errorf("execute: succeed, wanted failed")
	}
	// 测试错误数据调用
	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": [10, 10, "a", [11], {"key": "value"}]}`, source, 2000000)
	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	if receipt.Receipt.Status != types2.ReceiptStatusFailed {
		t.Errorf("execute: succeed, wanted failed")
	}
	// 测试错误数据调用
	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": [10, true, 10, [11], {"key": "value"}]}`, source, 2000000)
	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	if receipt.Receipt.Status != types2.ReceiptStatusFailed {
		t.Errorf("execute: succeed, wanted failed")
	}
	// 测试错误数据调用
	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": [10, true, "a", 10, {"key": "value"}]}`, source, 2000000)
	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	if receipt.Receipt.Status != types2.ReceiptStatusFailed {
		t.Errorf("execute: succeed, wanted failed")
	}
	// 测试错误数据调用
	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": [10, true, "a", [11], 10]}`, source, 2000000)
	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	if receipt.Receipt.Status != types2.ReceiptStatusFailed {
		t.Errorf("execute: succeed, wanted failed")
	}
	// 测试私有方法调用
	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test2", "Args": []}`, source, 2000000)
	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	if receipt.Receipt.Status != types2.ReceiptStatusFailed {
		t.Errorf("execute: succeed, wanted failed")
	}
}

func OnChainFunc(code string, source string) {
	//common.InitConf("d:/test1.ini")
	common.InitConf("../../deploy/tvm/test1.ini")
	network.Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
	//Clear()
	initBlockChain()
	BlockChainImpl.transactionPool.Clear()
	txpool := BlockChainImpl.GetTransactionPool()
	index := uint64(time.Now().Unix())
	txpool.Add(genContractTx(1, 20000000,  source, "", index, 0, []byte(code), nil, 0))
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
	txpool.Add(genContractTx(1, 20000000, source, contractAddr.GetHexString(), r.Uint64(), 44, []byte(abi), nil, 0))
	block2 := BlockChainImpl.CastingBlock(BlockChainImpl.Height() + 1, 123, 0, *castor, *groupid)
	block2.Header.QueueNumber = 2
	if 0 != BlockChainImpl.AddBlockOnChain(block2) {
		fmt.Println("fail to add empty block")
	}
}

//func TestVmTest(t *testing.T)  {
//	Clear()
//	code := tvm.Read0("/Users/yushenghui/tas/src/tvm/py/token/contract_token_tas.py")
//
//	contract := tvm.Contract{code, "MyAdvancedToken", nil}
//	jsonString, _ := json.Marshal(contract)
//	fmt.Println(string(jsonString))
//	contractAddress := common.HexToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
//	OnChainFunc(string(jsonString), contractAddress.GetHexString())
//}
//
//func VmTest1(code string)  {
//	contractAddr := "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
//	abi := code
//	sourceAddr := common.HexToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
//	CallContract2(contractAddr, abi, sourceAddr.GetHexString())
//}
//
//func TestVmTest2(t *testing.T)  {
//	code := tvm.Read0("/Users/guangyujing/workspace/tas/src/tvm/py/recharge/recharge.py")
//
//	contract := tvm.Contract{code, "Recharge", nil}
//	jsonString, _ := json.Marshal(contract)
//	fmt.Println(string(jsonString))
//	contractAddress := common.HexToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
//	OnChainFunc(string(jsonString), contractAddress.GetHexString())
//}

