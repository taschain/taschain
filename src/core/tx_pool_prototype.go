package core

import (
	"middleware"
	"github.com/hashicorp/golang-lru"
	"middleware/types"
	"sync"
	"sort"
	"common"
)

type txPoolPrototype struct {
	config *TransactionPoolConfig

	// 读写锁
	lock middleware.Loglock

	received    *container
	reserved    *lru.Cache
	sendingList []*types.Transaction

	sendingTxLock sync.Mutex

	totalReceived uint64
}

// 外部加锁
// 加缓冲区
func (pool *txPoolPrototype) AddTxs(txs []*types.Transaction) {
	pool.received.PushTxs(txs)
}

// 从池子里移除一批交易
func (pool *txPoolPrototype) Remove(hash common.Hash, transactions []common.Hash) {
	pool.received.Remove(transactions)
	pool.reserved.Remove(hash)
	pool.removeFromSendinglist(transactions)
}

// 返回待处理的transaction数组
func (pool *txPoolPrototype) GetTransactionsForCasting() []*types.Transaction {
	txs := pool.received.AsSlice()
	sort.Sort(types.Transactions(txs))
	return txs
}


func (pool *txPoolPrototype) GetReceived() []*types.Transaction {
	return pool.received.AsSlice()
}

// 返回待处理的transaction数组
func (pool *txPoolPrototype) ReserveTransactions(hash common.Hash, txs []*types.Transaction) {
	if 0 == len(txs) {
		return
	}
	pool.reserved.Add(hash, txs)
}

func (pool *txPoolPrototype)GetLock() *middleware.Loglock{
	return &pool.lock
}

func (p *txPoolPrototype) removeFromSendinglist(transactions []common.Hash) {
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

