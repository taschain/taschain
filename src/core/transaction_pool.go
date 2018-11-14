//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

import (
	"errors"
	"sync"
	"common"
	"os"
	"storage/tasdb"
	"middleware/types"
	"middleware"
	"github.com/hashicorp/golang-lru"
	"sort"
	"github.com/vmihailenco/msgpack"
	"time"
)

const (
	SendingListLength    = 50
	SendingTimerInterval = time.Second * 3
)

var (
	ErrNil = errors.New("nil transaction")

	ErrHash = errors.New("invalid transaction hash")

	ErrExist = errors.New("transaction already exist in pool")

	ErrMissingTrace = errors.New("trace not found")

	ErrInvalidSender = errors.New("invalid sender")

	ErrNonceTooLow = errors.New("nonce too low")

	ErrNonceTooHigh = errors.New("nonce too high")

	ErrUnderpriced = errors.New("transaction underpriced")

	ErrInsufficientFunds = errors.New("insufficient funds for gas * price + Value")

	ErrExisted = errors.New("executed transaction")

	ErrNegativeValue = errors.New("negative Value")

	ErrOversizedData = errors.New("oversized data")
)

// 配置文件
type TransactionPoolConfig struct {
	maxReceivedPoolSize int
	tx                  string
}

type TxPool struct {
	config *TransactionPoolConfig

	// 读写锁
	lock middleware.Loglock

	// 待上块的交易
	received    *container
	reserved    *lru.Cache
	sendingList []*types.Transaction

	sendingTxLock sync.Mutex
	sendingTimer  *time.Timer

	// 已经在块上的交易 key ：txhash Value： receipt
	executed tasdb.Database

	batch     tasdb.Batch
	batchLock sync.Mutex

	totalReceived uint64
}

type ReceiptWrapper struct {
	Receipt     *types.Receipt
	Transaction *types.Transaction
}

func DefaultPoolConfig() *TransactionPoolConfig {
	return &TransactionPoolConfig{
		maxReceivedPoolSize: 100000,
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

func NewTransactionPool() TransactionPool {
	pool := &TxPool{
		config:        getPoolConfig(),
		lock:          middleware.NewLoglock("txpool"),
		sendingList:   make([]*types.Transaction, 0),
		sendingTxLock: sync.Mutex{},
		batchLock:     sync.Mutex{},
		sendingTimer:  time.NewTimer(SendingTimerInterval),
	}
	pool.received = newContainer(pool.config.maxReceivedPoolSize)
	pool.reserved, _ = lru.New(pool.config.maxReceivedPoolSize / 4)

	executed, err := tasdb.NewDatabase(pool.config.tx)
	if err != nil {
		//todo: rebuild executedPool
		return nil
	}
	pool.executed = executed
	pool.batch = pool.executed.NewBatch()
	//go func() {
	//	for {
	//		<-pool.sendingTimer.C
	//		pool.CheckAndSend(true)
	//		pool.sendingTimer.Reset(SendingTimerInterval)
	//	}
	//}()
	return pool
}

func (pool *TxPool) AddTransaction(tx *types.Transaction) (bool, error) {
	pool.lock.Lock("AddTransaction")
	defer pool.lock.Unlock("AddTransaction")

	b, err := pool.addInner(tx, true)
	return b, err
}

// 不加锁
func (pool *TxPool) AddTransactions(txs []*types.Transaction) error {
	pool.lock.Lock("AddTransactions")
	defer pool.lock.Unlock("AddTransactions")

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

// 将一个合法的交易加入待处理队列。如果这个交易已存在，则丢掉
// 加锁
func (pool *TxPool) addInner(tx *types.Transaction, isBroadcast bool) (bool, error) {
	if tx == nil {
		return false, ErrNil
	}
	pool.totalReceived++
	// 简单规则校验
	if err := pool.validate(tx); err != nil {
		Logger.Debugf("Discarding invalid transaction,hash:%v,error:%v", tx.Hash, err)
		return false, err
	}

	// 检查交易是否已经存在
	hash := tx.Hash
	if pool.isTransactionExisted(hash) {
		//Logger.Debugf("Discarding already known transaction,hash:%s", hash.Hex())
		return false, ErrExist
	}

	pool.received.Push(tx)
	if tx.Type == types.TransactionTypeMinerApply {
		BroadcastMinerApplyTransactions([]*types.Transaction{tx})
	}
	// 日志记录分红交易信息

	// batch broadcast
	//if isBroadcast {
	//	//交易不广播
	//	pool.sendingTxLock.Lock()
	//	pool.sendingList = append(pool.sendingList, tx)
	//	pool.sendingTxLock.Unlock()
	//
	//	pool.CheckAndSend(false)
	//}

	return true, nil
}

func (pool *TxPool) CheckAndSend(immediately bool) {
	length := len(pool.sendingList)
	if immediately && length > 0 || SendingListLength <= length {
		pool.sendingTxLock.Lock()
		txs := pool.sendingList
		pool.sendingList = make([]*types.Transaction, 0)
		pool.sendingTxLock.Unlock()
		//Logger.Debugf("Broadcast txs,len:%d", len(txs))
		go BroadcastTransactions(txs)
	}
}

// 外部加锁，AddExecuted通常和remove操作是依次执行的，所以由外部控制锁
func (pool *TxPool) MarkExecuted(receipts types.Receipts, txs []*types.Transaction) {
	if nil == receipts || 0 == len(receipts) {
		return
	}

	go func() {
		pool.batchLock.Lock()
		defer pool.batchLock.Unlock()

		for i, receipt := range receipts {
			hash := receipt.TxHash.Bytes()
			receiptWrapper := &ReceiptWrapper{
				Receipt:     receipt,
				Transaction: getTransaction(txs, hash, i),
			}

			receiptJson, err := msgpack.Marshal(receiptWrapper)
			if nil != err {
				continue
			}
			pool.batch.Put(hash, receiptJson)

			if pool.batch.ValueSize() > 100*1024 {
				pool.batch.Write()
				pool.batch.Reset()
			}
		}

		if pool.batch.ValueSize() > 0 {
			pool.batch.Write()
			pool.batch.Reset()
		}
	}()
}

func getTransaction(txs []*types.Transaction, hash []byte, i int) *types.Transaction {
	if nil == txs || 0 == len(txs) {
		return nil
	}
	if common.ToHex(txs[i].Hash.Bytes()) == common.ToHex(hash) {
		return txs[i]
	}

	for _, tx := range txs {
		if common.ToHex(tx.Hash.Bytes()) == common.ToHex(hash) {
			return tx
		}
	}

	return nil
}

func (pool *TxPool) UnMarkExecuted(txs []*types.Transaction) {
	pool.lock.RLock("UnMarkExecuted")
	defer pool.lock.RUnlock("UnMarkExecuted")
	if nil == txs || 0 == len(txs) {
		return
	}
	for _, tx := range txs {
		pool.executed.Delete(tx.Hash.Bytes())
		pool.addInner(tx, false)
	}
}

//GeneratBlock
func (pool *TxPool) GetTransaction(hash common.Hash) (*types.Transaction, error) {
	pool.lock.RLock("GetTransaction")
	defer pool.lock.RUnlock("GetTransaction")

	return pool.getTransaction(hash)
}

func (pool *TxPool) GetTransactionStatus(hash common.Hash) (uint, error) {
	wrapper := pool.GetExecuted(hash)
	if wrapper == nil {
		return 0, ErrNil
	} else {
		return wrapper.Receipt.Status, nil
	}
}

func (pool *TxPool) getTransaction(hash common.Hash) (*types.Transaction, error) {
	//为解决交易池被打满后交易丢失无法GenereteBlock的情况
	reservedRaw, _ := pool.reserved.Get(hash)
	if nil != reservedRaw {
		reserved := reservedRaw.(*types.Transaction)
		return reserved, nil
	}
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
//验证组 verify
func (pool *TxPool) GetTransactions(reservedHash common.Hash, hashes []common.Hash) ([]*types.Transaction, []common.Hash, error) {
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

func getTx(reserved []*types.Transaction, hash common.Hash) (*types.Transaction, error) {
	if nil == reserved {
		return nil, nil
	}

	for _, tx := range reserved {
		if tx.Hash == hash {
			return tx, nil
		}
	}
	return nil, nil
}

func (pool *TxPool) Clear() {
	pool.lock.Lock("Clear")
	defer pool.lock.Unlock("Clear")

	os.RemoveAll(pool.config.tx)
	executed, _ := tasdb.NewDatabase(pool.config.tx)
	pool.executed = executed
	pool.batch.Reset()
	pool.received = newContainer(100000)
}

func (pool *TxPool) GetReceived() []*types.Transaction {
	return pool.received.txs
}

// 根据交易的hash判断交易是否存在：
// 1）已存在待处理列表
// 2）已存在块上
// 3）todo：曾经收到过的，不合法的交易
// 被add调用，外部加锁
func (pool *TxPool) isTransactionExisted(hash common.Hash) bool {
	result := pool.received.Contains(hash)
	if result {
		return true
	}

	existed, _ := pool.executed.Has(hash.Bytes())
	return existed
}

// 校验transaction是否合法
func (pool *TxPool) validate(tx *types.Transaction) error {
	if !tx.Hash.IsValid() {
		return ErrHash
	}

	return nil
}

// 从磁盘读取，不需要加锁（ldb自行保证）
func (pool *TxPool) GetExecuted(hash common.Hash) *ReceiptWrapper {
	receiptJson, _ := pool.executed.Get(hash.Bytes())
	if nil == receiptJson {
		return nil
	}

	var receipt *ReceiptWrapper
	//err := json.Unmarshal(receiptJson, &receipt)
	err := msgpack.Unmarshal(receiptJson, &receipt)
	if err != nil || receipt == nil {
		return nil
	}

	return receipt
}

func (p *TxPool) GetTotalReceivedTxCount() uint64 {
	return p.totalReceived
}

// 外部加锁
// 加缓冲区
func (pool *TxPool) AddTxs(txs []*types.Transaction) {
	pool.received.PushTxs(txs)
}

// 从池子里移除一批交易
func (pool *TxPool) Remove(hash common.Hash, transactions []common.Hash, evictedTxs []common.Hash) {
	pool.received.Remove(transactions)
	pool.received.Remove(evictedTxs)
	pool.reserved.Remove(hash)
	pool.removeFromSendinglist(transactions)
	pool.removeFromSendinglist(evictedTxs)

	for _, tx := range transactions {
		pool.reserved.Remove(tx)
	}
}

// 返回待处理的transaction数组
func (pool *TxPool) GetTransactionsForCasting() []*types.Transaction {
	txs := pool.received.AsSlice()
	sort.Sort(types.Transactions(txs))
	return txs
}

// 返回待处理的transaction数组
func (pool *TxPool) ReserveTransactions(hash common.Hash, txs []*types.Transaction) {
	if 0 == len(txs) {
		return
	}
	pool.reserved.Add(hash, txs)
	for _, tx := range txs {
		pool.reserved.Add(tx.Hash, tx)
	}
}

func (pool *TxPool) GetLock() *middleware.Loglock {
	return &pool.lock
}

func (p *TxPool) removeFromSendinglist(transactions []common.Hash) {
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
