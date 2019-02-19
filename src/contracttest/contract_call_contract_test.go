package contract_test

import (
	"common"
	"consensus/mediator"
	"consensus/model"
	"core"
	"encoding/json"
	"fmt"
	"middleware"
	"strconv"
	"taslog"
	"testing"
	"tvm"
)

//0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd 公钥对应的私钥
const SK string = "0x04e2dad12f3e49fc3a13044648998ee53654c6ab1e72029bec55e65c06b6136dee9fcf20cf2c12aecce1a2c971fa7251cba6ebc8320f839d687e2ac874ddad7dd0801ce92612e9112dbbe10ce182ba59cb2222758f5f2ce710bf3855ae857e79ba"

func TestContractCallOtherContract(t *testing.T) {
	middleware.InitMiddleware()
	common.InitConf("../../deploy/tvm/tas1.ini")
	common.DefaultLogger = taslog.GetLoggerByIndex(taslog.DefaultConfig, common.GlobalConf.GetString("instance", "index", ""))
	minerInfo := model.NewSelfMinerDO(common.HexToAddress("0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019"))
	core.InitCore(false, mediator.NewConsensusHelper(minerInfo.ID))
	mediator.ConsensusInit(minerInfo)

	mediator.Proc.Start()

	//code := tvm.Read0("../tvm/py/test/contract_becalled.py")
	//contract := tvm.Contract{code, "ContractBeCalled", nil}
	//jsonString, _ := json.Marshal(contract)
	////fmt.Println(string(jsonString))
	//OnChainFunc(string(jsonString), "0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019")

	code := tvm.Read0("../tvm/py/test/contract_call_contract_server.py")
	contract := tvm.Contract{code, "Server", nil}
	jsonString, _ := json.Marshal(contract)
	//fmt.Println(string(jsonString))
	OnChainFunc(string(jsonString), "0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd")
	//0x303e2d65fc7cec255932f6cbbfac69851d47a56d06e3f33dc63c9620e07b1872

	code = tvm.Read0("../tvm/py/test/contract_call_contract_client.py")
	contract = tvm.Contract{code, "Client", nil}
	jsonString, _ = json.Marshal(contract)
	//fmt.Println(string(jsonString))
	OnChainFunc(string(jsonString), "0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd")
	//0xb50aca677104d98b76c87a9576774db7540ff5464494eb5e73452fe658fcc5e5

	contractAddr := "0xb50aca677104d98b76c87a9576774db7540ff5464494eb5e73452fe658fcc5e5"
	abi := `{"FuncName": "client_print", "Args": []}`
	CallContract(contractAddr, abi, "0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd")

	contractAddr = "0xb50aca677104d98b76c87a9576774db7540ff5464494eb5e73452fe658fcc5e5"
	abi = `{"FuncName": "client_print_2", "Args": []}`
	CallContract(contractAddr, abi, "0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd")
}

func TestContractCallContract(t *testing.T) {
	middleware.InitMiddleware()
	common.InitConf("../../deploy/tvm/tas1.ini")
	common.DefaultLogger = taslog.GetLoggerByIndex(taslog.DefaultConfig, common.GlobalConf.GetString("instance", "index", ""))
	minerInfo := model.NewSelfMinerDO(common.HexToAddress("0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019"))

	core.InitCore(false,mediator.NewConsensusHelper(minerInfo.ID))
	mediator.ConsensusInit(minerInfo, common.GlobalConf)


	mediator.Proc.Start()

	//code := tvm.Read0("../tvm/py/test/contract_becalled.py")
	//contract := tvm.Contract{code, "ContractBeCalled", nil}
	//jsonString, _ := json.Marshal(contract)
	////fmt.Println(string(jsonString))
	//OnChainFunc(string(jsonString), "0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019")

	code := tvm.Read0("../tvm/py/test/contract_game.py")
	contract := tvm.Contract{code, "ContractGame", nil}
	jsonString, _ := json.Marshal(contract)
	//fmt.Println(string(jsonString))
	OnChainFunc(string(jsonString), "0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd")

	//int类型
	contractAddr := "0xb50aca677104d98b76c87a9576774db7540ff5464494eb5e73452fe658fcc5e5"
	abi := `{"FuncName": "contract_int", "Args": []}`
	CallContract(contractAddr, abi, "0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd")
	//断言
	result := string(getContractDatas(contractAddr, "a"))
	i, _ := strconv.Atoi(result)
	fmt.Printf("i:%d", i)
	if i != 1121 { //底层非map，默认加了1的前缀
		t.Fatal("call int failed.")
	}
	//
	//// str类型
	//contractAddr = "0x2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb"
	//abi = `{"FuncName": "contract_str", "Args": []}`
	//CallContract(contractAddr, abi)
	//result = string(getContractDatas("0x2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", "mystr"))
	//if result != "1\"myabcbcd\"" {
	//	t.Fatal("call str failed.")
	//}
	//
	//// bool类型
	//contractAddr = "0x2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb"
	//abi = `{"FuncName": "contract_bool", "Args": []}`
	//CallContract(contractAddr, abi)
	//result = string(getContractDatas("0x2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", "mybool"))
	//if result != "1false" {
	//	t.Fatal("call bool failed.")
	//}
	//
	//// none类型
	//contractAddr = "0x2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb"
	//abi = `{"FuncName": "contract_none", "Args": []}`
	//CallContract(contractAddr, abi)
	//result = string(getContractDatas("0x2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", "mynone"))
	//if result != "12" {
	//	t.Fatal("call none failed.")
	//}
}

// 合约深度测试用例，当前运行到第8层没有做控制，底层会有异常。整体控制是ok的，对应的日志也做了处理。
func TestContractMaxLength(t *testing.T) {
	core.BlockChainImpl.Clear()
	//部署合约contract_becalled
	print("\n1\n")
	code := tvm.Read0("../tvm/py/test/contract_becalled.py")
	contract := tvm.Contract{code, "ContractBeCalled", nil}
	jsonString, _ := json.Marshal(contract)
	//fmt.Println(string(jsonString))
	OnChainFunc(string(jsonString), "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")

	//部署合约contract_game
	//time.Sleep(5 * time.Second)
	print("\n2\n")
	code = tvm.Read0("../tvm/py/test/contract_game.py")
	contract = tvm.Contract{code, "ContractGame", nil}
	jsonString, _ = json.Marshal(contract)
	//fmt.Println(string(jsonString))
	OnChainFunc(string(jsonString), "0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103")

	//部署合约contract_becalled_deep
	//time.Sleep(5 * time.Second)
	print("\n3\n")
	code = tvm.Read0("../tvm/py/test/contract_becalled_deep.py")
	contract = tvm.Contract{code, "ContractBeCalledDeep", nil}
	jsonString, _ = json.Marshal(contract)
	//fmt.Println(string(jsonString))
	OnChainFunc(string(jsonString), "0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103")

	//int类型
	//time.Sleep(5 * time.Second)
	print("3")
	contractAddr := "0xe4d60f63188f69980e762cb38aad8727ceb86bbe"
	abi := `{"FuncName": "contract_deep", "Args": []}`
	CallContract(contractAddr, abi, "0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103")

}

func getContractDatas(contractAddr string, keyName string) []byte {
	addr := common.HexStringToAddress(contractAddr)
	stateDb := core.BlockChainImpl.LatestStateDB()
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
