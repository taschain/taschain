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

import (
	"middleware/types"
	"common"
	t "storage/core/types"
	"storage/core"
	"math/big"
	"storage/core/vm"
	"fmt"
	"tvm"
)

type TVMExecutor struct {
	bc     *BlockChain
}

func NewTVMExecutor(bc *BlockChain) *TVMExecutor {
	return &TVMExecutor{
		bc:     bc,
	}
}


func (executor *TVMExecutor) Execute(accountdb *core.AccountDB, block *types.Block, processor VoteProcessor) (common.Hash,[]*t.Receipt,error) {
	if 0 == len(block.Transactions) {
		hash := accountdb.IntermediateRoot(true)
		Logger.Infof("TVMExecutor Execute Hash:%s",hash.Hex())
		return hash, nil, nil
	}
	receipts := make([]*t.Receipt,len(block.Transactions))
	msg := tvm.Msg{}
	for i,transaction := range block.Transactions{
		var fail = false
		var contractAddress common.Address
		if transaction.Target == nil{
			tvm.EnvInit(accountdb, BlockChainImpl, block.Header, transaction)
			contractAddress, _ = createContract(accountdb, transaction)
			contract := tvm.LoadContract(contractAddress)
			vm := tvm.NewTvm(transaction.Source, contract, common.GlobalConf.GetString("tvm", "pylib", "lib"))

			msg = tvm.Msg{Data:[]byte{}, Value:transaction.Value, Sender: transaction.Source.GetHexString()}
			if !vm.Deploy(msg) || !vm.StoreData(){

			}
			vm.DelTvm()
		} else if len(transaction.Data) > 0 {
			tvm.EnvInit(accountdb, BlockChainImpl, block.Header, transaction)
			contract := tvm.LoadContract(*transaction.Target)
			vm := tvm.NewTvm(transaction.Source, contract, common.GlobalConf.GetString("tvm", "pylib", "lib"))
			snapshot := accountdb.Snapshot()
			msg = tvm.Msg{Data:transaction.Data, Value:transaction.Value, Sender: transaction.Source.GetHexString()}
			fail = !vm.LoadContractCode() || !vm.ExecuteABIJson(msg, string(transaction.Data))
			if !fail {
				fail = !vm.StoreData()
			}
			if fail {
				accountdb.RevertToSnapshot(snapshot)
			}
			block := vm.Block
			vm.DelTvm()
			if !fail {
				block()
			}
		} else {
			amount := big.NewInt(int64(transaction.Value))
			if CanTransfer(accountdb, *transaction.Source, amount){
				Transfer(accountdb, *transaction.Source, *transaction.Target, amount)
			} else {
				fail = true
			}
		}
		receipt := t.NewReceipt(nil,fail,0)
		receipt.TxHash = transaction.Hash
		receipt.ContractAddress = contractAddress
		receipts[i] = receipt
	}

	//if nil != processor {
	//	processor.AfterAllTransactionExecuted(block, statedb, receipts)
	//}

	return accountdb.IntermediateRoot(true), receipts, nil
}

func createContract(accountdb *core.AccountDB, transaction *types.Transaction) (common.Address, error) {
	amount := big.NewInt(int64(transaction.Value))
	if !CanTransfer(accountdb, *transaction.Source, amount){
		return common.Address{}, fmt.Errorf("balance not enough")
	}

	nance := accountdb.GetNonce(*transaction.Source)
	accountdb.SetNonce(*transaction.Source, nance + 1)
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(transaction.Source[:], common.Uint64ToByte(nance))))

	if accountdb.GetCodeHash(contractAddr) != (common.Hash{}){
		return common.Address{}, fmt.Errorf("contract address conflict")
	}
	accountdb.CreateAccount(contractAddr)
	accountdb.SetCode(contractAddr, transaction.Data)
	accountdb.SetNonce(contractAddr, 1)

	Transfer(accountdb, *transaction.Source, contractAddr, amount)
	return contractAddr, nil
}

func CanTransfer(db vm.AccountDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

func Transfer(db vm.AccountDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}
