package core

import (
	"middleware"
	"github.com/hashicorp/golang-lru"
	"middleware/types"
	"sync"
	"common"
	vtypes "storage/core/types"

)

type LightTransactionPool struct {
	txPoolPrototype
}

func NewLightTransactionPool() *LightTransactionPool {
	pool := &LightTransactionPool{
		txPoolPrototype:txPoolPrototype{
			config:        getPoolConfig(),
			lock:          middleware.NewLoglock("txpool"),
			sendingList:   make([]*types.Transaction, 0),
			sendingTxLock: sync.Mutex{},
		},
	}
	pool.received = newContainer(pool.config.maxReceivedPoolSize)
	pool.reserved, _ = lru.New(100)
	return pool
}


func (pool *LightTransactionPool)AddTransactions(txs []*types.Transaction) error{
	pool.received.PushTxs(txs)
	return nil
}


func (pool *LightTransactionPool)  AddExecuted(receipts vtypes.Receipts, txs []*types.Transaction){
	panic("Not support!")
}


func (pool *LightTransactionPool) RemoveExecuted(txs []*types.Transaction){
	panic("Not support!")
}


func (pool *LightTransactionPool) GetTransaction(hash common.Hash) (*types.Transaction, error) {
	pool.lock.RLock("GetTransaction")
	defer pool.lock.RUnlock("GetTransaction")

	return pool.getTransaction(hash)
}

func (pool *LightTransactionPool) GetTransactions(reservedHash common.Hash, hashes []common.Hash) ([]*types.Transaction, []common.Hash, error) {
	if nil == hashes || 0 == len(hashes) {
		return nil, nil, ErrNil
	}

	pool.lock.RLock("GetTransactions")
	defer pool.lock.RUnlock("GetTransactions")
	reservedRaw, _ := pool.reserved.Get(reservedHash)
	var reserved []*types.Transaction
	if nil != reservedRaw {
		reserved = reservedRaw.([]*types.Transaction)
	}

	txs := make([]*types.Transaction, 0)
	need := make([]common.Hash, 0)
	var err error
	for _, hash := range hashes {

		tx, errInner := getTx(reserved, hash)

		if tx == nil {
			tx, errInner = pool.getTransaction(hash)
		}

		if nil == errInner {
			txs = append(txs, tx)
		} else {
			need = append(need, hash)
			err = errInner
		}
	}

	return txs, need, err
}

func (pool *LightTransactionPool) getTransaction(hash common.Hash) (*types.Transaction, error) {
	// 先从received里获取
	result := pool.received.Get(hash)
	if nil != result {
		return result, nil
	}
	return nil, ErrNil
}


func (pool *LightTransactionPool) Clear() {
	pool.lock.Lock("Clear")
	defer pool.lock.Unlock("Clear")

	pool.received = newContainer(pool.config.maxReceivedPoolSize)
}


func (pool *LightTransactionPool) AddTransaction(tx *types.Transaction)(bool,error) {
	return true,nil
}