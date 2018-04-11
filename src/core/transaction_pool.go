package core

import (
	"errors"
	"fmt"
	"sync"
	"common"
)

var (
	ErrNil = errors.New("nil transaction")

	ErrHash = errors.New("invalid transaction hash")

	ErrInvalidSender = errors.New("invalid sender")

	ErrNonceTooLow = errors.New("nonce too low")

	ErrUnderpriced = errors.New("transaction underpriced")

	ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")

	ErrGasLimit = errors.New("exceeds block gas limit")

	ErrNegativeValue = errors.New("negative value")

	ErrOversizedData = errors.New("oversized data")
)

// 配置文件
type TransactionPoolConfig struct {
	maxReceivedPoolSize uint32
}

type TransactionPool struct {
	config *TransactionPoolConfig

	// 读写锁
	receivedLock sync.RWMutex

	// 收到的待处理transaction
	received map[common.Hash]*Transaction //todo: 替换成sync.Map 自带锁的map

	// 当前received数组里，price最小的transaction
	lowestPrice *Transaction
}

func DefaultPoolConfig() *TransactionPoolConfig {
	return &TransactionPoolConfig{
		maxReceivedPoolSize: 10000,
	}
}

func NewTransactionPool() *TransactionPool {

	return &TransactionPool{
		config:       DefaultPoolConfig(),
		receivedLock: sync.RWMutex{},
		received:     make(map[common.Hash]*Transaction),
		lowestPrice:  nil,
	}
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
		if pool.lowestPrice.Gasprice > tx.Gasprice {
			//log.Trace("Discarding underpriced transaction", "hash", hash, "price", tx.GasPrice())

			return false, ErrUnderpriced
		}

		pool.replace(tx)

	} else {
		pool.add(tx)

	}

	return true, nil
}

// 从池子里移除某个交易
func (pool *TransactionPool) Remove(tx *Transaction) (bool, error) {
	if tx == nil {
		return false, ErrNil
	}

	pool.receivedLock.Lock()
	defer pool.receivedLock.Unlock()
	delete(pool.received, tx.Hash)

	return true, nil

}

// 根据hash获取交易实例
// 如果本地没有，则需要从p2p网络中获取，此处不等待p2p网络的返回，而是直接返回error码
func (pool *TransactionPool) GetTransaction(hash common.Hash) (*Transaction, error) {
	pool.receivedLock.RLock()
	defer pool.receivedLock.RUnlock()

	result := pool.received[hash]
	if nil != result {
		return result, nil
	}

	//todo： 调用p2p模块，获取相应交易
	return nil, fmt.Errorf("get transaction: %x remotely", hash)
}

// 根据交易的hash判断交易是否存在：
// 1）已存在待处理列表
// 2）已存在块上
// 3）todo：曾经收到过的，不合法的交易
func (pool *TransactionPool) isTransactionExisted(hash common.Hash) bool {
	return pool.received[hash] != nil
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
	if lowestPrice == nil || lowestPrice.Gasprice > tx.Gasprice {
		pool.lowestPrice = tx
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
		if transaction.Gasprice < lowest.Gasprice {
			lowest = transaction
		}
	}

	pool.lowestPrice = lowest
	defer pool.receivedLock.RUnlock()

}
