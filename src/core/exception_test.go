package core

import (
	"testing"
	"tvm"
	"encoding/json"
)

func TestGasNotEnough(t *testing.T) {

}

func TestCallError(t *testing.T) {
	Clear()

	code := tvm.Read0("../tvm/py/test/contract_exception.py")
	contract := tvm.Contract{code, "ContractException", nil}
	jsonString, _ := json.Marshal(contract)
	//fmt.Println(string(jsonString))
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")

	contractAddr := "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
	abi := `{"FuncName": "callExcption1", "Args": []}`
	CallContract(contractAddr, abi)

}

func TestSyntaxError(t *testing.T) {

}