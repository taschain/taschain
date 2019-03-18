package core

import (
	"github.com/vmihailenco/msgpack"
	"common"
	"middleware/types"
	"bytes"
)

/*
**  Creator: pxf
**  Date: 2019/3/13 下午1:41
**  Description: 
*/

func generateTxKey(blockHash common.Hash, txHash common.Hash) []byte {
	buf := bytes.Buffer{}
	buf.Write(blockHash.Bytes())
	buf.Write(txHash.Bytes())
	return buf.Bytes()
}

func (pool *TxPool) saveTx(blockHash common.Hash, txHash common.Hash, dataBytes []byte) error {
	return pool.executed.AddKv(pool.batch, generateTxKey(blockHash, txHash), dataBytes)
}

func (pool *TxPool) saveReceipt(txHash common.Hash, dataBytes []byte) error {
	return pool.executed.AddKv(pool.batch, txHash.Bytes(), dataBytes)
}

func (pool *TxPool) MarkExecuted(blockHash common.Hash, receipts types.Receipts, txs []*types.Transaction, evictedTxs []common.Hash) error {
	if nil == receipts || 0 == len(receipts) {
		return nil
	}
	pool.lock.RLock("MarkExecuted")
	defer pool.lock.RUnlock("MarkExecuted")

	//先存交易
	for _, tx := range txs {
		txBytes, err := msgpack.Marshal(tx)
		if err != nil {
			return err
		}
		if err := pool.saveTx(blockHash, tx.Hash, txBytes); err != nil {
			return err
		}
	}

	//再存收据
	for _, receipt := range receipts {
		executedTx := &ReceiptStore{
			Receipt:     receipt,
			BlockHash: blockHash,
		}
		executedTxBytes, err := msgpack.Marshal(executedTx)
		if nil != err {
			return err
		}
		if err := pool.saveReceipt(receipt.TxHash, executedTxBytes); err != nil {
			return err
		}
		//pool.batch.Put(hash.Bytes(), executedTxBytes)
		//if pool.batch.ValueSize() > 100*1024 {
		//	pool.batch.Write()
		//	pool.batch.Reset()
		//}
	}
	//if pool.batch.ValueSize() > 0 {
	//	pool.batch.Write()
	//	pool.batch.Reset()
	//}

	for _, tx := range txs {
		pool.remove(tx.Hash)
	}
	if evictedTxs != nil {
		for _, hash := range evictedTxs {
			pool.remove(hash)
		}
	}
	return nil
}

func (pool *TxPool) UnMarkExecuted(blockHash common.Hash, txs []*types.Transaction) error {
	if nil == txs || 0 == len(txs) {
		return nil
	}
	pool.lock.RLock("UnMarkExecuted")
	defer pool.lock.RUnlock("UnMarkExecuted")
	var err error
	for _, tx := range txs {
		err = pool.saveTx(blockHash, tx.Hash, nil)
		if err != nil {
			return err
		}
		err = pool.saveReceipt(tx.Hash, nil)
		if err != nil {
			return err
		}
		//pool.executed.Delete(tx.Hash.Bytes())
		pool.add(tx, true)
	}
	return nil
}


func (pool *TxPool) GetTransactionStatus(hash common.Hash) (uint, error) {
	executedTx := pool.loadReceipt(hash)
	if executedTx == nil {
		return 0, ErrNil
	}
	return executedTx.Receipt.Status, nil
}

func (pool *TxPool) loadReceipt(hash common.Hash) *ReceiptStore {
	txBytes, _ := pool.executed.Get(hash.Bytes())
	if txBytes == nil {
		return nil
	}

	var rs ReceiptStore
	err := msgpack.Unmarshal(txBytes, &rs)
	if err != nil {
		return nil
	}
	return &rs
}

func (pool *TxPool) loadTransaction(blockHash common.Hash, hash common.Hash) *types.Transaction {
	txBytes, _ := pool.executed.Get(generateTxKey(blockHash, hash))
	if txBytes == nil {
		return nil
	}

	var tx types.Transaction
	err := msgpack.Unmarshal(txBytes, &tx)
	if err != nil {
		return nil
	}
	return &tx
}

func (pool *TxPool) GetReceipt(hash common.Hash) *types.Receipt {
	rs := pool.loadReceipt(hash)
	if rs == nil {
		return nil
	}
	return rs.Receipt
}

func (pool *TxPool) GetExecuted(hash common.Hash) *ExecutedTransaction {
	rs := pool.loadReceipt(hash)
	if rs == nil {
		return nil
	}
	tx := pool.loadTransaction(rs.BlockHash, hash)
	return &ExecutedTransaction{
		Receipt: rs.Receipt,
		Transaction: tx,
	}
}

func (pool *TxPool) GetTransactionsByBlockHash(hash common.Hash) []*types.Transaction {
	iter := pool.executed.NewIteratorWithPrefix(hash.Bytes())
	txs := make([]*types.Transaction, 0)
	for iter.Next() {
		var tx types.Transaction
		err := msgpack.Unmarshal(iter.Value(), &tx)
		if err == nil {
			txs = append(txs, &tx)
		}
	}
	return txs
}