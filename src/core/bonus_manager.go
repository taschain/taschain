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
	"middleware/types"
	"storage/vm"
	"sync"
)

/*
**  Creator: Kaede
**  Date: 2018/9/25 下午3:05
**  Description: 在AccountDB上使用特殊地址0存储分红交易执行记录，保证一块只执行一次分红交易
*/
type BonusManager struct {
	lock sync.RWMutex
}

func newBonusManager() *BonusManager {
	manager := &BonusManager{}
	return manager
}

func (bm *BonusManager) Contain(blockHash []byte, accountdb vm.AccountDB) bool {
	value := accountdb.GetData(common.BonusStorageAddress, string(blockHash))
	if value != nil {
		return true
	}
	return false
}

func (bm *BonusManager) Put(blockHash []byte, transactionHash []byte, accountdb vm.AccountDB) {
	accountdb.SetData(common.BonusStorageAddress, string(blockHash), transactionHash)
}

func (bm *BonusManager) WhetherBonusTransaction(transaction *types.Transaction) bool {
	return transaction.Type == types.TransactionTypeBonus
}


func (bm *BonusManager) GetBonusTransactionByBlockHash(blockHash []byte) *types.Transaction{
	transactionHash := BlockChainImpl.LatestStateDB().GetData(common.BonusStorageAddress, string(blockHash))
	if transactionHash == nil{
		return nil
	}
	transaction, _ := BlockChainImpl.(*FullBlockChain).transactionPool.GetTransaction(common.BytesToHash(transactionHash))
	return transaction
}

func (bm *BonusManager) GenerateBonus(targetIds []int32, blockHash common.Hash, groupId []byte, totalValue uint64) (*types.Bonus, *types.Transaction) {
	group := GroupChainImpl.getGroupById(groupId)
	buffer := &bytes.Buffer{}
	buffer.Write(groupId)
	//Logger.Debugf("GenerateBonus Group:%s",common.BytesToAddress(groupId).GetHexString())
	for i := 0; i < len(targetIds); i++ {
		index := targetIds[i]
		buffer.Write(group.Members[index])
		//Logger.Debugf("GenerateBonus Index:%d Member:%s",index,common.BytesToAddress(group.Members[index].Id).GetHexString())
	}
	transaction := &types.Transaction{}
	transaction.Data = blockHash.Bytes()
	transaction.ExtraData = buffer.Bytes()
	transaction.Value = totalValue / uint64(len(targetIds))
	transaction.Type = types.TransactionTypeBonus
	transaction.GasPrice = common.MaxUint64
	transaction.Hash = transaction.GenHash()
	return &types.Bonus{TxHash: transaction.Hash, TargetIds: targetIds, BlockHash: blockHash, GroupId: groupId, TotalValue: totalValue}, transaction
}

func (bm *BonusManager) ParseBonusTransaction(transaction *types.Transaction) ([]byte, [][]byte, common.Hash, uint64) {
	reader := bytes.NewReader(transaction.ExtraData)
	groupId := make([]byte, common.GroupIdLength)
	addr := make([]byte, common.AddressLength)
	if n, _ := reader.Read(groupId); n != common.GroupIdLength {
		panic("ParseBonusTransaction Read GroupId Fail")
	}
	ids := make([][]byte, 0)
	for n, _ := reader.Read(addr); n > 0; n, _ = reader.Read(addr) {
		if n != common.AddressLength {
			panic("ParseBonusTransaction Read Address Fail")
		}
		ids = append(ids, addr)
		addr = make([]byte, common.AddressLength)
	}
	blockHash := common.BytesToHash(transaction.Data)
	return groupId, ids, blockHash, transaction.Value
}
