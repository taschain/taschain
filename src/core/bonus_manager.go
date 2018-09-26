package core

import (
	"sync"
	"storage/core/vm"
	"common"
)


type BonusManager struct {
	lock sync.RWMutex
}

func newBonusManager() *BonusManager {
	manager := &BonusManager{}
	return manager
}

func (bm *BonusManager) Contain(blockHash []byte, accountdb vm.AccountDB) bool {
	value := accountdb.GetData(common.BonusStorageAddress,string(blockHash))
	if value != nil{
		return true
	}
	return false
}

func (bm *BonusManager) Put(blockHash []byte, transactionHash []byte, accountdb vm.AccountDB)  {
	accountdb.SetData(common.BonusStorageAddress, string(blockHash), transactionHash)
}