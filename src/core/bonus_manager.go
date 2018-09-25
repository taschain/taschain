package core

import (
	"storage/tasdb"
	"sync"
	"time"
)

const FlusPeriod = time.Second * 5

type BonusManager struct {
	//key: blockhash, value: transactionhash
	db tasdb.Database
	lock sync.RWMutex
	cache map[string][]byte
	timer *time.Timer
}

func newBonusManager(db tasdb.Database) *BonusManager {
	t := time.NewTimer(FlusPeriod)
	manager := &BonusManager{db:db,timer:t,cache:make(map[string][]byte)}
	go manager.doFlush()
	return manager
}

func (bm *BonusManager) Contain(blockHash []byte) bool {
	bm.lock.RLock()
	defer bm.lock.RUnlock()
	if _,ok := bm.cache[string(blockHash)];ok{
		return true
	}
	if ok,_ := bm.db.Has(blockHash);ok{
		return true
	}
	return false
}

func (bm *BonusManager) Put(blockHash []byte, transactionHash []byte)  {
	bm.lock.Lock()
	defer bm.lock.Unlock()
	bm.cache[string(blockHash)] = transactionHash
}

func (bm *BonusManager) flush()  {
	bm.lock.Lock()
	defer bm.lock.Unlock()
	batch := bm.db.NewBatch()
	for key,value := range bm.cache{
		batch.Put([]byte(key),value)
	}
	if batch.ValueSize() > 0{
		batch.Write()
	}
}

func (bm *BonusManager) doFlush(){
	for{
		select {
		case <-bm.timer.C:
			bm.flush()
			bm.timer.Reset(FlusPeriod)
		}
	}
}