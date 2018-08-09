package core

import (
	"middleware/types"
	"common"
	"storage/tasdb"
	"storage/core"
	"testing"
	"fmt"
)

func TestContractCreate(t *testing.T)  {
	block := types.Block{}
	transaction := types.Transaction{}
	addr := common.HexStringToAddress("0x5ed34dd026e1b695224df06fca9c4481649ff29e")
	transaction.Source = &addr
	transaction.Data = []byte("'print(\"hello world\")'")
	block.Transactions = make([]*types.Transaction,1)
	block.Transactions[0] = &transaction
	executor := TVMExecutor{}
	db, _ := tasdb.NewLDBDatabase("/Users/Kaede/TasProject/work/test2", 0, 0)
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
	db, _ := tasdb.NewLDBDatabase("/Users/Kaede/TasProject/work/test2", 0, 0)
	defer db.Close()
	triedb := core.NewDatabase(db)
	state, _ := core.NewAccountDB(common.HexToHash("0xcf545b9496a1665285aa385d9ee5542154f2fb4dcefc820b4ccb00741b88c9ed"), triedb)
	hash, receipts, _ := executor.Execute(state, &block, nil)
	fmt.Println(hash.Hex())
	fmt.Println(receipts[0].ContractAddress.GetHexString())
}
