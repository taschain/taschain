package core
//
//import (
//	"tvm"
//	"fmt"
//	"testing"
//	"encoding/json"
//	"common"
//)
//
//func TestLib(t *testing.T){
//	eventTest(t)
//	storageTest(t)
//	decorateTest(t)
//}
//
//func TestGas(t *testing.T){
//	//onlinegasTest(t)	//it should be failed
//	abigasTest(t)		//it should be failed
//}
//
//var contractAddr common.Address
//func onlinegasTest(t *testing.T){
//	ChainInit()
//	source := "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
//	code := `
//class A():
//    def __init__(self):
//        pass
//
//    @register.public()
//    def test(self):
//        print('this step has gas')
//        print('this step has no gas')
//`
//	contract := tvm.Contract{code, "A", nil}
//	jsonString, _ := json.Marshal(contract)
//	contractAddr = DeployContract(string(jsonString), source, 9126, 0)
//}
//
//func abigasTest(t *testing.T){
//	ChainInit()
//	source := "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
//	code := `
//class A():
//    def __init__(self):
//        pass
//
//    @register.public()
//    def test(self):
//        print('this step has gas')
//        print('this step has no gas')
//`
//	contract := tvm.Contract{code, "A", nil}
//	jsonString, _ := json.Marshal(contract)
//	contractAddr = DeployContract(string(jsonString), source, 9127, 0)
//	// 测试正常数据调用
//	hash := ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": []}`, source, 9664)
//	receipt := BlockChainImpl.GetTransactionPool().GetExecuted(hash)
//	fmt.Printf("=============>get hash %x \n",hash)
//	fmt.Println("test receipt", receipt)
//}
//
//func decorateTest(t *testing.T) {
//	ChainInit()
//	source := "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
//	code := `
//class A():
//    def __init__(self):
//        pass
//
//    @register.public(int, bool, str, list, dict)
//    def test(self, aa, bb, cc, dd, ee):
//        assert isinstance(aa, int)
//        assert isinstance(bb, bool)
//        assert isinstance(cc, str)
//        assert isinstance(dd, list)
//        assert isinstance(ee, dict)
//
//`
//	contract := tvm.Contract{code, "A", nil}
//	jsonString, _ := json.Marshal(contract)
//	contractAddr := DeployContract(string(jsonString), source, 200000, 0)
//	// 测试正常数据调用
//	hash := ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": [10, true, "a", [11], {"key": "value"}]}`, source, (2000000))
//	receipt := BlockChainImpl.GetTransactionPool().GetExecuted(hash)
//	fmt.Printf("=============>get hash %x \n",hash)
//	fmt.Println("test receipt", receipt)
//	//if receipt.Receipt.Status != types2.ReceiptStatusSuccessful {
//	//	t.Errorf("execute: failed, wanted succeed")
//	//}
//
//	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": ["10", true, "a", [11], {"key": "value"}]}`, source, (2000000))
//	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
//	fmt.Printf("=============2>get hash %x \n",hash)
//	fmt.Println("test receipt 2", receipt)
//
//	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": [true, "a", [11], {"key": "value"}]}`, source, (2000000))
//	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
//	fmt.Printf("=============3>get hash %x \n",hash)
//	fmt.Println("test receipt 3", receipt)
//
//	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test2", "Args": [10,true, "a", [11], {"key": "value"}]}`, source, (2000000))
//	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
//	fmt.Printf("=============3>get hash %x \n",hash)
//	fmt.Println("test receipt 3", receipt)
//}
//
//func eventTest(t *testing.T) {
//	ChainInit()
//	source := "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
//	code := `
//DefEvent('a')
//DefEvent('b')
//
//class A():
//    def __init__(self):
//        TEvents.a('123',{'val':10})
//        #TEvents.a(123,{'val':10})
//        #TEvents.a('123',"{'val':10}")
//
//    @register.public()
//    def test(self):
//        TEvents.b('234',{'val':20})
//
//    @register.public()
//    def test1(self):
//        TEvents.b(234,{'val':20})
//
//    @register.public()
//    def test2(self):
//        TEvents.b('234',"{'val':20}")
//
//    @register.public()
//    def test3(self):
//        DefEvent('c')
//        TEvents.c('345',{'val':30})
//
//    @register.public()
//    def test4(self):
//        DefEvent('c')
//        TEvents.c(345,{'val':30})
//
//    @register.public()
//    def test5(self):
//        DefEvent('c')
//        TEvents.c('345',"{'val':30}")
//
//`
//	contract := tvm.Contract{code, "A", nil}
//	jsonString, _ := json.Marshal(contract)
//	contractAddr := DeployContract(string(jsonString), source, 200000, 0)
//	// 测试正常数据调用
//	hash := ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test", "Args": []}`, source, (2000000))
//	receipt := BlockChainImpl.GetTransactionPool().GetExecuted(hash)
//	fmt.Printf("=============>get hash %x \n",hash)
//	fmt.Println("test receipt", receipt)
//
//	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test1", "Args": []}`, source, (2000000))
//	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
//	fmt.Printf("=============>get hash %x \n",hash)
//	fmt.Println("test receipt", receipt)
//
//	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test2", "Args": []}`, source, (2000000))
//	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
//	fmt.Printf("=============>get hash %x \n",hash)
//	fmt.Println("test receipt", receipt)
//
//	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test3", "Args": []}`, source, (2000000))
//	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
//	fmt.Printf("=============>get hash %x \n",hash)
//	fmt.Println("test receipt", receipt)
//
//	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test4", "Args": []}`, source, (2000000))
//	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
//	fmt.Printf("=============>get hash %x \n",hash)
//	fmt.Println("test receipt", receipt)
//
//	hash = ExecuteContract(contractAddr.GetHexString(), `{"FuncName": "test5", "Args": []}`, source, (2000000))
//	receipt = BlockChainImpl.GetTransactionPool().GetExecuted(hash)
//	fmt.Printf("=============>get hash %x \n",hash)
//	fmt.Println("test receipt", receipt)
//}
//
//func storageTest(t *testing.T){
//	Clear()
//
//	code := tvm.Read0("../tvm/py/test/contract_storage_exception.py")
//	contract := tvm.Contract{code, "ContractStorageException", nil}
//	jsonString, _ := json.Marshal(contract)
//	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
//
//	contractAddr := "0xe4d60f63188f69980e762cb38aad8727ceb86bbe"
//	abi := `{"FuncName": "callExcption1", "Args": []}`
//	CallContract(contractAddr, abi)
//	if !hasData(contractAddr,"callExcption1"){
//		t.Fatal("call contract failed.")
//	}
//
//
//	abi = `{"FuncName": "callExcption2", "Args": []}`
//	CallContract(contractAddr, abi)
//	if !hasData(contractAddr,"callExcption2"){
//		t.Fatal("call contract failed.")
//	}
//
//	abi = `{"FuncName": "callExcption3", "Args": []}`
//	CallContract(contractAddr, abi)
//	if !hasData(contractAddr,"callExcption3"){
//		t.Fatal("call contract failed.")
//	}
//}
//
//func TestSyntaxError(t *testing.T) {
//	Clear()
//
//	code := tvm.Read0("../tvm/py/test/contract_syntax_exception.py")
//	contract := tvm.Contract{code, "ContractSyntax", nil}
//	jsonString, _ := json.Marshal(contract)
//	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
//
//	contractAddr := "0x9a6bf01ba09a5853f898b2e9e6569157a01a7a00"
//	abi := `{"FuncName": "callExcption1", "Args": []}`
//	CallContract(contractAddr, abi)
//
//	if !hasData(contractAddr,"data"){
//		t.Fatal("call contract failed.")
//	}
//}
//
//func hasData(address string,key string)bool{
//	datas := GetContractDatas(address)
//	if datas[key] !=  `1"success"`{
//		return false
//	}
//	return true
//}
