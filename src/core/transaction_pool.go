package core

import (
	"errors"
	"fmt"
	"sync"
	"common"
	"core/datasource"
	"os"
	"vm/core/types"
	"encoding/json"
	"sort"
	"vm/common/hexutil"
	"vm/ethdb"
	"middleware"
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

	// 收到的待处理transaction
	received sync.Map
	//map[common.Hash]*Transaction

	// 当前received数组里，price最小的transaction
	lowestPrice *Transaction

	// 已经在块上的交易 key ：txhash value： receipt
	executed ethdb.Database
}

type ReceiptWrapper struct {
	Receipt     *types.Receipt
	Transaction *Transaction
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
		received:    sync.Map{},
		lowestPrice: nil,
	}
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
	pool.received = sync.Map{}
}

func (pool *TransactionPool) GetReceived() sync.Map {
	return pool.received
}

// 返回待处理的transaction数组
func (pool *TransactionPool) GetTransactionsForCasting() []*Transaction {
	txs := make([]*Transaction, 0)
	pool.received.Range(func(key, value interface{}) bool {
		txs = append(txs, value.(*Transaction))

		return true
	})

	sort.Sort(Transactions(txs))
	return txs
}

// 不加锁
func (pool *TransactionPool) AddTransactions(txs []*Transaction) error {
	if nil == txs || 0 == len(txs) {
		return ErrNil
	}

	for _, tx := range txs {
		_, err := pool.Add(tx)
		if nil != err {
			return err
		}
	}

	return nil
}

// 将一个合法的交易加入待处理队列。如果这个交易已存在，则丢掉
// 加锁
func (pool *TransactionPool) Add(tx *Transaction) (bool, error) {
	pool.lock.Lock("Add")
	defer pool.lock.Unlock("Add")

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

	// 池子满了
	if length(&pool.received) >= pool.config.maxReceivedPoolSize {
		// 如果price太低，丢弃
		if pool.lowestPrice.GasPrice > tx.GasPrice {
			//log.Trace("Discarding underpriced transaction", "hash", hash, "price", tx.GasPrice())

			return false, ErrUnderpriced
		}

		pool.replace(tx)

	} else {
		pool.add(tx)

	}

	txs := *new([]*Transaction)
	txs = append(txs, tx)
	go BroadcastTransactions(txs)
	return true, nil
}

// 从池子里移除一批交易
func (pool *TransactionPool) Remove(transactions []common.Hash) {
	if nil == transactions || 0 == len(transactions) {
		return
	}

	for _, tx := range transactions {
		pool.remove(tx)
	}
}

// 从池子里移除某个交易
func (pool *TransactionPool) remove(hash common.Hash) {

	pool.received.Delete(hash)

}

func (pool *TransactionPool) GetTransactions(hashes []common.Hash) ([]*Transaction, []common.Hash, error) {
	if nil == hashes || 0 == len(hashes) {
		return nil, nil, ErrNil
	}

	txs := make([]*Transaction, 0)
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
func (pool *TransactionPool) GetTransaction(hash common.Hash) (*Transaction, error) {
	pool.lock.RLock("GetTransaction")
	defer pool.lock.RUnlock("GetTransaction")

	// 先从received里获取
	result, _ := pool.received.Load(hash)
	if nil != result {
		return result.(*Transaction), nil
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
	result, _ := pool.received.Load(hash)
	if result != nil {
		return true
	}

	existed, _ := pool.executed.Has(hash.Bytes())
	return existed
}

// 校验transaction是否合法
func (pool *TransactionPool) validate(tx *Transaction) error {
	if !tx.Hash.IsValid() {
		return ErrHash
	}

	return nil
}

// 直接加入未满的池子
// 不需要加锁，sync.Map保证并发
func (pool *TransactionPool) add(tx *Transaction) {
	pool.received.Store(tx.Hash, tx)
	lowestPrice := pool.lowestPrice
	if lowestPrice == nil || lowestPrice.GasPrice > tx.GasPrice {
		pool.lowestPrice = tx
	}

}

// 外部加锁
// 加缓冲区
func (pool *TransactionPool) addTxs(txs []*Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}
	for _, tx := range txs {
		pool.add(tx)
	}
}

// Add时候会调用，在外面加锁
func (pool *TransactionPool) replace(tx *Transaction) {
	// 替换
	pool.received.Delete(pool.lowestPrice.Hash)
	pool.received.Store(tx.Hash, tx)

	// 更新lowest
	lowest := tx
	pool.received.Range(func(key, value interface{}) bool {
		transaction := value.(*Transaction)
		if transaction.GasPrice < lowest.GasPrice {
			lowest = transaction
		}

		return true
	})

	pool.lowestPrice = lowest

}

// 外部加锁，AddExecuted通常和remove操作是依次执行的，所以由外部控制锁
func (pool *TransactionPool) AddExecuted(receipts types.Receipts, txs []*Transaction) {
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

func getTransaction(txs []*Transaction, hash []byte, i int) *Transaction {
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
func (pool *TransactionPool) RemoveExecuted(txs []*Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}
	for _, tx := range txs {
		pool.executed.Delete(tx.Hash.Bytes())
	}
}

func length(m *sync.Map) int {
	count := 0
	m.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}
