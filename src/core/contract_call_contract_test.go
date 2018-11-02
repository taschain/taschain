package core

import (
	"encoding/json"
	"testing"
	"time"
	"tvm"
)

func TestContractCallContract(t *testing.T){
	Clear()

	code := tvm.Read0("/Users/mike/tas/code/tas/taschain/taschain/src/tvm/py/test/contract_becalled.py")
	contract := tvm.Contract{code, "ContractBeCalled", nil}
	jsonString, _ := json.Marshal(contract)
	//fmt.Println(string(jsonString))
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")

	time.Sleep(3 * time.Second)
	print("\n2\n")
	code = tvm.Read0("/Users/mike/tas/code/tas/taschain/taschain/src/tvm/py/test/contract_game.py")
	contract = tvm.Contract{code, "ContractGame", nil}
	jsonString, _ = json.Marshal(contract)
	//fmt.Println(string(jsonString))
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")

	//int类型
	print("\n3\n")
	time.Sleep(3 * time.Second)
	print("3")
	contractAddr := "0xe4d60f63188f69980e762cb38aad8727ceb86bbe"
	abi := `{"FuncName": "contract_int", "Args": []}`
	CallContract(contractAddr, abi)

	// str类型
	time.Sleep(3 * time.Second)
	print("3")
	contractAddr = "0xe4d60f63188f69980e762cb38aad8727ceb86bbe"
	abi = `{"FuncName": "contract_str", "Args": []}`
	CallContract(contractAddr, abi)

	// bool类型
	time.Sleep(4 * time.Second)
	print("3")
	contractAddr = "0xe4d60f63188f69980e762cb38aad8727ceb86bbe"
	abi = `{"FuncName": "contract_bool", "Args": []}`
	CallContract(contractAddr, abi)

	// none类型
	time.Sleep(3 * time.Second)
	print("3")
	contractAddr = "0xe4d60f63188f69980e762cb38aad8727ceb86bbe"
	abi = `{"FuncName": "contract_none", "Args": []}`
	CallContract(contractAddr, abi)
}


// 合约深度测试用例，当前运行到第8层没有做控制，底层会有异常。整体控制是ok的，对应的日志也做了处理。
func TestContractMaxLength(t *testing.T){
	Clear()
	//部署合约contract_becalled
	print("\n1\n")
	code := tvm.Read0("/Users/mike/tas/code/tas/taschain/taschain/src/tvm/py/test/contract_becalled.py")
	contract := tvm.Contract{code, "ContractBeCalled", nil}
	jsonString, _ := json.Marshal(contract)
	//fmt.Println(string(jsonString))
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")

	//部署合约contract_game
	time.Sleep(5 * time.Second)
	print("\n2\n")
	code = tvm.Read0("/Users/mike/tas/code/tas/taschain/taschain/src/tvm/py/test/contract_game.py")
	contract = tvm.Contract{code, "ContractGame", nil}
	jsonString, _ = json.Marshal(contract)
	//fmt.Println(string(jsonString))
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")

	//部署合约contract_becalled_deep
	time.Sleep(5 * time.Second)
	print("\n3\n")
	code = tvm.Read0("/Users/mike/tas/code/tas/taschain/taschain/src/tvm/py/test/contract_becalled_deep.py")
	contract = tvm.Contract{code, "ContractBeCalledDeep", nil}
	jsonString, _ = json.Marshal(contract)
	//fmt.Println(string(jsonString))
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")



	//int类型
	time.Sleep(5 * time.Second)
	print("3")
	contractAddr := "0xe4d60f63188f69980e762cb38aad8727ceb86bbe"
	abi := `{"FuncName": "contract_deep", "Args": []}`
	CallContract(contractAddr, abi)

}


