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

func (executor *TVMExecutor) Execute(accountdb *core.AccountDB, block *types.Block, height uint64,mark string) (common.Hash,[]*t.Receipt,error) {
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
					Logger.Debugf("TVMExecutor Execute Transfer Source:%s Target:%s Value:%d Height:%d Type:%s",transaction.Source.GetHexString(),
						transaction.Target.GetHexString(),transaction.Value,height,mark)
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
				mexist := MinerManagerImpl.GetMinerById(transaction.Source[:],miner.Type, accountdb)
				if mexist == nil{
					amount := big.NewInt(int64(transaction.Value))
					if CanTransfer(accountdb, *transaction.Source, amount){
						miner.ApplyHeight = height
						if MinerManagerImpl.AddMiner(transaction.Source[:],&miner,accountdb) > 0 {
							accountdb.SubBalance(*transaction.Source, amount)
							Logger.Debugf("TVMExecutor Execute MinerApply Success Source:%s Height:%d Type:%s",transaction.Source.GetHexString(),height,mark)
						}
					} else {
						fail = true
						Logger.Debugf("TVMExecutor Execute MinerApply Fail(Balance Not Enough) Source:%s Height:%d Type:%s",transaction.Source.GetHexString(),height,mark)
					}
				} else {
					fail = true
					Logger.Debugf("TVMExecutor Execute MinerApply Fail(Already Exist) Source %s",transaction.Source.GetHexString())
				}
			case types.TransactionTypeMinerAbort:
				if transaction.Data == nil{
					fail = true
					continue
				}
				fail = !MinerManagerImpl.AbortMiner(transaction.Source[:],transaction.Data[0],height,accountdb)
				Logger.Debugf("TVMExecutor Execute MinerAbort %s Success:%t",transaction.Source.GetHexString(),!fail)
			case types.TransactionTypeMinerRefund:
				mexist := MinerManagerImpl.GetMinerById(transaction.Source[:],transaction.Data[0], accountdb)
				if mexist != nil && mexist.Status == types.MinerStatusAbort{
					if mexist.Type == types.MinerTypeHeavy {
						if height > mexist.AbortHeight + 10{
							MinerManagerImpl.RemoveMiner(transaction.Source[:], mexist.Type, accountdb)
							amount := big.NewInt(int64(mexist.Stake))
							accountdb.AddBalance(*transaction.Source, amount)
							Logger.Debugf("TVMExecutor Execute MinerRefund Heavy Success %s", transaction.Source.GetHexString())
						}else {
							Logger.Debugf("TVMExecutor Execute MinerRefund Heavy Fail %s", transaction.Source.GetHexString())
						}
					}else {
						if !GroupChainImpl.WhetherMemberInActiveGroup(transaction.Source[:]) {
							MinerManagerImpl.RemoveMiner(transaction.Source[:], mexist.Type, accountdb)
							amount := big.NewInt(int64(mexist.Stake))
							accountdb.AddBalance(*transaction.Source, amount)
							Logger.Debugf("TVMExecutor Execute MinerRefund Light Success %s", transaction.Source.GetHexString())
						} else {
							fail = true
							Logger.Debugf("TVMExecutor Execute MinerRefund Light Fail(Still In Active Group) %s", transaction.Source.GetHexString())
						}
					}
				} else {
					fail = true
					Logger.Debugf("TVMExecutor Execute MinerRefund Fail(Not Exist Or Not Abort) %s",transaction.Source.GetHexString())
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
