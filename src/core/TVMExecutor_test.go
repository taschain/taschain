package core

import (
	"encoding/json"
	"middleware/types"
	"common"
	"storage/tasdb"
	"storage/core"
	"testing"
	"fmt"
	"os"
	"time"
	"tvm"
)

func ExampleNewTVMExecutor() {

}

func TestBlockLib(t *testing.T) {
	ChainInit()
	source := "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
	code := `
import block
class A(object):
	def __init__(self):
		if block.blockhash(0) != "0x81107a7a182d1172cdf84b819a827037123557b1b791198207ce198a0e2c0bef":
			raise Exception("blockhash of 0 error")
		if block.blockhash(1) != "0x0000000000000000000000000000000000000000000000000000000000000000":
			raise Exception("blockhash of 1 error")
		if block.coinbase() != "0x0000000000000000000000000000000000000000":
			raise Exception("block.coinbase() error")
		if block.number() != 1:
			raise Exception("block.number() error")
		if 2147483648<block.timestamp() or block.timestamp()<1542615541:
			raise Exception("block.timestamp() error")
		if block.gaslimit() != 200000:
			raise Exception("block.gaslimit() error")
`
	contract := tvm.Contract{code, "A", nil}
	jsonString, _ := json.Marshal(contract)
	// test deploy
	DeployContract(string(jsonString), source, 200000, 0)
}

func TestTxLib(t *testing.T) {
	ChainInit()
	source := "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
	code := `
import tx
class A(object):
	def __init__(self):
		if tx.origin() != "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b":
			raise Exception("tx.origin() error")
		if this != "0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610":
			raise Exception("contract address of this error")
		if msg.value != 1:
			raise Exception("msg.value error")
		if msg.sender != "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b":
			raise Exception("msg.sender error")
		
`
	contract := tvm.Contract{code, "A", nil}
	jsonString, _ := json.Marshal(contract)
	// test deploy
	DeployContract(string(jsonString), source, 200000, 1)
}

func TestCanTransfer(t *testing.T) {
	tt := time.Now()
	ChainInit()
	source := "0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
	code := `
class A(object):
	def __init__(self):
		self.a = 1 
		for i in range(2000000):
			self.a += 1
		
`
	contract := tvm.Contract{code, "A", nil}
	jsonString, _ := json.Marshal(contract)
	// test deploy
	DeployContract(string(jsonString), source, 20000000000000, 1)
	fmt.Println(time.Now().Sub(tt))
}

func Home() string{
	return os.Getenv("HOME")
}

func TestContractCreate(t *testing.T)  {
	block := types.Block{}
	transaction := types.Transaction{}
	addr := common.HexStringToAddress("0x5ed34dd026e1b695224df06fca9c4481649ff29e")
	transaction.Source = &addr
	transaction.Data = []byte("print(\"hello world\")")
	block.Transactions = make([]*types.Transaction,1)
	block.Transactions[0] = &transaction
	executor := TVMExecutor{}
	db, err := tasdb.NewLDBDatabase(Home() + "/TasProject/work/test2", 0, 0)
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()
	triedb := core.NewDatabase(db)
	state, _ := core.NewAccountDB(common.Hash{}, triedb)
	hash, receipts, _ := executor.Execute(state, &block, nil)
	fmt.Println(hash.Hex())
	fmt.Println(receipts[0].ContractAddress.GetHexString())
	root, _ := state.Commit(false)
	fmt.Println(root.Hex())
	triedb.TrieDB().Commit(root, false)
}

func TestContractCall(t *testing.T)  {
	block := types.Block{}
	transaction := types.Transaction{}
	addr := common.HexStringToAddress("0x5ed34dd026e1b695224df06fca9c4481649ff29e")
	transaction.Source = &addr
	transaction.Data = []byte("{}")
	addr = common.HexStringToAddress("0xe8ba89a51b095e63d83f1ec95441483415c64064")
	transaction.Target = &addr
	block.Transactions = make([]*types.Transaction,1)
	block.Transactions[0] = &transaction
	executor := TVMExecutor{}
	db, _ := tasdb.NewLDBDatabase(Home() + "/TasProject/work/test2", 0, 0)
	defer db.Close()
	triedb := core.NewDatabase(db)
	state, err := core.NewAccountDB(common.HexToHash("0xebe99d497383b3f492809715045f0b23324e0b723afd6b1405aa44c2ab6223a0"), triedb)
	if err != nil {
		fmt.Println(err)
	}
	hash, receipts, _ := executor.Execute(state, &block, nil)
	fmt.Println(hash.Hex())
	fmt.Println(receipts[0].ContractAddress.GetHexString())
}
