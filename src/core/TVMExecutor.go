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
	"bytes"
	"common"
	"fmt"
	"github.com/vmihailenco/msgpack"
	"math/big"
	"middleware/types"
	"storage/account"
	"storage/vm"
	"time"
	"tvm"
)

//var castorReward = big.NewInt(50)
//var bonusReward = big.NewInt(20)
const TransferGasCost = 1000
const CodeBytePrice = 0.3814697265625
const MaxCastBlockTime = time.Second * 3

type TVMExecutor struct {
	bc BlockChain
}

func NewTVMExecutor(bc BlockChain) *TVMExecutor {
	return &TVMExecutor{
		bc: bc,
	}
}

//获取交易中包含账户所在的分支，LightChain使用
//func (executor *TVMExecutor) GetBranches(accountdb *account.AccountDB, transactions []*types.Transaction, addresses []common.Address, nodes map[string]*[]byte) {
//	//todo  合约如何实现
//	Logger.Debugf("GetBranches, tx len:%d, accounts len:%d", len(transactions), len(addresses))
//	tr, _ := accountdb.GetTrie().(*trie.Trie)
//
//	for _, transaction := range transactions {
//		switch transaction.Type {
//		case types.TransactionTypeBonus:
//			//分红交易
//			reader := bytes.NewReader(transaction.ExtraData)
//			groupId := make([]byte, common.GroupIdLength)
//			addr := make([]byte, common.AddressLength)
//			if n, _ := reader.Read(groupId); n != common.GroupIdLength {
//				panic("TVMExecutor Read GroupId Fail")
//			}
//			Logger.Debugf("Bonus Transaction:%s Group:%s", common.BytesToHash(transaction.Data).Hex(), common.BytesToHash(groupId).ShortS())
//			for n, _ := reader.Read(addr); n > 0; n, _ = reader.Read(addr) {
//				address := common.BytesToAddress(addr)
//				tr.GetBranch(address[:], nodes)
//				Logger.Debugf("Bonus addr:%v,value:%v", address.GetHexString(), tr.Get(address[:]))
//			}
//		case types.TransactionTypeContractCreate, types.TransactionTypeContractCall:
//			//todo 合约交易
//		default:
//			//转账交易
//			source := transaction.Source
//			target := transaction.Target
//
//			Logger.Debugf("Transfer transaction source:%v.target:%v", source, target)
//
//			if source != nil {
//				tr.GetBranch(source[:], nodes)
//				Logger.Debugf("source:%v,value:%v", source.GetHexString(), tr.Get(source[:]))
//			}
//			if target != nil {
//				tr.GetBranch(target[:], nodes)
//				Logger.Debugf("target:%v,value:%v", target.GetHexString(), tr.Get(target[:]))
//			}
//		}
//	}
//
//	for _, account := range addresses {
//		tr.GetBranch(account[:], nodes)
//		//Logger.Debugf("GetBranches Account addr:%v,value:%v", account.GetHexString(), tr.Get(account[:]))
//		if account == common.BonusStorageAddress || account == common.LightDBAddress || account == common.HeavyDBAddress {
//			getNodeTrie(account[:], nodes, accountdb)
//		}
//	}
//}
//
//func getNodeTrie(address []byte, nodes map[string]*[]byte, accountdb *account.AccountDB) {
//	data, err := accountdb.GetTrie().TryGet(address[:])
//	if err != nil {
//		Logger.Errorf("Get nil from trie! addr:%v,err:%s", address, err.Error())
//		return
//	}
//
//	var account account.Account
//	if err := serialize.DecodeBytes(data, &account); err != nil {
//		Logger.Errorf("Failed to decode state object! addr:%v,err:%s", address, err.Error())
//		return
//	}
//	root := account.Root
//	t, err := accountdb.Database().OpenTrieWithMap(root, nodes)
//	if err != nil {
//		Logger.Errorf("OpenStorageTrie error! addr:%v,err:%s", address, err.Error())
//		return
//	}
//	t.GetAllNodes(nodes)
//}

//哪些交易涉及的account在AccountDB中缺失，LightChain使用
func (executor *TVMExecutor) FilterMissingAccountTransaction(accountdb *account.AccountDB, block *types.Block) ([]*types.Transaction, []common.Address) {
	missingAccountTransactions := []*types.Transaction{}
	missingAccounts := []common.Address{}
	Logger.Debugf("FilterMissingAccountTransaction block height:%d-%d,len(block.Transactions):%d", block.Header.Height, block.Header.ProveValue, len(block.Transactions))
	for _, transaction := range block.Transactions {

		//Logger.Debugf("FilterMissingAccountTransaction source:%v.target:%v", transaction.Source, transaction.Target)
		switch transaction.Type {
		case types.TransactionTypeBonus:
			addresses := getBonusAddress(*transaction)
			for _, addr := range addresses {
				if !IsAccountExist(accountdb, addr) {
					missingAccountTransactions = append(missingAccountTransactions, transaction)
				}
			}
			castor := common.BytesToAddress(block.Header.Castor)
			if !IsAccountExist(accountdb, castor) {
				missingAccounts = append(missingAccounts, castor)
			}
		case types.TransactionTypeContractCreate, types.TransactionTypeContractCall:
			//todo 合约交易
		default:
			//转账交易
			if transaction.Source != nil && !IsAccountExist(accountdb, *transaction.Source) {
				missingAccountTransactions = append(missingAccountTransactions, transaction)
				continue
			}
			if transaction.Target != nil && !IsAccountExist(accountdb, *transaction.Target) {
				missingAccountTransactions = append(missingAccountTransactions, transaction)
				continue
			}
		}
	}
	if !IsAccountExist(accountdb, common.BonusStorageAddress) {
		missingAccounts = append(missingAccounts, common.BonusStorageAddress)
	}
	if !IsAccountExist(accountdb, common.LightDBAddress) {
		missingAccounts = append(missingAccounts, common.LightDBAddress)
	}
	if !IsAccountExist(accountdb, common.HeavyDBAddress) {
		missingAccounts = append(missingAccounts, common.HeavyDBAddress)
	}
	accountdb.IntermediateRoot(true)
	return missingAccountTransactions, missingAccounts
}

func getBonusAddress(t types.Transaction) []common.Address {
	var result = make([]common.Address, 0)

	reader := bytes.NewReader(t.ExtraData)
	groupId := make([]byte, common.GroupIdLength)
	addr := make([]byte, common.AddressLength)
	if n, _ := reader.Read(groupId); n != common.GroupIdLength {
		panic("TVMExecutor Read GroupId Fail")
	}
	for n, _ := reader.Read(addr); n > 0; n, _ = reader.Read(addr) {
		address := common.BytesToAddress(addr)
		result = append(result, address)
	}
	return result
}

func (executor *TVMExecutor) Execute(accountdb *account.AccountDB, block *types.Block, height uint64, mark string) (common.Hash, []common.Hash, []*types.Transaction, []*types.Receipt, error, []*types.TransactionError) {
	beginTime := time.Now()
	receipts := make([]*types.Receipt, 0)
	transactions := make([]*types.Transaction, 0)
	evictedTxs := make([]common.Hash, 0)
	errs := make([]*types.TransactionError, len(block.Transactions))
	//Logger.Debugf("TVMExecutor Begin Execute State %s,height:%d,tx len:%d", block.Header.StateTree.Hex(), block.Header.Height, len(block.Transactions))

	for i, transaction := range block.Transactions {
		executeTime := time.Now()
		if mark == "casting" && executeTime.Sub(beginTime) > MaxCastBlockTime {
			Logger.Infof("Cast block execute tx time out!Tx hash:%s ", transaction.Hash.String())
			break
		}
		var fail = false
		var contractAddress common.Address
		var logs []*types.Log
		var err *types.TransactionError
		var cumulativeGasUsed uint64
		//Logger.Debugf("TVMExecutor Execute %v,type:%d", transaction.Hash, transaction.Type)

		if transaction.Type != types.TransactionTypeBonus && !types.IsTestTransaction(transaction) {
			nonce := accountdb.GetNonce(*transaction.Source)
			if transaction.Nonce != nonce+1 {
				Logger.Infof("Tx nonce error! Source:%s,expect nonce:%d,real nonce:%d ", transaction.Source.GetHexString(), nonce+1, transaction.Nonce)
				evictedTxs = append(evictedTxs, transaction.Hash)
				continue
			}
		}
		switch transaction.Type {
		case types.TransactionTypeTransfer:
			amount := big.NewInt(int64(transaction.Value))
			if CanTransfer(accountdb, *transaction.Source, amount) {
				Transfer(accountdb, *transaction.Source, *transaction.Target, amount)
				//Logger.Debugf("TVMExecutor Execute Transfer Source:%s Target:%s Value:%d Height:%d Type:%s", transaction.Source.GetHexString(),
				//	transaction.Target.GetHexString(), transaction.Value, height, mark)
				gas := big.NewInt(int64(transaction.GasPrice * TransferGasCost))
				accountdb.SubBalance(*transaction.Source, gas)
				cumulativeGasUsed = gas.Uint64()
			} else {
				fail = true
				err = types.TxErrorBalanceNotEnough
			}

		case types.TransactionTypeContractCreate:
			amount := big.NewInt(int64(transaction.GasLimit * transaction.GasPrice))
			if CanTransfer(accountdb, *transaction.Source, amount) {
				accountdb.SubBalance(*transaction.Source, amount)
				controller := tvm.NewController(accountdb, BlockChainImpl, block.Header, transaction, common.GlobalConf.GetString("tvm", "pylib", "lib"))
				snapshot := controller.AccountDB.Snapshot()
				contractAddress, err = createContract(accountdb, transaction)
				Logger.Debugf("contract addr:%s", contractAddress.String())
				if err != nil {
					Logger.Debugf("ContractCreate error:%s ", err.Message)
					fail = true
					controller.AccountDB.RevertToSnapshot(snapshot)
				} else {
					deploySpend := uint64(float32(len(transaction.Data)) * CodeBytePrice)
					if controller.Transaction.GasLimit < deploySpend { //gas not enough
						fail = true
						err = types.TxErrorDeployGasNotEnough
						controller.AccountDB.RevertToSnapshot(snapshot)
					} else {
						controller.Transaction.GasLimit -= deploySpend
						contract := tvm.LoadContract(contractAddress)
						errorCode, errorMsg := controller.Deploy(transaction.Source, contract)
						if errorCode != 0 {
							fail = true
							err = types.NewTransactionError(errorCode, errorMsg)
							controller.AccountDB.RevertToSnapshot(snapshot)
						}
					}
				}
				gasLeft := big.NewInt(int64(controller.GetGasLeft() * transaction.GasPrice))
				accountdb.AddBalance(*transaction.Source, gasLeft)
				gasUsed := new(big.Int).Sub(amount, gasLeft)
				cumulativeGasUsed = gasUsed.Uint64()
			} else {
				fail = true
				err = types.TxErrorBalanceNotEnough
				Logger.Debugf("ContractCreate transaction source %s balance not enough! ", transaction.Source.String())
			}

			Logger.Debugf("TVMExecutor Execute ContractCreate Transaction %s", transaction.Hash.Hex())
		case types.TransactionTypeContractCall:
			amount := big.NewInt(int64(transaction.GasLimit * transaction.GasPrice))
			if CanTransfer(accountdb, *transaction.Source, amount) {
				accountdb.SubBalance(*transaction.Source, big.NewInt(int64(transaction.GasLimit*transaction.GasPrice)))
				controller := tvm.NewController(accountdb, BlockChainImpl, block.Header, transaction, common.GlobalConf.GetString("tvm", "pylib", "lib"))
				contract := tvm.LoadContract(*transaction.Target)
				if contract.Code == "" {
					err = types.NewTransactionError(types.TxErrorCode_NO_CODE, fmt.Sprintf(types.NO_CODE_ERROR_MSG, *transaction.Target))
					fail = true
				} else {
					snapshot := controller.AccountDB.Snapshot()
					var success bool
					success, logs, err = controller.ExecuteAbi(transaction.Source, contract, string(transaction.Data))
					if !success {
						controller.AccountDB.RevertToSnapshot(snapshot)
						fail = true
					}
					gasLeft := big.NewInt(int64(controller.GetGasLeft() * transaction.GasPrice))
					accountdb.AddBalance(*transaction.Source, gasLeft)
					gasUsed := new(big.Int).Sub(amount, gasLeft)
					cumulativeGasUsed = gasUsed.Uint64()
				}
			} else {
				fail = true
				err = types.TxErrorBalanceNotEnough
			}
			Logger.Debugf("TVMExecutor Execute ContractCall Transaction %s", transaction.Hash.Hex())
		case types.TransactionTypeBonus:
			if executor.bc.GetBonusManager().Contain(transaction.Data, accountdb) == false {
				reader := bytes.NewReader(transaction.ExtraData)
				groupId := make([]byte, common.GroupIdLength)
				addr := make([]byte, common.AddressLength)
				value := big.NewInt(int64(transaction.Value))
				if n, _ := reader.Read(groupId); n != common.GroupIdLength {
					panic("TVMExecutor Read GroupId Fail")
				}
				//Logger.Debugf("TVMExecutor Execute Bonus Transaction:%s Group:%s", common.BytesToHash(transaction.Data).Hex(), common.BytesToHash(groupId).ShortS())
				for n, _ := reader.Read(addr); n > 0; n, _ = reader.Read(addr) {
					if n != common.AddressLength {
						Logger.Debugf("TVMExecutor Bonus Addr Size:%d Invalid", n)
						break
					}
					address := common.BytesToAddress(addr)
					accountdb.AddBalance(address, value)
				}

				executor.bc.GetBonusManager().Put(transaction.Data, transaction.Hash[:], accountdb)
				//Logger.Debugf("TVMExecutor Bonus BonusManager Put BlockHash:%s TransactionHash:%s", common.BytesToHash(transaction.Data).Hex(),
				//	transaction.Hash.Hex())
				//分红交易奖励
				accountdb.AddBalance(common.BytesToAddress(block.Header.Castor), executor.bc.GetConsensusHelper().PackBonus())
				//Logger.Debugf("TVMExecutor Bonus AddBalance Addr:%s Value:%d", block.Header.Castor, executor.bc.GetConsensusHelper().PackBonus())
			} else {
				fail = true
			}
		case types.TransactionTypeMinerApply:
			Logger.Debugf("--------miner apply tx found! from %v\n", transaction.Source.GetHexString())
			if transaction.Data == nil {
				fail = true
				continue
			}
			data := common.FromHex(string(transaction.Data))
			var miner types.Miner
			msgpack.Unmarshal(data, &miner)
			mexist := MinerManagerImpl.GetMinerById(transaction.Source[:], miner.Type, accountdb)
			if mexist == nil {
				amount := big.NewInt(int64(miner.Stake))
				if CanTransfer(accountdb, *transaction.Source, amount) {
					miner.ApplyHeight = height
					if MinerManagerImpl.AddMiner(transaction.Source[:], &miner, accountdb) > 0 {
						accountdb.SubBalance(*transaction.Source, amount)
						Logger.Debugf("TVMExecutor Execute MinerApply Success Source:%s Height:%d Type:%s", transaction.Source.GetHexString(), height, mark)
					}
				} else {
					fail = true
					Logger.Debugf("TVMExecutor Execute MinerApply Fail(Balance Not Enough) Source:%s Height:%d Type:%s", transaction.Source.GetHexString(), height, mark)
				}
			} else {
				fail = true
				Logger.Debugf("TVMExecutor Execute MinerApply Fail(Already Exist) Source %s", transaction.Source.GetHexString())
			}
		case types.TransactionTypeMinerAbort:
			if transaction.Data == nil {
				fail = true
				continue
			}
			fail = !MinerManagerImpl.AbortMiner(transaction.Source[:], transaction.Data[0], height, accountdb)
			Logger.Debugf("TVMExecutor Execute MinerAbort %s Success:%t", transaction.Source.GetHexString(), !fail)
		case types.TransactionTypeMinerRefund:
			mexist := MinerManagerImpl.GetMinerById(transaction.Source[:], transaction.Data[0], accountdb)
			if mexist != nil && mexist.Status == types.MinerStatusAbort {
				if mexist.Type == types.MinerTypeHeavy {
					if height > mexist.AbortHeight+10 {
						MinerManagerImpl.RemoveMiner(transaction.Source[:], mexist.Type, accountdb)
						amount := big.NewInt(int64(mexist.Stake))
						accountdb.AddBalance(*transaction.Source, amount)
						Logger.Debugf("TVMExecutor Execute MinerRefund Heavy Success %s", transaction.Source.GetHexString())
					} else {
						Logger.Debugf("TVMExecutor Execute MinerRefund Heavy Fail %s", transaction.Source.GetHexString())
					}
				} else {
					if !GroupChainImpl.WhetherMemberInActiveGroup(transaction.Source[:], height, mexist.ApplyHeight, mexist.AbortHeight) {
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
				Logger.Debugf("TVMExecutor Execute MinerRefund Fail(Not Exist Or Not Abort) %s", transaction.Source.GetHexString())
			}

		}
		if !fail {
			transactions = append(transactions, transaction)
			receipt := types.NewReceipt(nil, fail, cumulativeGasUsed)
			receipt.Logs = logs
			receipt.TxHash = transaction.Hash
			receipt.ContractAddress = contractAddress
			receipts = append(receipts, receipt)
			errs[i] = err
			if transaction.Source != nil {
				accountdb.SetNonce(*transaction.Source, transaction.Nonce)
			}
		} else {
			evictedTxs = append(evictedTxs, transaction.Hash)
		}
	}

	//筑块奖励
	accountdb.AddBalance(common.BytesToAddress(block.Header.Castor), executor.bc.GetConsensusHelper().ProposalBonus())

	state := accountdb.IntermediateRoot(true)
	Logger.Debugf("TVMExecutor End Execute State %s", state.Hex())

	return state, evictedTxs, transactions, receipts, nil, errs
}

func createContract(accountdb *account.AccountDB, transaction *types.Transaction) (common.Address, *types.TransactionError) {
	amount := big.NewInt(int64(transaction.Value))
	if !CanTransfer(accountdb, *transaction.Source, amount) {
		return common.Address{}, types.TxErrorBalanceNotEnough
	}

	nance := accountdb.GetNonce(*transaction.Source)
	accountdb.SetNonce(*transaction.Source, nance+1)
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(transaction.Source[:], common.Uint64ToByte(nance))))

	if accountdb.GetCodeHash(contractAddr) != (common.Hash{}) {
		return common.Address{}, types.NewTransactionError(types.TxErrorCode_ContractAddressConflict, "contract address conflict")
	}
	accountdb.CreateAccount(contractAddr)
	accountdb.SetCode(contractAddr, transaction.Data)
	accountdb.SetNonce(contractAddr, 1)

	Transfer(accountdb, *transaction.Source, contractAddr, amount)
	return contractAddr, nil
}

func IsAccountExist(db vm.AccountDB, addr common.Address) bool {
	data, _ := db.GetTrie().TryGet(addr[:])
	if data == nil {
		return false
	}
	return true
}

func CanTransfer(db vm.AccountDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

func Transfer(db vm.AccountDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}
