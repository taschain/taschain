package core

import (
	"bytes"
	"common"
	"middleware/types"
	"storage/vm"
	"sync"
)

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
		buffer.Write(group.Members[index].Id)
		//Logger.Debugf("GenerateBonus Index:%d Member:%s",index,common.BytesToAddress(group.Members[index].Id).GetHexString())
	}
	transaction := &types.Transaction{}
	transaction.Data = blockHash.Bytes()
	transaction.ExtraData = buffer.Bytes()
	transaction.Hash = transaction.GenHash()
	transaction.Value = totalValue / uint64(len(targetIds))
	transaction.Type = types.TransactionTypeBonus
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
