package core

import (
	"testing"

	"fmt"
	"tvm"
	"encoding/json"
	"common"
	"time"
)

type DATA_TYPE int
const(
	INT_TYPE DATA_TYPE = iota
	STR_TYPE
)
type valueStruct struct {
	Tp int
	Vl interface{}
}

func getFile(fileName string)string{
	return "../tvm/py/test/"+fileName
}

func TestBaseLv1(t *testing.T) {
	Clear()
	code := tvm.Read0(getFile("test_strorage_optimize.py"))
	contract := tvm.Contract{code, "ContractStorage", nil}
	jsonString, _ := json.Marshal(contract)
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")

	for i :=1;i<=2;i++{
		contractAddr := "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
		abi := fmt.Sprintf(`{"FuncName": "setBaseNeedSuccess%d", "Args": []}`,i)
		CallContract(contractAddr, abi)

		contractAddr = "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
		abi = fmt.Sprintf(`{"FuncName": "getBaseNeedSuccess%d", "Args": []}`,i)
		CallContract(contractAddr, abi)
	}


}

func TestStoreDictForMapTypes(t *testing.T) {
	Clear()
	code := tvm.Read0(getFile("test_strorage_optimize.py"))
	contract := tvm.Contract{code, "ContractStorage", nil}
	jsonString, _ := json.Marshal(contract)
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")

	time.Sleep(1* time.Second)
	contractAddr := "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
	abi := `{"FuncName": "setBase", "Args": []}`
	CallContract(contractAddr, abi)

	time.Sleep(1* time.Second)
	contractAddr = "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
	abi = `{"FuncName": "setMap", "Args": []}`
	CallContract(contractAddr, abi)

	//GetContractDatas(contractAddr)
}


func GetContractDatas(contractAddr string)map[string]string{
	addr := common.HexStringToAddress(contractAddr)
	stateDb := BlockChainImpl.LatestStateDB()
	iterator := stateDb.DataIterator(addr, "")
	data := make(map[string]string)
	for iterator != nil {
		if len(iterator.Key) != 0 {
			fmt.Printf("level db key = %s,value=%s \n",string(iterator.Key),string(iterator.Value))
			data[string(iterator.Key)] = string(iterator.Value)
		}
	if !iterator.Next() {
		break
	}
 }
 return data
}

