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
	"bytes"
	"github.com/vmihailenco/msgpack"
)

var castorReward = big.NewInt(50)
var bonusReward = big.NewInt(20)

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
		hash := accountdb.IntermediateRoot(false)
		Logger.Infof("TVMExecutor Execute Empty State:%s",hash.Hex())
		return hash, nil, nil
	}
	receipts := make([]*t.Receipt,len(block.Transactions))
	Logger.Debugf("TVMExecutor Begin Execute State %s",block.Header.StateTree.Hex())
	for i,transaction := range block.Transactions{
		var fail = false
		var contractAddress common.Address
		//Logger.Debugf("TVMExecutor Execute %+v",transaction)

		switch transaction.Type {
			case types.TransactionTypeTransfer:
				amount := big.NewInt(int64(transaction.Value))
				if CanTransfer(accountdb, *transaction.Source, amount){
					Transfer(accountdb, *transaction.Source, *transaction.Target, amount)
					Logger.Debugf("TVMExecutor Execute Transfer Transaction %s",transaction.Hash.Hex())
				} else {
					fail = true
				}
			case types.TransactionTypeContractCreate:
				controller := tvm.NewController(accountdb, BlockChainImpl, block.Header, transaction, common.GlobalConf.GetString("tvm", "pylib", "lib"))
				contractAddress, _ = createContract(accountdb, transaction)
				contract := tvm.LoadContract(contractAddress)
				fail = !controller.Deploy(transaction.Source, contract)
				Logger.Debugf("TVMExecutor Execute ContractCreate Transaction %s",transaction.Hash.Hex())
			case types.TransactionTypeContractCall:
				controller := tvm.NewController(accountdb, BlockChainImpl, block.Header, transaction, common.GlobalConf.GetString("tvm", "pylib", "lib"))
				contract := tvm.LoadContract(*transaction.Target)
				fail = !controller.ExecuteAbi(transaction.Source, contract, string(transaction.Data))
				Logger.Debugf("TVMExecutor Execute ContractCall Transaction %s",transaction.Hash.Hex())
			case types.TransactionTypeBonus:
				if executor.bc.bonusManager.Contain(transaction.Data,accountdb) == false{
					Logger.Debugf("TVMExecutor Execute Bonus Transaction")
					reader := bytes.NewReader(transaction.ExtraData)
					groupId := make([]byte,common.GroupIdLength)
					addr := make([]byte,common.AddressLength)
					value := big.NewInt(int64(transaction.Value))
					if n,_ := reader.Read(groupId);n != common.GroupIdLength{
						panic("TVMExecutor Read GroupId Fail")
					}
					for n,_ := reader.Read(addr);n > 0;n,_ = reader.Read(addr){
						address := common.BytesToAddress(addr)
						accountdb.AddBalance(address,value)
						Logger.Debugf("TVMExecutor Bonus AddBalance Addr:%s Value:%d",address.GetHexString(),transaction.Value)
					}
					executor.bc.bonusManager.Put(transaction.Data, transaction.Hash[:],accountdb)
					//分红交易奖励
					accountdb.AddBalance(common.BytesToAddress(block.Header.Castor), bonusReward)
				} else {
					fail = true
				}
			case types.TransactionTypeMinerApply:
				Logger.Debugf("--------miner apply tx found! from %v\n", transaction.Source.GetHexString())
				if transaction.Data == nil{
					fail = true
					continue
				}
				var miner types.Miner
				msgpack.Unmarshal(transaction.Data,&miner)
				mexist := MinerManagerImpl.GetMinerById(transaction.Source[:],miner.Type)
				if mexist == nil{
					amount := big.NewInt(int64(transaction.Value))
					if CanTransfer(accountdb, *transaction.Source, amount){
						Logger.Debugf("TVMExecutor Execute MinerApply Transaction %s",transaction.Hash.Hex())
						MinerManagerImpl.AddMiner(transaction.Source[:],&miner)
						accountdb.SubBalance(*transaction.Source,amount)
					} else {
						fail = true
					}
				} else {
					fail = true
				}
			case types.TransactionTypeMinerAbort:
				if transaction.Data == nil{
					fail = true
					continue
				}
				fail = !MinerManagerImpl.AbortMiner(transaction.Source[:],transaction.Data[0],transaction.Value)
				Logger.Debugf("TVMExecutor Execute MinerAbort Transaction %s",transaction.Hash.Hex())
			case types.TransactionTypeMinerRefund:
				mexist := MinerManagerImpl.GetMinerById(transaction.Source[:],transaction.Data[0])
				if mexist != nil && mexist.Status == types.MinerStatusAbort{
					if !GroupChainImpl.WhetherMemberInActiveGroup(transaction.Source[:]) {
						MinerManagerImpl.RemoveMiner(transaction.Source[:], mexist.Type)
						amount := big.NewInt(int64(mexist.Stake))
						accountdb.AddBalance(*transaction.Source, amount)
						Logger.Debugf("TVMExecutor Execute MinerRefund Transaction %s",transaction.Hash.Hex())
					} else {
						fail = true
					}
				} else {
					fail = true
				}
		}

		receipt := t.NewReceipt(nil,fail,0)
		receipt.TxHash = transaction.Hash
		receipt.ContractAddress = contractAddress
		receipts[i] = receipt
	}
	//筑块奖励
	accountdb.AddBalance(common.BytesToAddress(block.Header.Castor), castorReward)

	//if nil != processor {
	//	processor.AfterAllTransactionExecuted(block, statedb, receipts)
	//}
	state := accountdb.IntermediateRoot(false)
	Logger.Debugf("TVMExecutor End Execute State %s",state.Hex())
	return state, receipts, nil
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
