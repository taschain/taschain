//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

//
//import (
//	"middleware/types"
//	"common"
//	"storage/tasdb"
//	"storage/core"
//	"testing"
//	"fmt"
//	"os"
//)
//
//func ExampleNewTVMExecutor() {
//
//}
//
//func TestContract(t *testing.T) {
//	scripts := []string{
//		`import account
//account.create_account("0x1234")
//if account.get_balance("0x1234") != 0:
//	raise Exception("get_balance error")
//account.add_balance("0x1234",1000000000000000000)
//if account.get_balance("0x1234") != 1000000000000000000:
//	raise Exception("get_balance error")
//account.sub_balance("0x1234",1)
//if account.get_balance("0x1234") != 999999999999999999:
//	raise Exception("get_balance error")
//account.set_nonce("0x1234", 10)
//if account.get_nonce("0x1234") != 10:
//	raise Exception("get_nonce error")
//account.set_code("0xe8ba89a51b095e63d83f1ec95441483415c64065", "print('hello world')")
//`,
//`
//import account
//code_hash = account.get_code_hash("0xe8ba89a51b095e63d83f1ec95441483415c64065")
//code = account.get_code("0xe8ba89a51b095e63d83f1ec95441483415c64065")
//assert code == "print('hello world')"
//size = account.get_code_size("0xe8ba89a51b095e63d83f1ec95441483415c64065")
//assert size == 20
//account.add_refund(10)
//account.add_refund(5)
//refund = account.get_refund()
//assert refund == 15
//account.set_data("0xe8ba89a51b095e63d83f1ec95441483415c64066", "test", "right")
//assert account.get_data("0xe8ba89a51b095e63d83f1ec95441483415c64066", "test") == "right"
//before = account.has_suicided("0xe8ba89a51b095e63d83f1ec95441483415c64066")
//account.suicide("0xe8ba89a51b095e63d83f1ec95441483415c64066")
//after = account.has_suicided("0xe8ba89a51b095e63d83f1ec95441483415c64066")
//assert before != after
//assert account.exists("0xe8ba89a51b095e63d83f1ec95441483415c64066") != False
//assert account.exists("0xe8ba89a51b095e63d83f1ec95441483415c64000") != True
//account.create_account("0x123456")
//num = account.snapshot()
//account.add_balance("0x123456",100)
//assert account.get_balance("0x123456") == 100
//account.revert_to_snapshot(num)
//assert account.get_balance("0x123456") == 0
//`,
//	}
//	block := types.Block{}
//	block.Transactions = make([]*types.Transaction, 0)
//	for _, script := range scripts{
//		transaction := types.Transaction{}
//		addr := common.HexStringToAddress("0x5ed34dd026e1b695224df06fca9c4481649ff29e")
//		transaction.Source = &addr
//		transaction.Data = []byte(script)
//		block.Transactions = append(block.Transactions, &transaction)
//	}
//	executor := TVMExecutor{}
//	db, err := tasdb.NewLDBDatabase(Home() + "/TasProject/work/test2", 0, 0)
//	if err != nil {
//		fmt.Println(err)
//	}
//	defer db.Close()
//	triedb := core.NewDatabase(db)
//	state, _ := core.NewAccountDB(common.Hash{}, triedb)
//	_, receipts, _ := executor.Execute(state, &block, nil)
//	//fmt.Println(hash.Hex())
//	//fmt.Println(receipts[0].ContractAddress.GetHexString())
//	root, _ := state.Commit(false)
//	//fmt.Println(root.Hex())
//	triedb.TrieDB().Commit(root, false)
//
//	block = types.Block{}
//	block.Transactions = make([]*types.Transaction,0)
//	for _, receipt := range receipts{
//		fmt.Println(receipt.ContractAddress.GetHexString())
//		transaction := types.Transaction{}
//		addr := common.HexStringToAddress("0x5ed34dd026e1b695224df06fca9c4481649ff29e")
//		transaction.Source = &addr
//		transaction.Data = []byte("{}")
//		addr = receipt.ContractAddress
//		transaction.Target = &addr
//		block.Transactions = append(block.Transactions, &transaction)
//	}
//	executor = TVMExecutor{}
//	triedb = core.NewDatabase(db)
//	state, err = core.NewAccountDB(common.HexToHash(root.Hex()), triedb)
//	if err != nil {
//		fmt.Println(err)
//	}
//	_, receipts, _ = executor.Execute(state, &block, nil)
//}
//
//func Home() string{
//	return os.Getenv("HOME")
//}
//
//func TestContractCreate(t *testing.T)  {
//	block := types.Block{}
//	transaction := types.Transaction{}
//	addr := common.HexStringToAddress("0x5ed34dd026e1b695224df06fca9c4481649ff29e")
//	transaction.Source = &addr
//	transaction.Data = []byte("print(\"hello world\")")
//	block.Transactions = make([]*types.Transaction,1)
//	block.Transactions[0] = &transaction
//	executor := TVMExecutor{}
//	db, err := tasdb.NewLDBDatabase(Home() + "/TasProject/work/test2", 0, 0)
//	if err != nil {
//		fmt.Println(err)
//	}
//	defer db.Close()
//	triedb := core.NewDatabase(db)
//	state, _ := core.NewAccountDB(common.Hash{}, triedb)
//	hash, receipts, _ := executor.Execute(state, &block, nil)
//	fmt.Println(hash.Hex())
//	fmt.Println(receipts[0].ContractAddress.GetHexString())
//	root, _ := state.Commit(false)
//	fmt.Println(root.Hex())
//	triedb.TrieDB().Commit(root, false)
//}
//
//func TestContractCall(t *testing.T)  {
//	block := types.Block{}
//	transaction := types.Transaction{}
//	addr := common.HexStringToAddress("0x5ed34dd026e1b695224df06fca9c4481649ff29e")
//	transaction.Source = &addr
//	transaction.Data = []byte("{}")
//	addr = common.HexStringToAddress("0xe8ba89a51b095e63d83f1ec95441483415c64064")
//	transaction.Target = &addr
//	block.Transactions = make([]*types.Transaction,1)
//	block.Transactions[0] = &transaction
//	executor := TVMExecutor{}
//	db, _ := tasdb.NewLDBDatabase(Home() + "/TasProject/work/test2", 0, 0)
//	defer db.Close()
//	triedb := core.NewDatabase(db)
//	state, err := core.NewAccountDB(common.HexToHash("0xebe99d497383b3f492809715045f0b23324e0b723afd6b1405aa44c2ab6223a0"), triedb)
//	if err != nil {
//		fmt.Println(err)
//	}
//	hash, receipts, _ := executor.Execute(state, &block, nil)
//	fmt.Println(hash.Hex())
//	fmt.Println(receipts[0].ContractAddress.GetHexString())
//}
