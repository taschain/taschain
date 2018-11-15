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

func TestBaseTypes(t *testing.T) {
	Clear()
	code := tvm.Read0(getFile("test_strorage_optimize.py"))
	contract := tvm.Contract{code, "ContractStorage", nil}
	jsonString, _ := json.Marshal(contract)
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
	for i :=1;i<=4;i++{
		contractAddr := "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
		abi := fmt.Sprintf(`{"FuncName": "setBaseNeedSuccess%d", "Args": []}`,i)
		CallContract(contractAddr, abi)
		time.Sleep(2*time.Second)
		funcStr := fmt.Sprintf("setBaseNeedSuccess%d",i)
		if !hasValue(contractAddr,funcStr){
			t.Fatal("storage failed.")
		}

		contractAddr = "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
		abi = fmt.Sprintf(`{"FuncName": "getBaseNeedSuccess%d", "Args": []}`,i)
		CallContract(contractAddr, abi)
		time.Sleep(2*time.Second)
		funcStr = fmt.Sprintf("getBaseNeedSuccess%d",i)
		if !hasValue(contractAddr,funcStr){
			t.Fatal("storage failed.")
		}
	}

	contractAddr := "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
	abi := `{"FuncName": "setChangeKey", "Args": []}`
	CallContract(contractAddr, abi)
	time.Sleep(2*time.Second)

	funcStr := "setChangeKey"
	if !hasValue(contractAddr,funcStr){
		t.Fatal("storage failed.")
	}

	contractAddr = "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
	abi = `{"FuncName": "getChangeKey", "Args": []}`
	CallContract(contractAddr, abi)
	time.Sleep(2*time.Second)
	funcStr = "getChangeKey"
	if !hasValue(contractAddr,funcStr){
		t.Fatal("storage failed.")
	}
}

func TestBaseTypeErrors(t *testing.T) {
	Clear()
	code := tvm.Read0(getFile("test_strorage_optimize.py"))
	contract := tvm.Contract{code, "ContractStorage", nil}
	jsonString, _ := json.Marshal(contract)
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")

	contractAddr := "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
	abi := `{"FuncName": "baseErrors", "Args": []}`
	CallContract(contractAddr, abi)

	time.Sleep(2*time.Second)
	funcStr := "baseErrors"
	if !hasValue(contractAddr,funcStr){
		t.Fatal("storage failed.")
	}
}


func TestMapBaseNeedSuccess1(t *testing.T) {
	time.Sleep(2*time.Second)
	Clear()
	code := tvm.Read0(getFile("test_strorage_map_optimize.py"))
	contract := tvm.Contract{code, "ContractMapStorage", nil}
	jsonString, _ := json.Marshal(contract)
	OnChainFunc(string(jsonString), "0x8b9b5d03019c07d8b6c51f90da3a666eec13ab35")
	time.Sleep(2*time.Second)
	contractAddr := "0x263d21332a876bafce5dc1258c13479eb1e7bf87"
	abi := `{"FuncName": "setMapBaseDataSetNeedSuccess", "Args": []}`
	CallContract(contractAddr, abi)
	time.Sleep(2*time.Second)
	funcStr := "setMapBaseDataSetNeedSuccess"
	if !hasValue(contractAddr,funcStr){
		t.Fatal("storage failed.")
	}
	abi = `{"FuncName": "getMapBaseDataSetNeedSuccess", "Args": []}`
	CallContract(contractAddr, abi)
	time.Sleep(2*time.Second)
	funcStr = "getMapBaseDataSetNeedSuccess"
	if !hasValue(contractAddr,funcStr){
		t.Fatal("storage failed.")
	}
	abi = `{"FuncName": "getMapBaseDataSetNeedSuccess2", "Args": []}`
	CallContract(contractAddr, abi)
	time.Sleep(2*time.Second)
	funcStr = "getMapBaseDataSetNeedSuccess2"
	if !hasValue(contractAddr,funcStr){
		t.Fatal("storage failed.")
	}
}

func TestMapCoverValue(t *testing.T) {
	time.Sleep(2*time.Second)
	Clear()
	code := tvm.Read0(getFile("test_strorage_map_optimize.py"))
	contract := tvm.Contract{code, "ContractMapStorage", nil}
	jsonString, _ := json.Marshal(contract)
	OnChainFunc(string(jsonString), "0x8b9b5d03019c07d8b6c51f90da3a666eec13ab35")
	time.Sleep(2*time.Second)
	contractAddr := "0x263d21332a876bafce5dc1258c13479eb1e7bf87"
	abi := `{"FuncName": "setMapCoverValue", "Args": []}`
	CallContract(contractAddr, abi)
	time.Sleep(2*time.Second)
	funcStr := "setMapCoverValue"
	if !hasValue(contractAddr,funcStr){
		t.Fatal("storage failed.")
	}
	abi = `{"FuncName": "getMapCoverValue", "Args": []}`
	CallContract(contractAddr, abi)
	time.Sleep(2*time.Second)
	funcStr = "getMapCoverValue"
	if !hasValue(contractAddr,funcStr){
		t.Fatal("storage failed.")
	}
}


func TestMapNestIn(t *testing.T) {
	time.Sleep(2*time.Second)
	Clear()
	code := tvm.Read0(getFile("test_strorage_map_optimize.py"))
	contract := tvm.Contract{code, "ContractMapStorage", nil}
	jsonString, _ := json.Marshal(contract)
	OnChainFunc(string(jsonString), "0x8b9b5d03019c07d8b6c51f90da3a666eec13ab35")
	time.Sleep(2*time.Second)
	contractAddr := "0x263d21332a876bafce5dc1258c13479eb1e7bf87"
	abi := `{"FuncName": "setMapNestIn", "Args": []}`
	CallContract(contractAddr, abi)
	time.Sleep(2*time.Second)
	funcStr := "setMapNestIn"
	if !hasValue(contractAddr,funcStr){
		t.Fatal("storage failed.")
	}
	abi = `{"FuncName": "getMapNestIn", "Args": []}`
	CallContract(contractAddr, abi)
	time.Sleep(2*time.Second)
	funcStr = "getMapNestIn"
	if !hasValue(contractAddr,funcStr){
		t.Fatal("storage failed.")
	}
	abi = `{"FuncName": "getMapNestIn2", "Args": []}`
	CallContract(contractAddr, abi)
	time.Sleep(2*time.Second)
	funcStr = "getMapNestIn2"
	if !hasValue(contractAddr,funcStr){
		t.Fatal("storage failed.")
	}
}


//func TestMapIter(t *testing.T) {
//	Clear()
//	code := tvm.Read0(getFile("test_strorage_map_optimize.py"))
//	contract := tvm.Contract{code, "ContractMapStorage", nil}
//	jsonString, _ := json.Marshal(contract)
//	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
//
//	contractAddr := "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
//	abi := `{"FuncName": "setMapIter", "Args": []}`
//	CallContract(contractAddr, abi)
//
//	contractAddr = "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610"
//	abi = `{"FuncName": "getMapIter", "Args": []}`
//	CallContract(contractAddr, abi)
//
//}

func TestMapErrors(t *testing.T) {
	time.Sleep(2*time.Second)
	Clear()
	code := tvm.Read0(getFile("test_strorage_map_optimize.py"))
	contract := tvm.Contract{code, "ContractMapStorage", nil}
	jsonString, _ := json.Marshal(contract)
	OnChainFunc(string(jsonString), "0x8b9b5d03019c07d8b6c51f90da3a666eec13ab35")
	time.Sleep(2*time.Second)
	contractAddr := "0x263d21332a876bafce5dc1258c13479eb1e7bf87"
	abi := `{"FuncName": "setMapErrors", "Args": []}`
	CallContract(contractAddr, abi)
	time.Sleep(2*time.Second)
	funcStr := "setMapErrors"
	if !hasValue(contractAddr,funcStr){
		t.Fatal("storage failed.")
	}
}

func TestMapNone(t *testing.T) {
	time.Sleep(2*time.Second)
	Clear()
	code := tvm.Read0(getFile("test_strorage_map_optimize.py"))
	contract := tvm.Contract{code, "ContractMapStorage", nil}
	jsonString, _ := json.Marshal(contract)
	OnChainFunc(string(jsonString), "0x8b9b5d03019c07d8b6c51f90da3a666eec13ab35")
	time.Sleep(2*time.Second)
	contractAddr := "0x263d21332a876bafce5dc1258c13479eb1e7bf87"
	abi := `{"FuncName": "setNull", "Args": []}`
	CallContract(contractAddr, abi)
	time.Sleep(2*time.Second)
	funcStr := "setNull"
	if !hasValue(contractAddr,funcStr){
		t.Fatal("storage failed.")
	}
	abi = `{"FuncName": "getNull1", "Args": []}`
	CallContract(contractAddr, abi)
	time.Sleep(2*time.Second)
	funcStr = "getNull1"
	if !hasValue(contractAddr,funcStr){
		t.Fatal("storage failed.")
	}
	abi = `{"FuncName": "getNull2", "Args": []}`
	CallContract(contractAddr, abi)
	time.Sleep(2*time.Second)
	funcStr = "getNull2"
	if !hasValue(contractAddr,funcStr){
		t.Fatal("storage failed.")
	}
}


//func TestFomo3d(t *testing.T) {
//	time.Sleep(2*time.Second)
//	Clear()
//	code := tvm.Read0(getFile("fomo3d.py"))
//	contract := tvm.Contract{code, "Fomo3D", nil}
//	jsonString, _ := json.Marshal(contract)
//	OnChainFunc(string(jsonString), "0xdecfe3ad16b72230967de01f640b7e4729b49fce")
//	time.Sleep(2*time.Second)
//	contractAddr := "0xf812cd00f26a27e83bfee0592567e4cf99cc8d7d"
//	abi := `{"FuncName": "buy", "Args": [10,0,0]}`
//	CallContract(contractAddr, abi)
//	time.Sleep(2*time.Second)
//	abi = `{"FuncName": "withdraw", "Args": []}`
//	CallContract(contractAddr, abi)
//
//}


func hasValue(address string,key string)bool{
	datas := GetContractDatas(address)
	if datas[key] !=  `1"success"`{
		return false
	}
	return true
}

func GetContractDatas(contractAddr string)map[string]string{
	addr := common.HexStringToAddress(contractAddr)
	stateDb := BlockChainImpl.LatestStateDB()
	iterator := stateDb.DataIterator(addr, "")
	data := make(map[string]string)
	for iterator != nil {
		if len(iterator.Key) != 0 {
			data[string(iterator.Key)] = string(iterator.Value)
		}
	if !iterator.Next() {
		break
	}
 }
 return data
}

