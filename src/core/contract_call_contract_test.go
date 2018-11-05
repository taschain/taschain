package core

import (
	"common"
	"encoding/json"
	"strconv"
	"testing"
	"time"
	"tvm"
)

func TestContractCallContract(t *testing.T) {
	Clear()

	print("\n1\n")
	code := tvm.Read0("/Users/mike/tas/code/tas/taschain/taschain/src/tvm/py/test/contract_becalled.py")
	contract := tvm.Contract{code, "ContractBeCalled", nil}
	jsonString, _ := json.Marshal(contract)
	//fmt.Println(string(jsonString))
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")

	print("\n2\n")
	time.Sleep(3 * time.Second)
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
	//断言
	result := string(getContractDatas("0xe4d60f63188f69980e762cb38aad8727ceb86bbe", "a"))
	i, _ := strconv.Atoi(result)
	if i != 210 {
		t.Fatal("call int failed.")
	}

	// str类型
	print("\n4\n")
	time.Sleep(3 * time.Second)
	contractAddr = "0xe4d60f63188f69980e762cb38aad8727ceb86bbe"
	abi = `{"FuncName": "contract_str", "Args": []}`
	CallContract(contractAddr, abi)
	result = string(getContractDatas("0xe4d60f63188f69980e762cb38aad8727ceb86bbe", "mystr"))
	if result != "\"myabcbcd\"" {
		t.Fatal("call str failed.")
	}

	// bool类型
	print("\n5\n")
	time.Sleep(4 * time.Second)
	contractAddr = "0xe4d60f63188f69980e762cb38aad8727ceb86bbe"
	abi = `{"FuncName": "contract_bool", "Args": []}`
	CallContract(contractAddr, abi)
	result = string(getContractDatas("0xe4d60f63188f69980e762cb38aad8727ceb86bbe", "mybool"))
	if result != "false" {
		t.Fatal("call bool failed.")
	}

	// none类型
	print("\n6\n")
	time.Sleep(3 * time.Second)
	contractAddr = "0xe4d60f63188f69980e762cb38aad8727ceb86bbe"
	abi = `{"FuncName": "contract_none", "Args": []}`
	CallContract(contractAddr, abi)
	result = string(getContractDatas("0xe4d60f63188f69980e762cb38aad8727ceb86bbe", "mynone"))
	if result != "null" {
		t.Fatal("call none failed.")
	}
}

// 合约深度测试用例，当前运行到第8层没有做控制，底层会有异常。整体控制是ok的，对应的日志也做了处理。
func TestContractMaxLength(t *testing.T) {
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

func getContractDatas(contractAddr string, keyName string) []byte {
	addr := common.HexStringToAddress(contractAddr)
	stateDb := BlockChainImpl.LatestStateDB()
	iterator := stateDb.DataIterator(addr, "")
	if iterator != nil {
		for iterator != nil {
			if string(iterator.Key) == keyName {
				return iterator.Value
			}

			if !iterator.Next() {
				break
			}
		}
	}

	return nil
}
