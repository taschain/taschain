package core

import (
	"errors"
	"fmt"
	"sync"
	"common"
	"core/datasource"
	"os"

	"encoding/json"
	"sort"
	"vm/common/hexutil"
	"vm/ethdb"
	vtypes "vm/core/types"
	"middleware/types"
	"middleware"
	"container/heap"
)

var (
	ErrNil = errors.New("nil transaction")

	ErrHash = errors.New("invalid transaction hash")

	ErrInvalidSender = errors.New("invalid sender")

	ErrNonceTooLow = errors.New("nonce too low")

	ErrNonceTooHigh = errors.New("nonce too high")

	ErrUnderpriced = errors.New("transaction underpriced")

	ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")

	ErrExisted = errors.New("executed transaction")

	ErrNegativeValue = errors.New("negative value")

	ErrOversizedData = errors.New("oversized data")

	sendingListLength = 20
)

// 配置文件
type TransactionPoolConfig struct {
	maxReceivedPoolSize int
	tx                  string
}

type TransactionPool struct {
	config *TransactionPoolConfig

	// 读写锁
	lock middleware.Loglock

	// 待上块的交易
	received *container

	sendingList []*types.Transaction

	// 已经在块上的交易 key ：txhash value： receipt
	executed ethdb.Database
}

type ReceiptWrapper struct {
	Receipt     *vtypes.Receipt
	Transaction *types.Transaction
}

func DefaultPoolConfig() *TransactionPoolConfig {
	return &TransactionPoolConfig{
		maxReceivedPoolSize: 10000,
		tx:                  "tx",
	}
}

func getPoolConfig() *TransactionPoolConfig {
	defaultConfig := DefaultPoolConfig()
	if nil == common.GlobalConf {
		return defaultConfig
	}

	return &TransactionPoolConfig{
		maxReceivedPoolSize: common.GlobalConf.GetInt(CONFIG_SEC, "maxReceivedPoolSize", defaultConfig.maxReceivedPoolSize),
		tx:                  common.GlobalConf.GetString(CONFIG_SEC, "tx", defaultConfig.tx),
	}

}

func NewTransactionPool() *TransactionPool {

	pool := &TransactionPool{
		config:      getPoolConfig(),
		lock:        middleware.NewLoglock("txpool"),
		sendingList: make([]*types.Transaction, sendingListLength),
	}
	pool.received = newContainer(pool.config.maxReceivedPoolSize)

	executed, err := datasource.NewDatabase(pool.config.tx)
	if err != nil {
		//todo: rebuild executedPool
		return nil
	}
	pool.executed = executed

	return pool
}

func (pool *TransactionPool) Clear() {
	pool.lock.Lock("Clear")
	defer pool.lock.Unlock("Clear")

	os.RemoveAll(pool.config.tx)
	executed, _ := datasource.NewDatabase(pool.config.tx)
	pool.executed = executed
	pool.received = newContainer(pool.config.maxReceivedPoolSize)
}

func (pool *TransactionPool) GetReceived() []*types.Transaction {
	return pool.received.AsSlice()
}

// 返回待处理的transaction数组
func (pool *TransactionPool) GetTransactionsForCasting() []*types.Transaction {
	txs := pool.received.AsSlice()
	sort.Sort(types.Transactions(txs))
	return txs
}

// 不加锁
func (pool *TransactionPool) AddTransactions(txs []*types.Transaction) error {
	pool.lock.Lock("AddTransactions")
	defer pool.lock.Unlock("Add")

	if nil == txs || 0 == len(txs) {
		return ErrNil
	}

	for _, tx := range txs {
		_, err := pool.addInner(tx, false)
		if nil != err {
			return err
		}
	}

	return nil
}
func (pool *TransactionPool) Add(tx *types.Transaction) (bool, error) {
	pool.lock.Lock("Add")
	defer pool.lock.Unlock("Add")

	return pool.addInner(tx, true)
}

// 将一个合法的交易加入待处理队列。如果这个交易已存在，则丢掉
// 加锁
func (pool *TransactionPool) addInner(tx *types.Transaction, isBroadcast bool) (bool, error) {
	if tx == nil {
		return false, ErrNil
	}

	// 简单规则校验
	if err := pool.validate(tx); err != nil {
		//log.Trace("Discarding invalid transaction", "hash", hash, "err", err)

		return false, err
	}

	// 检查交易是否已经存在
	hash := tx.Hash
	if pool.isTransactionExisted(hash) {

		//log.Trace("Discarding already known transaction", "hash", hash)
		return false, fmt.Errorf("known transaction: %x", hash)
	}

	pool.add(tx)

	// batch broadcast
	if isBroadcast {
		pool.sendingList = append(pool.sendingList, tx)
		if sendingListLength == len(pool.sendingList) {
			txs := make([]*types.Transaction, sendingListLength)
			copy(txs, pool.sendingList)
			pool.sendingList = make([]*types.Transaction, sendingListLength)
			go BroadcastTransactions(txs)
		}
	}

	return true, nil
}

// 从池子里移除一批交易
func (pool *TransactionPool) Remove(transactions []common.Hash) {
	pool.received.Remove(transactions)

}

func (pool *TransactionPool) GetTransactions(hashes []common.Hash) ([]*types.Transaction, []common.Hash, error) {
	if nil == hashes || 0 == len(hashes) {
		return nil, nil, ErrNil
	}

	txs := make([]*types.Transaction, 0)
	need := make([]common.Hash, 0)
	var err error
	for _, hash := range hashes {
		tx, errInner := pool.GetTransaction(hash)
		if nil == errInner {
			txs = append(txs, tx)
		} else {
			need = append(need, hash)
			err = errInner
		}
	}

	return txs, need, err
}

// 根据hash获取交易实例
// 此处加锁
func (pool *TransactionPool) GetTransaction(hash common.Hash) (*types.Transaction, error) {
	pool.lock.RLock("GetTransaction")
	defer pool.lock.RUnlock("GetTransaction")

	// 先从received里获取
	result := pool.received.Get(hash)
	if nil != result {
		return result, nil
	}

	// 再从executed里获取
	executed := pool.GetExecuted(hash)
	if nil != executed {
		return executed.Transaction, nil
	}

	return nil, ErrNil
}

// 根据交易的hash判断交易是否存在：
// 1）已存在待处理列表
// 2）已存在块上
// 3）todo：曾经收到过的，不合法的交易
// 被add调用，外部加锁
func (pool *TransactionPool) isTransactionExisted(hash common.Hash) bool {
	result := pool.received.Contains(hash)
	if result {
		return true
	}

	existed, _ := pool.executed.Has(hash.Bytes())
	return existed
}

// 校验transaction是否合法
func (pool *TransactionPool) validate(tx *types.Transaction) error {
	if !tx.Hash.IsValid() {
		return ErrHash
	}

	return nil
}

// 直接加入未满的池子
// 不需要加锁，sync.Map保证并发
func (pool *TransactionPool) add(tx *types.Transaction) {
	pool.received.Push(tx)

}

// 外部加锁
// 加缓冲区
func (pool *TransactionPool) addTxs(txs []*types.Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}
	for _, tx := range txs {
		pool.add(tx)
	}
}

// 外部加锁，AddExecuted通常和remove操作是依次执行的，所以由外部控制锁
func (pool *TransactionPool) AddExecuted(receipts vtypes.Receipts, txs []*types.Transaction) {
	if nil == receipts || 0 == len(receipts) {
		return
	}

	for i, receipt := range receipts {
		hash := receipt.TxHash.Bytes()
		receiptWrapper := &ReceiptWrapper{
			Receipt:     receipt,
			Transaction: getTransaction(txs, hash, i),
		}

		receiptJson, err := json.Marshal(receiptWrapper)
		if nil != err {
			continue
		}
		pool.executed.Put(hash, receiptJson)
	}
}

func getTransaction(txs []*types.Transaction, hash []byte, i int) *types.Transaction {
	if nil == txs || 0 == len(txs) {
		return nil
	}
	if hexutil.Encode(txs[i].Hash.Bytes()) == hexutil.Encode(hash) {
		return txs[i]
	}

	for _, tx := range txs {
		if hexutil.Encode(tx.Hash.Bytes()) == hexutil.Encode(hash) {
			return tx
		}
	}

	return nil
}

// 从磁盘读取，不需要加锁（ldb自行保证）
func (pool *TransactionPool) GetExecuted(hash common.Hash) *ReceiptWrapper {
	receiptJson, _ := pool.executed.Get(hash.Bytes())
	if nil == receiptJson {
		return nil
	}

	var receipt *ReceiptWrapper
	err := json.Unmarshal(receiptJson, &receipt)
	if err != nil || receipt == nil {
		return nil
	}

	return receipt
}

// 本身不需要加锁（ldb自己保证）
func (pool *TransactionPool) RemoveExecuted(txs []*types.Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}
	for _, tx := range txs {
		pool.executed.Delete(tx.Hash.Bytes())
	}
}

type container struct {
	lock   sync.RWMutex
	txs    types.PriorityTransactions
	txsMap map[common.Hash]*types.Transaction
	limit  int
	inited bool
}

func newContainer(l int) *container {
	return &container{
		lock:   sync.RWMutex{},
		limit:  l,
		txsMap: map[common.Hash]*types.Transaction{},
		txs:    types.PriorityTransactions{},
		inited: false,
	}

}

func (c *container) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.txs)
}

func (c *container) Contains(key common.Hash) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.txsMap[key] != nil
}

func (c *container) Get(key common.Hash) *types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.txsMap[key]
}

func (c *container) AsSlice() []*types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

	result := make([]*types.Transaction, c.txs.Len())
	copy(result, c.txs)
	return result
}

func (c *container) Push(tx *types.Transaction) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.txs.Len() < c.limit {
		c.txs = append(c.txs, tx)
		c.txsMap[tx.Hash] = tx
		return
	}

	if !c.inited {
		heap.Init(&c.txs)
		c.inited = true

	}

	evicted := heap.Pop(&c.txs).(*types.Transaction)
	delete(c.txsMap, evicted.Hash)
	heap.Push(&c.txs, tx)
	c.txsMap[tx.Hash] = tx
}

func (c *container) Remove(keys []common.Hash) {
	if nil == keys || 0 == len(keys) {
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	for _, key := range keys {
		if c.txsMap[key] == nil {
			return
		}

		delete(c.txsMap, key)
		index := -1
		for i, tx := range c.txs {
			if tx.Hash == key {
				index = i
				break
			}
		}
		heap.Remove(&c.txs, index)
	}

}
