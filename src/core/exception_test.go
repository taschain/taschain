package core

import (
	"tvm"
	"fmt"
	"testing"
	"encoding/json"
)

func TestLib(t *testing.T) {
	ChainInit()
	source := "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
	code := `
class A():
    def __init__(self):
        pass

    @register.public(int, bool, str, list, dict)
    def test(self, aa, bb, cc, dd, ee):
        assert isinstance(aa, int)
        assert isinstance(bb, bool)
        assert isinstance(cc, str)
        assert isinstance(dd, list)
        assert isinstance(ee, dict)

`
	contract := tvm.Contract{code, "A", nil}
	jsonString, _ := json.Marshal(contract)
	contractAddr := DeployContract(string(jsonString), source, 200000, 0)
	// 测试正常数据调用
	hash := ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": [10, true, "a", [11], {"key": "value"}]}`, source, (2000000))
	receipt := BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	fmt.Printf("=============>get hash %x \n",hash)
	fmt.Println("test receipt", receipt)
	//if receipt.Receipt.Status != types2.ReceiptStatusSuccessful {
	//	t.Errorf("execute: failed, wanted succeed")
	//}

	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": ["10", true, "a", [11], {"key": "value"}]}`, source, (2000000))
	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	fmt.Printf("=============2>get hash %x \n",hash)
	fmt.Println("test receipt 2", receipt)

	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": [true, "a", [11], {"key": "value"}]}`, source, (2000000))
	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	fmt.Printf("=============3>get hash %x \n",hash)
	fmt.Println("test receipt 3", receipt)

	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test2", "Args": [10,true, "a", [11], {"key": "value"}]}`, source, (2000000))
	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	fmt.Printf("=============3>get hash %x \n",hash)
	fmt.Println("test receipt 3", receipt)
}

func TestGas(t *testing.T) {
	ChainInit()
	source := "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
	code := `
DefEvent('a')
DefEvent('b')

class A():
    def __init__(self):
        TEvents.a('123',{'val':10})
        #TEvents.a(123,{'val':10})
        #TEvents.a('123',"{'val':10}")

    @register.public()
    def test(self):
        TEvents.b('234',{'val':20})

    @register.public()
    def test1(self):
        TEvents.b(234,{'val':20})
        
    @register.public()
    def test2(self):
        TEvents.b('234',"{'val':20}")

    @register.public()
    def test3(self):
        DefEvent('c')
        TEvents.c('345',{'val':30})

    @register.public()
    def test4(self):
        DefEvent('c')
        TEvents.c(345,{'val':30})

    @register.public()
    def test5(self):
        DefEvent('c')
        TEvents.c('345',"{'val':30}")

`
	contract := tvm.Contract{code, "A", nil}
	jsonString, _ := json.Marshal(contract)
	contractAddr := DeployContract(string(jsonString), source, 200000, 0)
	// 测试正常数据调用
	hash := ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": []}`, source, (2000000))
	receipt := BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	fmt.Printf("=============>get hash %x \n",hash)
	fmt.Println("test receipt", receipt)

	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test1", "Args": []}`, source, (2000000))
	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	fmt.Printf("=============>get hash %x \n",hash)
	fmt.Println("test receipt", receipt)

	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test2", "Args": []}`, source, (2000000))
	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	fmt.Printf("=============>get hash %x \n",hash)
	fmt.Println("test receipt", receipt)

	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test3", "Args": []}`, source, (2000000))
	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	fmt.Printf("=============>get hash %x \n",hash)
	fmt.Println("test receipt", receipt)

	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test4", "Args": []}`, source, (2000000))
	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	fmt.Printf("=============>get hash %x \n",hash)
	fmt.Println("test receipt", receipt)

	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test5", "Args": []}`, source, (2000000))
	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
	fmt.Printf("=============>get hash %x \n",hash)
	fmt.Println("test receipt", receipt)
}
