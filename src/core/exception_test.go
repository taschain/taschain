package core

import (
	"testing"
	"tvm"
	"encoding/json"
)

func TestGasNotEnough(t *testing.T) {

}

func hasData(address string,key string)bool{
	datas := GetContractDatas(address)
	if datas[key] !=  `1"success"`{
		return false
	}
	return true
}

func TestCallError(t *testing.T) {
	Clear()

	code := tvm.Read0("../tvm/py/test/contract_call_exception.py")
	contract := tvm.Contract{code, "ContractException", nil}
	jsonString, _ := json.Marshal(contract)
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")

	contractAddr := "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
	abi := `{"FuncName": "callExcption1", "Args": []}`
	CallContract(contractAddr, abi)

	if !hasData(contractAddr,"data"){
		t.Fatal("call contract failed.")
	}
}

func TestLib(t *testing.T){
	storageTest(t)
}

func storageTest(t *testing.T){
	Clear()

	code := tvm.Read0("../tvm/py/test/contract_storage_exception.py")
	contract := tvm.Contract{code, "ContractStorageException", nil}
	jsonString, _ := json.Marshal(contract)
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")

	contractAddr := "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
	abi := `{"FuncName": "callExcption1", "Args": []}`
	CallContract(contractAddr, abi)
	if !hasData(contractAddr,"callExcption1"){
		t.Fatal("call contract failed.")
	}

	contractAddr = "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
	abi = `{"FuncName": "callExcption2", "Args": []}`
	CallContract(contractAddr, abi)
	if !hasData(contractAddr,"callExcption2"){
		t.Fatal("call contract failed.")
	}

	contractAddr = "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
	abi = `{"FuncName": "callExcption3", "Args": []}`
	CallContract(contractAddr, abi)
	if !hasData(contractAddr,"callExcption3"){
		t.Fatal("call contract failed.")
	}
}

func TestSyntaxError(t *testing.T) {
	Clear()

	code := tvm.Read0("../tvm/py/test/contract_syntax_exception.py")
	contract := tvm.Contract{code, "ContractSyntax", nil}
	jsonString, _ := json.Marshal(contract)
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")

	contractAddr := "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
	abi := `{"FuncName": "callExcption1", "Args": []}`
	CallContract(contractAddr, abi)

	if !hasData(contractAddr,"data"){
		t.Fatal("call contract failed.")
	}
}