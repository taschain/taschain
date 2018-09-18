package core


import (
	"middleware"
	"github.com/hashicorp/golang-lru"
	"middleware/types"
	"sync"
	"common"
)

type LightTransactionPool struct {
	PublicTransactionPool
}

func NewLightTransactionPool() *LightTransactionPool {
	pool := &LightTransactionPool{
		PublicTransactionPool:PublicTransactionPool{
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


func (pool *LightTransactionPool) GetTransaction(hash common.Hash) (*types.Transaction, error) {
	pool.lock.RLock("GetTransaction")
	defer pool.lock.RUnlock("GetTransaction")

	return pool.getTransaction(hash)
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


// 从池子里移除一批交易
func (pool *LightTransactionPool) Remove(hash common.Hash, transactions []common.Hash) {
	pool.received.Remove(transactions)
	pool.reserved.Remove(hash)
	pool.removeFromSendinglist(transactions)
}

func (p *LightTransactionPool) removeFromSendinglist(transactions []common.Hash) {
	if nil == transactions || 0 == len(transactions) {
		return
	}
	p.sendingTxLock.Lock()
	for _, hash := range transactions {
		for i, tx := range p.sendingList {
			if tx.Hash == hash {
				p.sendingList = append(p.sendingList[:i], p.sendingList[i+1:]...)
			}
			break
		}
	}
	p.sendingTxLock.Unlock()
}
