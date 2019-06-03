package main

import (
	"encoding/json"
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/storage/account"
	"github.com/taschain/taschain/storage/tasdb"
	"github.com/taschain/taschain/tvm"
	"math/big"
	"os"
	"path/filepath"
)

type Transaction struct {
	tvm.ControllerTransactionInterface
}

func (Transaction) GetGasLimit() uint64 { return 500000 }
func (Transaction) GetValue() uint64    { return 0 }
func (Transaction) GetSource() *common.Address {
	address := common.HexToAddress("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103")
	return &address
}
func (Transaction) GetTarget() *common.Address {
	address := common.HexToAddress("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103")
	return &address
}
func (Transaction) GetData() []byte      { return nil }
func (Transaction) GetHash() common.Hash { return common.Hash{} }

func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

var (
	DefaultAccounts = [...]string{"0x6c63b15aac9b94927681f5fb1a7343888dece14e3160b3633baa9e0d540228cd",
		"0x3eed3f4a15d238dc2ab658dcaa069a7d072437c9c86e1605ce74cd9f4730bbf2",
		"0x36ae29871aed1bc21e708c4e2f5ff7c03218f5ffcd3eeae31d94a2985143abd7",
		"0xf798010011a0f17510ce4fdea9b3e7b458392b4bb8205ead3eb818609e93746c",
		"0xcd54640ff11b6ffe601566008872c87a4f3ec01a2890404b6ce30905ee3b2137"}
)

type TvmCli struct {
	settings common.ConfManager
	db       *tasdb.LDBDatabase
	database account.AccountDatabase
}

func NewTvmCli() *TvmCli {
	tvmCli := new(TvmCli)
	tvmCli.init()
	return tvmCli
}

func (t *TvmCli) DeleteTvmCli() {
	defer t.db.Close()
}

func (t *TvmCli) init() {

	currentPath, error := filepath.Abs(filepath.Dir(os.Args[0]))
	if error != nil {
		fmt.Println(error)
		return
	}
	fmt.Println(currentPath)
	t.db, _ = tasdb.NewLDBDatabase(currentPath+"/db", 0, 0)
	t.database = account.NewDatabase(t.db)

	if Exists(currentPath + "/settings.ini") {
		t.settings = common.NewConfINIManager(currentPath + "/settings.ini")
		//stateHash := settings.GetString("root", "StateHash", "")
		//state, error := account.NewAccountDB(common.HexToHash(stateHash), database)
		//if error != nil {
		//	fmt.Println(error)
		//	return
		//}
		//fmt.Println(stateHash)
		//fmt.Println(state.GetBalance(common.StringToAddress(defaultAccounts[0])))
	} else {
		t.settings = common.NewConfINIManager(currentPath + "/settings.ini")
		state, _ := account.NewAccountDB(common.Hash{}, t.database)
		for i := 0; i < len(DefaultAccounts); i++ {
			accountAddress := common.HexToAddress(DefaultAccounts[i])
			state.SetBalance(accountAddress, big.NewInt(200))
		}
		hash, error := state.Commit(false)
		t.database.TrieDB().Commit(hash, false)
		if error != nil {
			fmt.Println(error)
			return
		} else {
			t.settings.SetString("root", "StateHash", hash.Hex())
			fmt.Println(hash.Hex())
		}
	}
}

func (t *TvmCli) Deploy(contractName string, contractCode string) string {
	stateHash := t.settings.GetString("root", "StateHash", "")
	state, _ := account.NewAccountDB(common.HexToHash(stateHash), t.database)
	transaction := Transaction{}
	controller := tvm.NewController(state, nil, nil, transaction, 0, "../py", nil, nil)

	nonce := state.GetNonce(*transaction.GetSource())
	contractAddress := common.BytesToAddress(common.Sha256(common.BytesCombine(transaction.GetSource()[:], common.Uint64ToByte(nonce))))
	fmt.Println("contractAddress: ", contractAddress.Hex())
	state.SetNonce(*transaction.GetSource(), nonce+1)

	contract := tvm.Contract{
		ContractName: contractName,
		Code:         contractCode,
		//ContractAddress: &contractAddress,
	}

	jsonBytes, errMarsh := json.Marshal(contract)
	if errMarsh != nil {
		fmt.Println(errMarsh)
		return ""
	}
	state.CreateAccount(contractAddress)
	state.SetCode(contractAddress, jsonBytes)

	contract.ContractAddress = &contractAddress
	controller.VM.SetGas(500000)
	controller.Deploy(&contract)
	fmt.Println("gas: ", 500000-controller.VM.Gas())

	hash, error := state.Commit(false)
	t.database.TrieDB().Commit(hash, false)
	if error != nil {
		fmt.Println(error)
	}
	t.settings.SetString("root", "StateHash", hash.Hex())
	fmt.Println(hash.Hex())
	return contractAddress.Hex()
}

func (t *TvmCli) Call(contractAddress string, abiJSON string) {
	stateHash := t.settings.GetString("root", "StateHash", "")
	state, _ := account.NewAccountDB(common.HexToHash(stateHash), t.database)

	controller := tvm.NewController(state, nil, nil, Transaction{}, 0, "../py", nil, nil)

	//abi := tvm.ABI{}
	//abiJsonError := json.Unmarshal([]byte(abiJSON), &abi)
	//if abiJsonError != nil{
	//	fmt.Println(abiJSON, " json.Unmarshal failed ", abiJsonError)
	//	return
	//}
	_contractAddress := common.HexToAddress(contractAddress)
	contract := tvm.LoadContract(_contractAddress)
	//fmt.Println(contract.Code)
	sender := common.HexToAddress(DefaultAccounts[0])
	controller.VM.SetGas(500000)
	executeResult := controller.ExecuteAbiEval(&sender, contract, abiJSON)
	fmt.Println("gas: ", 500000-controller.VM.Gas())

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
	t.database.TrieDB().Commit(hash, false)
	if error != nil {
		fmt.Println(error)
	}
	t.settings.SetString("root", "StateHash", hash.Hex())
	fmt.Println(hash.Hex())
}

func (t *TvmCli) ExportAbi(contractName string, contractCode string) {
	contract := tvm.Contract{
		ContractName: contractName,
		//Code: contractCode,
		//ContractAddress: &contractAddress,
	}
	vm := tvm.NewTvm(nil, &contract, "../py")
	defer func() {
		vm.DelTvm()
	}()
	str := `
class Register(object):
    def __init__(self):
        self.funcinfo = {}
        self.abiinfo = []

    def public(self , *dargs):
        def wrapper(func):
            paranametuple = func.__para__
            paraname = list(paranametuple)
            paraname.remove("self")
            paratype = []
            for i in range(len(paraname)):
                paratype.append(dargs[i])
            self.funcinfo[func.__name__] = [paraname,paratype]
            tmp = {}
            tmp["FuncName"] = func.__name__
            tmp["Args"] = paratype
            self.abiinfo.append(tmp)
            abiexport(str(self.abiinfo))

            def _wrapper(*args , **kargs):
                return func(*args, **kargs)
            return _wrapper
        return wrapper

import builtins
builtins.register = Register()
`

	errorCode, errorMsg := vm.ExecutedScriptVMSucceed(str)
	if errorCode == types.Success {
		result := vm.ExecutedScriptKindFile(contractCode)
		fmt.Println(result.Abi)
	} else {
		fmt.Println(errorMsg)
	}

}
