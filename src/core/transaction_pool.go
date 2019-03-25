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
	"sync"
	"common"
	"middleware/types"
	"storage/tasdb"
)

const (
	rcvTxPoolSize    = 50000
	bonusTxMaxSize = 1000
	//missTxCacheSize  = 60000

	//broadcastListLength         = 100
	//
	//oldTxBroadcastTimerInterval = time.Second * 30
	//oldTxInterval               = time.Second * 60

	txCountPerBlock = 3000
	gasLimitMax     = 500000

	txMaxSize 		= 64000	//每个交易最大大小
)

var (
	ErrNil = errors.New("nil transaction")

	ErrHash = errors.New("invalid transaction hash")

	ErrExist = errors.New("transaction already exist in pool")
)

type TxPool struct {
	//bonusTxs *lru.Cache // bonus tx
	bonPool *bonusPool
	//missTxs  *lru.Cache
	received *simpleContainer

	receiptdb *tasdb.PrefixedDatabase
	batch     tasdb.Batch

	//ticker 	*ticker.GlobalTicker
	//broadcastList   []*types.Transaction
	//broadcastTxLock sync.Mutex
	//broadcastTimer  *time.Timer
	//
	//txBroadcastTime       *lru.Cache
	//oldTxBroadTimer *time.Timer

	lock   sync.RWMutex
}


func NewTransactionPool(batch tasdb.Batch, receiptdb *tasdb.PrefixedDatabase, bm *BonusManager) TransactionPool {
	pool := &TxPool{
		//broadcastTimer:  time.NewTimer(broadcastTimerInterval),
		//oldTxBroadTimer: time.NewTimer(oldTxBroadcastTimerInterval),
		receiptdb: receiptdb,
		batch:     batch,
	}
	pool.received = newSimpleContainer(rcvTxPoolSize)
	pool.bonPool = newBonusPool(bm, bonusTxMaxSize)

	initBroadcastAget(pool)
	return pool
}

func (pool *TxPool) tryAddTransaction(tx *types.Transaction, from int) (bool, error) {
	Logger.Debugf("addtx begin hash=%v", tx.Hash.String())
	defer Logger.Debugf("addTx finish hash=%v", tx.Hash.String())
	if err := pool.verifyTransaction(tx); err != nil {
		//Logger.Debugf("Tx verify sig error:%s, tx from %v, type:%d, tx %+v", err.Error(), from, tx.Type, tx)
		Logger.Debugf("tryAddTransaction err %v, from %v, hash %v, sign %v", err.Error(), from, tx.Hash.String(), tx.Sign.GetHexString())
		return false, err
	} else {
		Logger.Debugf("tryAddTransaction success, from %v, hash %v, sign %v", from, tx.Hash.String(), tx.Sign.GetHexString())

	}
	Logger.Debugf("do addtx hash=%v", tx.Hash.String())
	b, err := pool.add(tx)
	return b, err
}

func (pool *TxPool) AddTransaction(tx *types.Transaction) (bool, error) {
	return pool.tryAddTransaction(tx, 0)
}

func (pool *TxPool) AddTransactions(txs []*types.Transaction, from int) {
	if nil == txs || 0 == len(txs) {
		return
	}
	for _, tx := range txs {
		pool.tryAddTransaction(tx, from)
	}
}


func (pool *TxPool) GetTransaction(hash common.Hash) (*types.Transaction) {
	bonusTx := pool.bonPool.get(hash)
	if bonusTx != nil {
		return bonusTx
	}

	receivedTx := pool.received.get(hash)
	if nil != receivedTx {
		return receivedTx
	}

	//executedTx := pool.GetExecuted(hash)
	//if nil != executedTx {
	//	return executedTx.Transaction, nil
	//}
	return nil
}

func (pool *TxPool) Clear() {
	pool.batch.Reset()
}

func (pool *TxPool) GetReceived() []*types.Transaction {
	return pool.received.asSlice(rcvTxPoolSize)
}

func (pool *TxPool) TxNum() uint64 {
	return uint64(pool.received.Len() + pool.bonPool.len())
}

func (pool *TxPool) PackForCast() []*types.Transaction {
	result := pool.packTx()
	return result
}

func (pool *TxPool) verifyTransaction(tx *types.Transaction) error {
	if !tx.Hash.IsValid() {
		return ErrHash
	}
	size := 0
	if tx.Data != nil {
		size += len(tx.Data)
	}
	if tx.ExtraData != nil {
		size += len(tx.ExtraData)
	}
	if size > txMaxSize {
		return fmt.Errorf("tx size(%v) should not larger than %v", size, txMaxSize)
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

func (pool *TxPool) add(tx *types.Transaction) (bool, error) {
	if tx == nil {
		return false, ErrNil
	}

	hash := tx.Hash
	if pool.isTransactionExisted(hash) {
		return false, ErrExist
	}

	if tx.Type == types.TransactionTypeBonus {
		pool.bonPool.add(tx)
	} else {
		pool.received.push(tx)
	}

	return true, nil
}

func (pool *TxPool) remove(txHash common.Hash) {
	pool.bonPool.remove(txHash)
	pool.received.remove(txHash)
}

func (pool *TxPool) isTransactionExisted(hash common.Hash) bool {
	existInMinerTxs := pool.bonPool.contains(hash)
	if existInMinerTxs {
		return true
	}

	existInReceivedTxs := pool.received.contains(hash)
	if existInReceivedTxs {
		return true
	}

	return pool.hasReceipt(hash)
}

func (pool *TxPool) packTx() []*types.Transaction {
	txs := make([]*types.Transaction, 0, txCountPerBlock)
	pool.bonPool.forEach(func(tx *types.Transaction) bool {
		txs = append(txs, tx)
		return len(txs) < txCountPerBlock
	})
	if len(txs) < txCountPerBlock {
		for _, tx := range pool.received.asSlice(txCountPerBlock-len(txs)) {
			txs = append(txs, tx)
			if len(txs) >= txCountPerBlock {
				break
			}
		}
	}
	return txs
}

func (pool *TxPool) RemoveFromPool(txs []common.Hash)  {
	for _, tx := range txs {
		pool.remove(tx)
	}
}

func (pool *TxPool) BackToPool(txs []*types.Transaction)  {
	for _, tx := range txs {
		pool.add(tx)
	}
}
