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
	"fmt"
	"sort"
	"sync"
	"time"

	"common"
	"middleware"
	"middleware/types"
	"storage/tasdb"

	"github.com/hashicorp/golang-lru"
	"github.com/vmihailenco/msgpack"
)

const (
	txDataBasePrefix = "tx"

	rcvTxPoolSize    = 50000
	minerTxCacheSize = 1000
	missTxCacheSize  = 60000

	broadcastListLength         = 50
	broadcastTimerInterval      = time.Second * 3
	oldTxBroadcastTimerInterval = time.Second * 30
	oldTxInterval               = time.Minute * 1

	txCountPerBlock = 1000
	gasLimitMax     = 500000
)

var (
	ErrNil = errors.New("nil transaction")

	ErrHash = errors.New("invalid transaction hash")

	ErrExist = errors.New("transaction already exist in pool")
)

type TxPool struct {
	minerTxs *lru.Cache // miner and bonus tx
	missTxs  *lru.Cache
	received *simpleContainer

	executed tasdb.Database
	batch    tasdb.Batch

	broadcastList   []*types.Transaction
	broadcastTxLock sync.Mutex
	broadcastTimer  *time.Timer

	txRcvTime       *lru.Cache
	oldTxBroadTimer *time.Timer

	txCount uint64
	lock    middleware.Loglock
}

func NewTransactionPool() TransactionPool {
	pool := &TxPool{
		broadcastList:   make([]*types.Transaction, 0, broadcastListLength),
		broadcastTxLock: sync.Mutex{},
		broadcastTimer:  time.NewTimer(broadcastTimerInterval),
		oldTxBroadTimer: time.NewTimer(oldTxBroadcastTimerInterval),
		lock:            middleware.NewLoglock("txPool"),
	}
	pool.received = newSimpleContainer(rcvTxPoolSize)
	pool.minerTxs, _ = lru.New(minerTxCacheSize)
	pool.missTxs, _ = lru.New(missTxCacheSize)
	pool.txRcvTime, _ = lru.New(rcvTxPoolSize)

	executed, err := tasdb.NewDatabase(txDataBasePrefix)
	if err != nil {
		Logger.Errorf("Init transaction pool error! Error:%s", err.Error())
		return nil
	}
	pool.executed = executed
	pool.batch = pool.executed.NewBatch()
	go pool.loop()
	return pool
}

func (pool *TxPool) AddTransaction(tx *types.Transaction) (bool, error) {
	if err := pool.verifyTransaction(tx); err != nil {
		Logger.Debugf("Tx %s verify sig error:%s, tx type:%d", tx.Hash.String(), err.Error(), tx.Type)
		return false, err
	}
	pool.lock.Lock("AddTransaction")
	defer pool.lock.Unlock("AddTransaction")

	b, err := pool.add(tx, !types.IsTestTransaction(tx))
	return b, err
}

func (pool *TxPool) AddBroadcastTransactions(txs []*types.Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}
	pool.lock.Lock("AddBroadcastTransactions")
	defer pool.lock.Unlock("AddBroadcastTransactions")

	for _, tx := range txs {
		if err := pool.verifyTransaction(tx); err != nil {
			Logger.Debugf("Tx %s verify sig error:%s, tx type:%d", tx.Hash.String(), err.Error(), tx.Type)
			continue
		}
		pool.add(tx, false)
	}
}

func (pool *TxPool) AddMissTransactions(txs []*types.Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}
	for _, tx := range txs {
		pool.missTxs.Add(tx.Hash, tx)
	}
	return
}

func (pool *TxPool) MarkExecuted(receipts types.Receipts, txs []*types.Transaction) {
	if nil == receipts || 0 == len(receipts) {
		return
	}
	pool.lock.RLock("MarkExecuted")
	defer pool.lock.RUnlock("MarkExecuted")

	for i, receipt := range receipts {
		hash := receipt.TxHash
		executedTx := &ExecutedTransaction{
			Receipt:     receipt,
			Transaction: findTxInList(txs, hash, i),
		}
		executedTxBytes, err := msgpack.Marshal(executedTx)
		if nil != err {
			continue
		}
		pool.batch.Put(hash.Bytes(), executedTxBytes)
		if pool.batch.ValueSize() > 100*1024 {
			pool.batch.Write()
			pool.batch.Reset()
		}
	}
	if pool.batch.ValueSize() > 0 {
		pool.batch.Write()
		pool.batch.Reset()
	}

	for _, tx := range txs {
		pool.remove(tx.Hash)
	}
}

func (pool *TxPool) UnMarkExecuted(txs []*types.Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}
	pool.lock.RLock("UnMarkExecuted")
	defer pool.lock.RUnlock("UnMarkExecuted")

	for _, tx := range txs {
		pool.executed.Delete(tx.Hash.Bytes())
		pool.add(tx, true)
	}
}

func (pool *TxPool) GetTransaction(hash common.Hash) (*types.Transaction, error) {
	minerTx, existInMinerTxs := pool.minerTxs.Get(hash)
	if existInMinerTxs {
		return minerTx.(*types.Transaction), nil
	}

	receivedTx := pool.received.get(hash)
	if nil != receivedTx {
		return receivedTx, nil
	}

	missTx, existInMissTxs := pool.missTxs.Get(hash)
	if existInMissTxs {
		return missTx.(*types.Transaction), nil
	}

	executedTx := pool.GetExecuted(hash)
	if nil != executedTx {
		return executedTx.Transaction, nil
	}
	return nil, ErrNil
}

func (pool *TxPool) GetTransactionStatus(hash common.Hash) (uint, error) {
	executedTx := pool.GetExecuted(hash)
	if executedTx == nil {
		return 0, ErrNil
	}
	return executedTx.Receipt.Status, nil
}

func (pool *TxPool) Clear() {
	pool.lock.Lock("Clear")
	defer pool.lock.Unlock("Clear")

	executed, _ := tasdb.NewDatabase(txDataBasePrefix)
	pool.executed = executed
	pool.batch.Reset()

	pool.received = newSimpleContainer(rcvTxPoolSize)
	pool.minerTxs, _ = lru.New(minerTxCacheSize)
	pool.missTxs, _ = lru.New(missTxCacheSize)
}

func (pool *TxPool) GetReceived() []*types.Transaction {
	return pool.received.txs
}

func (pool *TxPool) GetExecuted(hash common.Hash) *ExecutedTransaction {
	txBytes, _ := pool.executed.Get(hash.Bytes())
	if txBytes == nil {
		return nil
	}

	var executedTx *ExecutedTransaction
	err := msgpack.Unmarshal(txBytes, &executedTx)
	if err != nil || executedTx == nil {
		return nil
	}
	return executedTx
}

func (p *TxPool) TxNum() uint64 {
	return p.txCount
}

func (pool *TxPool) PackForCast() []*types.Transaction {
	minerTxs := pool.packMinerTx()
	if len(minerTxs) >= txCountPerBlock {
		return minerTxs
	}
	result := pool.packTx(minerTxs)
	return result
}

func (pool *TxPool) verifyTransaction(tx *types.Transaction) error {
	if !tx.Hash.IsValid() {
		return ErrHash
	}

	if tx.Hash != tx.GenHash() {
		return fmt.Errorf("tx hash error")
	}

	if tx.Sign == nil {
		return fmt.Errorf("tx sign nil")
	}

	if tx.Type != types.TransactionTypeBonus && tx.GasPrice == 0 {
		return fmt.Errorf("illegal tx gasPrice")
	}

	if tx.Type != types.TransactionTypeBonus && tx.GasLimit > gasLimitMax {
		return fmt.Errorf("gasLimit too  big! max gas limit is 500000 Ra")
	}

	if tx.Type == types.TransactionTypeBonus {
		if ok, err := BlockChainImpl.GetConsensusHelper().VerifyBonusTransaction(tx); !ok {
			return err
		}
	} else {
		if err := pool.verifySign(tx); err != nil {
			return err
		}
	}
	return nil
}

func (pool *TxPool) verifySign(tx *types.Transaction) error {
	hashByte := tx.Hash.Bytes()
	pk, err := tx.Sign.RecoverPubkey(hashByte)
	if err != nil {
		return err
	}
	if !pk.Verify(hashByte, tx.Sign) {
		return fmt.Errorf("verify sign fail, hash=%v", tx.Hash.Hex())
	}
	return nil
}

func (pool *TxPool) add(tx *types.Transaction, broadcast bool) (bool, error) {
	if tx == nil {
		return false, ErrNil
	}

	hash := tx.Hash
	if pool.isTransactionExisted(hash) {
		return false, ErrExist
	}
	pool.txCount++

	if tx.Type == types.TransactionTypeMinerApply || tx.Type == types.TransactionTypeMinerAbort ||
		tx.Type == types.TransactionTypeBonus || tx.Type == types.TransactionTypeMinerRefund {

		if tx.Type == types.TransactionTypeMinerApply {
			Logger.Debugf("Add TransactionTypeMinerApply,hash:%s,", tx.Hash.String())
		}
		pool.minerTxs.Add(tx.Hash, tx)
	} else {
		pool.received.push(tx)
	}
	pool.txRcvTime.Add(tx.Hash, time.Now())

	if broadcast {
		pool.broadcastTxLock.Lock()
		pool.broadcastList = append(pool.broadcastList, tx)
		pool.broadcastTxLock.Unlock()
		pool.checkAndBroadcastTx(false)
	}
	return true, nil
}

func (pool *TxPool) remove(txHash common.Hash) {
	pool.minerTxs.Remove(txHash)
	pool.missTxs.Remove(txHash)
	pool.received.remove(txHash)
	pool.txRcvTime.Remove(txHash)
	pool.removeFromBroadcastList(txHash)
}

func (pool *TxPool) removeFromBroadcastList(txHash common.Hash) {
	pool.broadcastTxLock.Lock()
	defer pool.broadcastTxLock.Unlock()
	for i, tx := range pool.broadcastList {
		if tx.Hash == txHash {
			pool.broadcastList = append(pool.broadcastList[:i], pool.broadcastList[i+1:]...)
		}
		break
	}
}

func (pool *TxPool) checkAndBroadcastTx(immediately bool) {
	length := len(pool.broadcastList)
	if immediately && length > 0 || length >= broadcastListLength {
		pool.broadcastTxLock.Lock()
		txs := pool.broadcastList
		pool.broadcastList = make([]*types.Transaction, 0, broadcastListLength)
		pool.broadcastTxLock.Unlock()
		go BroadcastTransactions(txs)
	}
}

func (pool *TxPool) isTransactionExisted(hash common.Hash) bool {
	existInMinerTxs := pool.minerTxs.Contains(hash)
	if existInMinerTxs {
		return true
	}

	existInReceivedTxs := pool.received.contains(hash)
	if existInReceivedTxs {
		return true
	}

	isExecutedTx, _ := pool.executed.Has(hash.Bytes())
	return isExecutedTx
}

func findTxInList(txs []*types.Transaction, txHash common.Hash, receiptIndex int) *types.Transaction {
	if nil == txs || 0 == len(txs) {
		return nil
	}
	if txs[receiptIndex].Hash == txHash {
		return txs[receiptIndex]
	}

	for _, tx := range txs {
		if tx.Hash == txHash {
			return tx
		}
	}
	return nil
}

func (pool *TxPool) packMinerTx() []*types.Transaction {
	minerTxs := make([]*types.Transaction, 0, txCountPerBlock)
	minerTxHashes := pool.minerTxs.Keys()
	for _, minerTxHash := range minerTxHashes {
		if v, ok := pool.minerTxs.Get(minerTxHash); ok {
			minerTxs = append(minerTxs, v.(*types.Transaction))
			if v.(*types.Transaction).Type == types.TransactionTypeMinerApply {
				Logger.Debugf("pack miner apply tx hash:%s,", v.(*types.Transaction).Hash.String())
			}
		}
		if len(minerTxs) >= txCountPerBlock {
			return minerTxs
		}
	}
	return minerTxs
}

func (pool *TxPool) packTx(packedTxs []*types.Transaction) []*types.Transaction {
	txs := pool.received.asSlice()
	sort.Sort(types.Transactions(txs))
	for _, tx := range txs {
		packedTxs = append(packedTxs, tx)
		if len(packedTxs) >= txCountPerBlock {
			return packedTxs
		}
	}
	return packedTxs
}

func (pool *TxPool) broadcastOldTx() {
	pool.broadcastTxLock.Lock()
	defer pool.broadcastTxLock.Unlock()

	if len(pool.broadcastList) > 0 {
		txs := pool.broadcastList
		pool.broadcastList = make([]*types.Transaction, 0, broadcastListLength)
		go BroadcastTransactions(txs)
	}

	txHashes := pool.txRcvTime.Keys()
	for _, txHash := range txHashes {
		if tx := pool.getOldTx(txHash.(common.Hash)); tx != nil {
			pool.broadcastList = append(pool.broadcastList, tx)
			if len(pool.broadcastList) >= broadcastListLength {
				break
			}
		}
	}
	txs := pool.broadcastList
	pool.broadcastList = make([]*types.Transaction, 0, broadcastListLength)
	go BroadcastTransactions(txs)
}

func (pool *TxPool) getOldTx(txHash common.Hash) *types.Transaction {
	now := time.Now()
	if v, ok := pool.txRcvTime.Get(txHash); ok {
		rcvTime := v.(time.Time)
		if now.Sub(rcvTime) > oldTxInterval {
			tx, _ := pool.GetTransaction(txHash)
			if tx != nil {
				return tx
			}
		}
	}
	return nil
}

func (pool *TxPool) loop() {
	for {
		select {
		case <-pool.broadcastTimer.C:
			pool.checkAndBroadcastTx(true)
			pool.broadcastTimer.Reset(broadcastTimerInterval)
		case <-pool.oldTxBroadTimer.C:
			pool.broadcastOldTx()
			Logger.Debugf("TxPool status: received:%d,minerTxPool:%d", len(pool.received.txs), pool.minerTxs.Len())
			pool.oldTxBroadTimer.Reset(oldTxBroadcastTimerInterval)
		}
	}
}
