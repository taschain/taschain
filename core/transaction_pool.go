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
	"github.com/hashicorp/golang-lru"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/notify"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/storage/tasdb"
	"sync"
)

const (
	maxTxPoolSize  = 50000
	bonusTxMaxSize = 1000
	//missTxCacheSize  = 60000

	//broadcastListLength         = 100
	//
	//oldTxBroadcastTimerInterval = time.Second * 30
	//oldTxInterval               = time.Second * 60

	txCountPerBlock = 3000
	gasLimitMax     = 500000

	txMaxSize = 64000 //每个交易最大大小
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

	asyncAdds *lru.Cache //异步添加的，加速块上链时validateTxs，不参与广播

	receiptdb *tasdb.PrefixedDatabase
	batch     tasdb.Batch

	chain BlockChain
	//broadcastList   []*types.Transaction
	//broadcastTxLock sync.Mutex
	//broadcastTimer  *time.Timer
	//
	//txBroadcastTime       *lru.Cache
	//oldTxBroadTimer *time.Timer

	gasPriceLowerBound uint64

	lock sync.RWMutex
}

type TxPoolAddMessage struct {
	txs   []*types.Transaction
	txSrc txSource
}

func (m *TxPoolAddMessage) GetRaw() []byte {
	panic("implement me")
}

func (m *TxPoolAddMessage) GetData() interface{} {
	panic("implement me")
}

func NewTransactionPool(chain *FullBlockChain, receiptdb *tasdb.PrefixedDatabase) TransactionPool {
	pool := &TxPool{
		//broadcastTimer:  time.NewTimer(broadcastTimerInterval),
		//oldTxBroadTimer: time.NewTimer(oldTxBroadcastTimerInterval),
		receiptdb:          receiptdb,
		batch:              chain.batch,
		asyncAdds:          common.MustNewLRUCache(txCountPerBlock * maxReqBlockCount),
		chain:              chain,
		gasPriceLowerBound: uint64(common.GlobalConf.GetInt("chain", "gasprice_lower_bound", 1)),
	}
	pool.received = newSimpleContainer(maxTxPoolSize)
	pool.bonPool = newBonusPool(chain.bonusManager, bonusTxMaxSize)
	initTxSyncer(chain, pool)

	return pool
}

func (pool *TxPool) tryAddTransaction(tx *types.Transaction, from txSource) (bool, error) {
	if err := pool.RecoverAndValidateTx(tx); err != nil {
		//Logger.Debugf("Tx verify sig error:%s, txRaw from %v, type:%d, txRaw %+v", err.Error(), from, txRaw.Type, txRaw)
		Logger.Debugf("tryAddTransaction err %v, from %v, hash %v, sign %v", err.Error(), from, tx.Hash.String(), tx.HexSign())
		return false, err
	} else {
		b, err := pool.tryAdd(tx)
		if err != nil {
			Logger.Debugf("tryAdd tx fail: from %v, hash=%v, type=%v, err=%v", from, tx.Hash.String(), tx.Type, err)
		}
		return b, err
	}
}

func (pool *TxPool) AddTransaction(tx *types.Transaction) (bool, error) {
	return pool.tryAddTransaction(tx, 0)
}

func (pool *TxPool) AddTransactions(txs []*types.Transaction, from txSource) {
	if nil == txs || 0 == len(txs) {
		return
	}
	//Logger.Debugf("add transactions size %v, txsrc %v", len(txs), from)
	for _, tx := range txs {
		pool.tryAddTransaction(tx, from)
	}
	notify.BUS.Publish(notify.TxPoolAddTxs, &TxPoolAddMessage{txs: txs, txSrc: from})
}

func (pool *TxPool) AsyncAddTxs(txs []*types.Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}
	for _, tx := range txs {
		if tx.Source != nil {
			continue
		}
		if tx.Type == types.TransactionTypeBonus {
			if pool.bonPool.get(tx.Hash) != nil {
				continue
			}
		} else {
			if pool.received.get(tx.Hash) != nil {
				continue
			}
		}
		if pool.asyncAdds.Contains(tx.Hash) {
			continue
		}
		if err := pool.RecoverAndValidateTx(tx); err == nil {
			pool.asyncAdds.Add(tx.Hash, tx)
			TxSyncer.add(tx)
		}
	}
}

func (pool *TxPool) GetTransaction(bonus bool, hash common.Hash) *types.Transaction {
	var tx = pool.bonPool.get(hash)
	if bonus || tx != nil {
		return tx
	}
	tx = pool.received.get(hash)
	if tx != nil {
		return tx
	}
	if v, ok := pool.asyncAdds.Get(hash); ok {
		return v.(*types.Transaction)
	}
	return nil
}

func (pool *TxPool) Clear() {
	pool.batch.Reset()
}

func (pool *TxPool) GetReceived() []*types.Transaction {
	return pool.received.asSlice(maxTxPoolSize)
}

func (pool *TxPool) TxNum() uint64 {
	return uint64(pool.received.Len() + pool.bonPool.len())
}

func (pool *TxPool) PackForCast() []*types.Transaction {
	result := pool.packTx()
	return result
}

func (pool *TxPool) RecoverAndValidateTx(tx *types.Transaction) error {
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

	var source *common.Address = nil
	if tx.Type == types.TransactionTypeBonus {
		if ok, err := BlockChainImpl.GetConsensusHelper().VerifyBonusTransaction(tx); !ok {
			return err
		}
	} else {
		if tx.GasPrice == 0 {
			return fmt.Errorf("illegal tx gasPrice")
		}
		if tx.GasLimit > gasLimitMax {
			return fmt.Errorf("gasLimit too  big! max gas limit is 500000 Ra")
		}
		var sign = common.BytesToSign(tx.Sign)
		if sign == nil {
			return fmt.Errorf("BytesToSign fail, sign=%v", tx.Sign)
		}
		msg := tx.Hash.Bytes()
		pk, err := sign.RecoverPubkey(msg)
		if err != nil {
			return err
		}
		src := pk.GetAddress()
		source = &src
		tx.Source = source

		//check nonce
		stateNonce := pool.chain.LatestStateDB().GetNonce(src)
		if !IsTestTransaction(tx) && (tx.Nonce <= stateNonce || tx.Nonce > stateNonce+1000) {
			return fmt.Errorf("nonce error:%v %v", tx.Nonce, stateNonce)
		}

		if !pk.Verify(msg, sign) {
			return fmt.Errorf("verify sign fail, hash=%v", tx.Hash.Hex())
		}
	}

	return nil
}

func (pool *TxPool) tryAdd(tx *types.Transaction) (bool, error) {
	if tx == nil {
		return false, ErrNil
	}
	pool.lock.Lock()
	defer pool.lock.Unlock()

	if exist, where := pool.isTransactionExisted(tx); exist {
		return false, fmt.Errorf("tx exist in %v", where)
	}

	pool.add(tx)

	return true, nil
}

func (pool *TxPool) add(tx *types.Transaction) (bool, error) {
	if tx.Type == types.TransactionTypeBonus {
		pool.bonPool.add(tx)
	} else {
		pool.received.push(tx)
	}
	TxSyncer.add(tx)

	return true, nil
}

func (pool *TxPool) remove(txHash common.Hash) {
	pool.bonPool.remove(txHash)
	pool.received.remove(txHash)
	pool.asyncAdds.Remove(txHash)
}

func (pool *TxPool) isTransactionExisted(tx *types.Transaction) (exists bool, where int) {
	if tx.Type == types.TransactionTypeBonus {
		if pool.bonPool.contains(tx.Hash) {
			return true, 1
		}
		//if pool.bonPool.hasBonus(tx.Data) {
		//	return true, 2
		//}
	} else {
		if pool.received.contains(tx.Hash) {
			return true, 1
		}
	}
	if pool.asyncAdds.Contains(tx.Hash) {
		return true, 2
	}

	if pool.hasReceipt(tx.Hash) {
		return true, 3
	}
	return false, -1
}

func (pool *TxPool) packTx() []*types.Transaction {
	txs := make([]*types.Transaction, 0, txCountPerBlock)
	pool.bonPool.forEach(func(tx *types.Transaction) bool {
		txs = append(txs, tx)
		return len(txs) < txCountPerBlock
	})
	if len(txs) < txCountPerBlock {
		for _, tx := range pool.received.asSlice(txCountPerBlock - len(txs)) {
			//gas price too low
			if tx.GasPrice < pool.gasPriceLowerBound {
				continue
			}
			txs = append(txs, tx)
			if len(txs) >= txCountPerBlock {
				break
			}
		}
	}
	return txs
}

func (pool *TxPool) RemoveFromPool(txs []common.Hash) {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	for _, tx := range txs {
		pool.remove(tx)
	}
}

func (pool *TxPool) BackToPool(txs []*types.Transaction) {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	for _, txRaw := range txs {
		if txRaw.Type != types.TransactionTypeBonus && txRaw.Source == nil {
			err := txRaw.RecoverSource()
			if err != nil {
				Logger.Errorf("backtopPool recover source fail:tx=%v", txRaw.Hash.String())
				continue
			}
		}
		pool.add(txRaw)
	}
}

func (pool *TxPool) GetBonusTxs() []*types.Transaction {
	txs := make([]*types.Transaction, 0)
	pool.bonPool.forEach(func(tx *types.Transaction) bool {
		txs = append(txs, tx)
		return true
	})
	return txs
}
