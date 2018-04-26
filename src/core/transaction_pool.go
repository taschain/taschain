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
)

var (
	ErrNil = errors.New("nil transaction")

	ErrHash = errors.New("invalid transaction hash")

	ErrInvalidSender = errors.New("invalid sender")

	ErrNonceTooLow = errors.New("nonce too low")

	ErrNonceTooHigh = errors.New("nonce too high")

	ErrUnderpriced = errors.New("transaction underpriced")

	ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")

	ErrGasLimit = errors.New("exceeds block gas limit")

	ErrNegativeValue = errors.New("negative value")

	ErrOversizedData = errors.New("oversized data")
)

// 配置文件
type TransactionPoolConfig struct {
	maxReceivedPoolSize uint32
	tx                  string
	txCache             int
	txHandle            int
}

type TransactionPool struct {
	config *TransactionPoolConfig

	// 读写锁
	receivedLock sync.RWMutex

	// 收到的待处理transaction
	received map[common.Hash]*Transaction //todo: 替换成sync.Map 自带锁的map

	// 当前received数组里，price最小的transaction
	lowestPrice *Transaction

	// 已经在块上的交易 key ：txhash value： receipt
	executed datasource.Database
}

func DefaultPoolConfig() *TransactionPoolConfig {
	return &TransactionPoolConfig{
		maxReceivedPoolSize: 10000,
		tx:                  "tx",
		txCache:             128,
		txHandle:            1024,
	}
}

func NewTransactionPool() *TransactionPool {

	pool := &TransactionPool{
		config:       DefaultPoolConfig(),
		receivedLock: sync.RWMutex{},
		received:     make(map[common.Hash]*Transaction),
		lowestPrice:  nil,
	}
	executed, err := datasource.NewLDBDatabase(pool.config.tx, pool.config.txCache, pool.config.txHandle)
	if err != nil {
		//todo: rebuild executedPool
		return nil
	}
	pool.executed = executed

	return pool
}

func (pool *TransactionPool) Clear() {
	pool.receivedLock.Lock()
	defer pool.receivedLock.Unlock()

	os.RemoveAll(pool.config.tx)
	executed, _ := datasource.NewLDBDatabase(pool.config.tx, pool.config.txCache, pool.config.txHandle)
	pool.executed = executed
	pool.received = make(map[common.Hash]*Transaction)
}

func (pool *TransactionPool) GetReceived() map[common.Hash]*Transaction {
	return pool.received
}

// 返回待处理的transaction数组
func (pool *TransactionPool) GetTransactionsForCasting() []*Transaction {
	pool.receivedLock.Lock()
	defer pool.receivedLock.Unlock()

	//pool.casting = make(map[common.Hash256]*Transaction)

	txs := make([]*Transaction, len(pool.received))
	i := 0
	for _, tx := range pool.received {
		txs[i] = tx
		i++
	}

	sort.Sort(Transactions(txs))
	return txs
}

// 将一个合法的交易加入待处理队列。如果这个交易已存在，则丢掉
// todo: 收到的交易是否需要广播出去？暂时先不广播
func (pool *TransactionPool) Add(tx *Transaction) (bool, error) {
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
	if uint32(len(pool.received)) >= pool.config.maxReceivedPoolSize {
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
	BroadcastTransactions(txs)
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

	pool.receivedLock.Lock()
	defer pool.receivedLock.Unlock()
	delete(pool.received, hash)

}

// 根据hash获取交易实例
// 如果本地没有，则需要从p2p网络中获取，此处不等待p2p网络的返回，而是直接返回error码
func (pool *TransactionPool) GetTransaction(hash common.Hash) (*Transaction, error) {
	pool.receivedLock.RLock()
	defer pool.receivedLock.RUnlock()

	// 先从received里获取
	result := pool.received[hash]
	if nil != result {
		return result, nil
	}

	// 再从executed里获取

	//todo： 调用p2p模块，获取相应交易
	return nil, fmt.Errorf("get transaction: %x remotely", hash)
}

// 根据交易的hash判断交易是否存在：
// 1）已存在待处理列表
// 2）已存在块上
// 3）todo：曾经收到过的，不合法的交易
func (pool *TransactionPool) isTransactionExisted(hash common.Hash) bool {
	if pool.received[hash] != nil {
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
func (pool *TransactionPool) add(tx *Transaction) {
	pool.receivedLock.Lock()
	defer pool.receivedLock.Unlock()

	pool.received[tx.Hash] = tx

	lowestPrice := pool.lowestPrice
	if lowestPrice == nil || lowestPrice.GasPrice > tx.GasPrice {
		pool.lowestPrice = tx
	}

}

func (pool *TransactionPool) addTxs(txs []*Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}
	for _, tx := range txs {
		pool.add(tx)
	}
}

func (pool *TransactionPool) replace(tx *Transaction) {
	// 替换
	pool.receivedLock.Lock()
	delete(pool.received, pool.lowestPrice.Hash)
	pool.received[tx.Hash] = tx
	pool.receivedLock.Unlock()

	// 更新lowest
	lowest := tx
	pool.receivedLock.RLock()

	for _, transaction := range pool.received {
		if transaction.GasPrice < lowest.GasPrice {
			lowest = transaction
		}
	}

	pool.lowestPrice = lowest
	defer pool.receivedLock.RUnlock()

}

func (pool *TransactionPool) AddExecuted(receipts types.Receipts) {
	if nil == receipts || 0 == len(receipts) {
		return
	}

	for _, receipt := range receipts {
		hash := receipt.TxHash.Bytes()
		receiptJson, err := json.Marshal(receipt)
		if nil != err {
			continue
		}
		pool.executed.Put(hash, receiptJson)
	}
}

func (pool *TransactionPool) GetExecuted(hash common.Hash) *types.Receipt {
	receiptJson, _ := pool.executed.Get(hash.Bytes())
	if nil == receiptJson {
		return nil
	}

	var receipt *types.Receipt
	err := json.Unmarshal(receiptJson, receipt)
	if err != nil || receipt == nil {
		return nil
	}

	return receipt
}

func (pool *TransactionPool) RemoveExecuted(txs []*Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}
	for _, tx := range txs {
		pool.executed.Delete(tx.Hash.Bytes())
	}
}
