package tns

import "C"
import (
	"common"
	"fmt"
	"storage/account"
	"tvm"
	"encoding/json"
)

func SetupGenesisContract(stateDB *account.AccountDB) {

	tnsManager :=common.HexStringToAddress("0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd")
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(tnsManager[:], common.Uint64ToByte(uint64(1)))))
	stateDB.SetNonce(tnsManager, 1)
	code := tvm.Read0("./py/tns.py")
	contractData := tvm.Contract{code, "Tns", &contractAddr}
	jsonString, _ := json.Marshal(contractData)
	if len(jsonString) <= 0 {
		return
	}

	fmt.Println("tns contract address: %v", contractAddr)

	stateDB.CreateAccount(contractAddr)
	stateDB.SetCode(contractAddr, jsonString)
	stateDB.SetNonce(contractAddr, 1)

	controller := tvm.NewController(stateDB, nil, nil, nil, common.GlobalConf.GetString("tvm", "pylib", "lib"))
	controller.GasLeft = 1000000

	msg := tvm.Msg{Data: nil, Value: 0, Sender: tnsManager.GetHexString()}

	errorCode, errorMsg := controller.DeployWithMsg(&tnsManager, &contractData, msg)
	if errorCode != 0 {
		fmt.Println("tns contract deploy error: %v", errorMsg)
		return
	}

	//设置地址
	abi := fmt.Sprintf(`{"FuncName": "set_short_account_address", "Args": ["tns", "%v"]}`, contractAddr)
	success, errorMsg := controller.ExecuteAbi(&tnsManager, &contractData, abi, msg)
	if !success  {
		fmt.Println("tns contract set_account_address ExecuteAbi error: %v", errorMsg)
		return
	}


	//设置地址
	abi = fmt.Sprintf(`{"FuncName": "set_short_account_address", "Args": ["tnsmanager", "%v"]}`, contractAddr)
	success, errorMsg = controller.ExecuteAbi(&tnsManager, &contractData, abi, msg)
	if !success  {
		fmt.Println("tns contract set_account_address ExecuteAbi error: %v", errorMsg)
		return
	}

	//设置地址
	abi = fmt.Sprintf(`{"FuncName": "set_short_account_address", "Args": ["node1", "%v"]}`, "0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019")
	success, errorMsg = controller.ExecuteAbi(&tnsManager, &contractData, abi, msg)
	if !success  {
		fmt.Println("tns contract set_account_address ExecuteAbi error: %v", errorMsg)
		return
	}

	//设置地址
	abi = fmt.Sprintf(`{"FuncName": "set_short_account_address", "Args": ["node2", "%v"]}`, "0xd3d410ec7c917f084e0f4b604c7008f01a923676d0352940f68a97264d49fb76")
	success, errorMsg = controller.ExecuteAbi(&tnsManager, &contractData, abi, msg)
	if !success  {
		fmt.Println("tns contract set_account_address ExecuteAbi error: %v", errorMsg)
		return
	}

	//获取account对应的地址
	abi = fmt.Sprintf(`{"FuncName": "get_address", "Args": ["tns"]}`)
	result := controller.ExecuteAbiResult(&tnsManager, &contractData, abi, msg)
	if result != nil  {
		fmt.Println("tns contract get_address: ", result.Content)

	}
	tnsAddr := GetAddressByAccount(stateDB,"tns")
	fmt.Println("tnsGetAddressByAccount: ", tnsAddr)

}

func GetAddressByAccount(stateDB *account.AccountDB ,account string) string{

	tnsManager :=common.HexStringToAddress("0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd")
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(tnsManager[:], common.Uint64ToByte(uint64(1)))))
	contract := tvm.LoadContract(contractAddr)



	controller := tvm.NewController(stateDB, nil, nil, nil, common.GlobalConf.GetString("tvm", "pylib", "lib"))
	controller.GasLeft = 1000000

	msg := tvm.Msg{Data: nil, Value: 0, Sender: ""}

	//获取account对应的地址
	abi := fmt.Sprintf(`{"FuncName": "get_address", "Args": ["%v"]}`, account)
	result := controller.ExecuteAbiResult(&tnsManager, contract, abi, msg)
	if result != nil && result.ResultType ==  2 /*C.RETURN_TYPE_STRING*/ {
		return result.Content
	}
	return ""
}