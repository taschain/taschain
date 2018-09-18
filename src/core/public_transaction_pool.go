package core

import (
	"middleware"
	"github.com/hashicorp/golang-lru"
	"middleware/types"
	"sync"
	"common"
	"sort"
	vtypes "storage/core/types"

)

type ITransactionPool interface{
	ReserveTransactions(hash common.Hash, txs []*types.Transaction)
	GetReceived()[]*types.Transaction
	Clear()
	GetTransaction(hash common.Hash) (*types.Transaction, error)
	GetTransactionsForCasting()[]*types.Transaction
	GetTransactions(reservedHash common.Hash, hashes []common.Hash) ([]*types.Transaction, []common.Hash, error)
	Remove(hash common.Hash, transactions []common.Hash)
	AddExecuted(receipts vtypes.Receipts, txs []*types.Transaction)
	RemoveExecuted(txs []*types.Transaction)
	GetLock() middleware.Loglock
	AddTxs(txs []*types.Transaction)
}

type PublicTransactionPool struct {
	config *TransactionPoolConfig

	// 读写锁
	lock middleware.Loglock

	received    *container
	reserved    *lru.Cache
	sendingList []*types.Transaction

	sendingTxLock sync.Mutex

	totalReceived uint64
}

// 返回待处理的transaction数组
func (pool *PublicTransactionPool) ReserveTransactions(hash common.Hash, txs []*types.Transaction) {
	if 0 == len(txs) {
		return
	}
	pool.reserved.Add(hash, txs)
}

// 返回待处理的transaction数组
func (pool *PublicTransactionPool) GetTransactionsForCasting() []*types.Transaction {
	txs := pool.received.AsSlice()
	sort.Sort(types.Transactions(txs))
	return txs
}


func (pool *PublicTransactionPool) Clear() {
	panic("not expect enter here")
}

func (pool *PublicTransactionPool) GetReceived() []*types.Transaction {
	return pool.received.AsSlice()
}
func (pool *PublicTransactionPool) AddExecuted(receipts vtypes.Receipts, txs []*types.Transaction){
	//nothing to do
}
func (pool *PublicTransactionPool)RemoveExecuted(txs []*types.Transaction){
	//nothing to do
}

func (pool *PublicTransactionPool)GetLock() middleware.Loglock{
	return pool.lock
}
// 外部加锁
// 加缓冲区
func (pool *PublicTransactionPool) AddTxs(txs []*types.Transaction) {
	pool.received.PushTxs(txs)
}


