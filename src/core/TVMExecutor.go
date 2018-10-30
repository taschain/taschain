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
	"storage/account"
	"math/big"
	"storage/vm"
	"fmt"
	"tvm"
	"bytes"
	"github.com/vmihailenco/msgpack"
	"storage/trie"
	"storage/serialize"
)

//var castorReward = big.NewInt(50)
//var bonusReward = big.NewInt(20)

type TVMExecutor struct {
	bc BlockChain
}

func NewTVMExecutor(bc BlockChain) *TVMExecutor {
	return &TVMExecutor{
		bc: bc,
	}
}

//获取交易中包含账户所在的分支
func (executor *TVMExecutor) GetBranches(accountdb *account.AccountDB, transactions []*types.Transaction, addresses []common.Address, nodes map[string]*[]byte) {
	//todo  合约如何实现
	Logger.Debugf("GetBranches, tx len:%d, accounts len:%d", len(transactions), len(addresses))
	tr, _ := accountdb.GetTrie().(*trie.Trie)

	for _, transaction := range transactions {
		switch transaction.Type {
		case types.TransactionTypeBonus:
			//分红交易
			reader := bytes.NewReader(transaction.ExtraData)
			groupId := make([]byte, common.GroupIdLength)
			addr := make([]byte, common.AddressLength)
			if n, _ := reader.Read(groupId); n != common.GroupIdLength {
				panic("TVMExecutor Read GroupId Fail")
			}
			Logger.Debugf("Bonus Transaction:%s Group:%s", common.BytesToHash(transaction.Data).Hex(), common.BytesToHash(groupId).ShortS())
			for n, _ := reader.Read(addr); n > 0; n, _ = reader.Read(addr) {
				address := common.BytesToAddress(addr)
				tr.GetBranch(address[:], nodes)
				Logger.Debugf("Bonus addr:%v,value:%v", address.GetHexString(),tr.Get(address[:]))
			}
		case types.TransactionTypeContractCreate, types.TransactionTypeContractCall:
			//todo 合约交易
		default:
			//转账交易
			source := transaction.Source
			target := transaction.Target
			Logger.Debugf("Transfer transaction source:%v.target:%v", source, target)

			if source != nil {
				tr.GetBranch(source[:], nodes)
				Logger.Debugf("source:%v,value:%v", source.GetHexString(),tr.Get(source[:]))
			}
			if target != nil {
				tr.GetBranch(target[:], nodes)
				Logger.Debugf("target:%v,value:%v", target.GetHexString(), tr.Get(target[:]))
			}
		}
	}

	for _, account := range addresses {
		tr.GetBranch(account[:], nodes)
		//Logger.Debugf("GetBranches Account addr:%v,value:%v", account.GetHexString(), tr.Get(account[:]))
		if account == common.BonusStorageAddress || account == common.LightDBAddress || account == common.HeavyDBAddress {
			getNodeTrie(account[:], nodes, accountdb)
		}
	}
}

func getNodeTrie(address []byte, nodes map[string]*[]byte, accountdb *account.AccountDB) {
	data, err := accountdb.GetTrie().TryGet(address[:])
	if err != nil {
		Logger.Errorf("Get nil from trie! addr:%v,err:%s", address, err.Error())
		return
	}

	var account account.Account
	if err := serialize.DecodeBytes(data, &account); err != nil {
		Logger.Errorf("Failed to decode state object! addr:%v,err:%s", address, err.Error())
		return
	}
	root := account.Root
	t, err := accountdb.Database().OpenTrieWithMap(root,nodes)
	if err != nil {
		Logger.Errorf("OpenStorageTrie error! addr:%v,err:%s", address, err.Error())
		return
	}
	t.GetAllNodes(nodes)
}

func (executor *TVMExecutor) FilterMissingAccountTransaction(accountdb *account.AccountDB, block *types.Block) ([]*types.Transaction, []common.Address) {
	missingAccountTransactions := []*types.Transaction{}
	missingAccounts := []common.Address{}
	Logger.Infof("FilterMissingAccountTransaction block height:%d-%d,len(block.Transactions):%d", block.Header.Height, block.Header.ProveValue, len(block.Transactions))
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
func (executor *TVMExecutor) Execute(accountdb *account.AccountDB, block *types.Block, height uint64, mark string) (common.Hash, []*types.Receipt, error) {
	receipts := make([]*types.Receipt, len(block.Transactions))
	//Logger.Debugf("TVMExecutor Begin Execute State %s,height:%d,tx len:%d", block.Header.StateTree.Hex(), block.Header.Height, len(block.Transactions))
	//tr := accountdb.GetTrie()
	//Logger.Debugf("TVMExecutor  Execute tree hash:%v", tr.Hash().String())

	for i, transaction := range block.Transactions {
		var fail = false
		var contractAddress common.Address
		//Logger.Debugf("TVMExecutor Execute %v,type:%d", transaction.Hash, transaction.Type)

		switch transaction.Type {
		case types.TransactionTypeTransfer:
			amount := big.NewInt(int64(transaction.Value))
			if CanTransfer(accountdb, *transaction.Source, amount) {
				Transfer(accountdb, *transaction.Source, *transaction.Target, amount)
				//Logger.Debugf("TVMExecutor Execute Transfer Source:%s Target:%s Value:%d Height:%d Type:%s", transaction.Source.GetHexString(),
				//	transaction.Target.GetHexString(), transaction.Value, height, mark)
			} else {
				fail = true
			}
		case types.TransactionTypeContractCreate:
			controller := tvm.NewController(accountdb, BlockChainImpl, block.Header, transaction, common.GlobalConf.GetString("tvm", "pylib", "lib"))
			contractAddress, _ = createContract(accountdb, transaction)
			contract := tvm.LoadContract(contractAddress)
			fail = !controller.Deploy(transaction.Source, contract)
			Logger.Debugf("TVMExecutor Execute ContractCreate Transaction %s", transaction.Hash.Hex())
		case types.TransactionTypeContractCall:
			controller := tvm.NewController(accountdb, BlockChainImpl, block.Header, transaction, common.GlobalConf.GetString("tvm", "pylib", "lib"))
			contract := tvm.LoadContract(*transaction.Target)
			fail = !controller.ExecuteAbi(transaction.Source, contract, string(transaction.Data))
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
			var miner types.Miner
			msgpack.Unmarshal(transaction.Data, &miner)
			mexist := MinerManagerImpl.GetMinerById(transaction.Source[:], miner.Type, accountdb)
			if mexist == nil {
				amount := big.NewInt(int64(transaction.Value))
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
					if !GroupChainImpl.WhetherMemberInActiveGroup(transaction.Source[:],height,mexist.ApplyHeight,mexist.AbortHeight) {
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

		receipt := types.NewReceipt(nil, fail, 0)
		receipt.TxHash = transaction.Hash
		receipt.ContractAddress = contractAddress
		receipts[i] = receipt
	}
	//筑块奖励
	accountdb.AddBalance(common.BytesToAddress(block.Header.Castor), executor.bc.GetConsensusHelper().ProposalBonus())

	//Logger.Debugf("After TVMExecutor  Execute tree root:%v",tr.Fstring())
	//Logger.Debugf("After TVMExecutor  Execute tree hash:%v", tr.Hash().String())
	state := accountdb.IntermediateRoot(true)
	Logger.Debugf("TVMExecutor End Execute State %s", state.Hex())

	return state, receipts, nil
}

func createContract(accountdb *account.AccountDB, transaction *types.Transaction) (common.Address, error) {
	amount := big.NewInt(int64(transaction.Value))
	if !CanTransfer(accountdb, *transaction.Source, amount) {
		return common.Address{}, fmt.Errorf("balance not enough")
	}

	nance := accountdb.GetNonce(*transaction.Source)
	accountdb.SetNonce(*transaction.Source, nance+1)
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(transaction.Source[:], common.Uint64ToByte(nance))))

	if accountdb.GetCodeHash(contractAddr) != (common.Hash{}) {
		return common.Address{}, fmt.Errorf("contract address conflict")
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
