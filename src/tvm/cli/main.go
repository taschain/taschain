package main

import "C"
import (
	"common"
	"encoding/json"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"storage/account"
	"storage/tasdb"
	"tvm"
)


var (
	app = kingpin.New("chat", "A command-line chat application.")

	deployContract = app.Command("deploy", "deploy contract.")
	contractName = deployContract.Arg("name", "").Required().String()
	contractPath = deployContract.Arg("path", "").Required().String()

	callContract = app.Command("call", "call contract.")
	contractAbi = callContract.Arg("abiPath", "").Required().String()


)

type Transaction struct {
	tvm.ControllerTransactionInterface
}

func (Transaction) GetGasLimit() uint64 {return 500000}
func (Transaction) GetValue() uint64    {return 0}
func (Transaction) GetSource() *common.Address {
	address := common.StringToAddress("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103")
	return &address
}
func (Transaction) GetTarget() *common.Address {
	address := common.StringToAddress("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103")
	return &address
}
func (Transaction) GetData() []byte      {return nil}
func (Transaction) GetHash() common.Hash {return common.Hash{}}


func Exists(path string) bool {
	_, err := os.Stat(path)    //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func main() {

	defaultAccounts := [...]string{"0x6c63b15aac9b94927681f5fb1a7343888dece14e3160b3633baa9e0d540228cd",
		"0x3eed3f4a15d238dc2ab658dcaa069a7d072437c9c86e1605ce74cd9f4730bbf2",
		"0x36ae29871aed1bc21e708c4e2f5ff7c03218f5ffcd3eeae31d94a2985143abd7",
		"0xf798010011a0f17510ce4fdea9b3e7b458392b4bb8205ead3eb818609e93746c",
		"0xcd54640ff11b6ffe601566008872c87a4f3ec01a2890404b6ce30905ee3b2137"}

	currentPath, error := filepath.Abs(filepath.Dir(os.Args[0]))
	if error != nil {
		fmt.Println(error)
		return
	}
	fmt.Println(currentPath)
	db, _ := tasdb.NewLDBDatabase(currentPath + "/db", 0, 0)
	defer db.Close()
	database := account.NewDatabase(db)

	var settings common.ConfManager
	if Exists(currentPath + "/settings.ini") {
		settings = common.NewConfINIManager(currentPath + "/settings.ini")
		//stateHash := settings.GetString("root", "StateHash", "")
		//state, error := account.NewAccountDB(common.HexToHash(stateHash), database)
		//if error != nil {
		//	fmt.Println(error)
		//	return
		//}
		//fmt.Println(stateHash)
		//fmt.Println(state.GetBalance(common.StringToAddress(defaultAccounts[0])))
	} else {
		settings = common.NewConfINIManager(currentPath + "/settings.ini")
		state, _ := account.NewAccountDB(common.Hash{}, database)
		for i := 0; i < len(defaultAccounts); i++ {
			accountAddress := common.StringToAddress(defaultAccounts[i])
			state.SetBalance(accountAddress, big.NewInt(200))
		}
		hash, error := state.Commit(false)
		database.TrieDB().Commit(hash, false)
		if error != nil {
			fmt.Println(error)
			return
		} else {
			settings.SetString("root", "StateHash", hash.String())
			fmt.Println(hash.String())
		}
	}


	switch kingpin.MustParse(app.Parse(os.Args[1:])) {

	// deploy Token ./cli/erc20.py
	case deployContract.FullCommand():
		stateHash := settings.GetString("root", "StateHash", "")
		state, _ := account.NewAccountDB(common.HexToHash(stateHash), database)
		controller := tvm.NewController(state, nil, nil, Transaction{}, "../py")

		f, err := ioutil.ReadFile(filepath.Dir(os.Args[0]) + "/" + *contractPath) //读取文件
		if err != nil {
			fmt.Println("read the " + *contractPath + " file failed ", err)
			return
		}
		contractAddress := common.HexToAddress("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103")

		contract := tvm.Contract{
			ContractName: *contractName,
			Code: string(f),
			//ContractAddress: &contractAddress,
		}

		jsonBytes, errMarsh := json.Marshal(contract)
		if errMarsh != nil {
			fmt.Println(errMarsh)
			return
		}
		state.CreateAccount(contractAddress)
		state.SetCode(contractAddress, jsonBytes)

		contract.ContractAddress = &contractAddress
		controller.Deploy(&contract)

		hash, error := state.Commit(false)
		database.TrieDB().Commit(hash, false)
		if error != nil {
			fmt.Println(error)
		}
		settings.SetString("root", "StateHash", hash.String())
		fmt.Println(hash.String())

	// call ./cli/call_Token_abi.json
	case callContract.FullCommand():

		stateHash := settings.GetString("root", "StateHash", "")
		state, _ := account.NewAccountDB(common.HexToHash(stateHash), database)

		controller := tvm.NewController(state, nil, nil, Transaction{}, "../py")

		f, err := ioutil.ReadFile(filepath.Dir(os.Args[0]) + "/" + *contractAbi) //读取文件
		if err != nil {
			fmt.Println("read the " + *contractAbi + " file failed ", err)
			return
		}
		abi := tvm.ABI{}
		abiJsonError := json.Unmarshal(f, &abi)
		if abiJsonError!= nil{
			fmt.Println(*contractAbi + " json.Unmarshal failed ", err)
			return
		}
		contractAddress := common.HexToAddress("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103")
		contract := tvm.LoadContract(contractAddress)
		//fmt.Println(contract.Code)
		sender := common.HexToAddress(defaultAccounts[0])
		executeResult := controller.ExecuteAbiEval(&sender, contract, string(f))
		//TODO 需不需要快照
		if executeResult == nil {
			fmt.Println("ExecuteAbiEval error")
			return
		} else if executeResult.ResultType == 4 /*C.RETURN_TYPE_EXCEPTION*/ {
			fmt.Println("error code: ", executeResult.ErrorCode, " error info: ", executeResult.Content)
		} else {
			fmt.Println("executeResult: ", executeResult.Content)
		}

		hash, error := state.Commit(false)
		database.TrieDB().Commit(hash, false)
		if error != nil {
			fmt.Println(error)
		}
		settings.SetString("root", "StateHash", hash.String())
		fmt.Println(hash.String())
	}

}