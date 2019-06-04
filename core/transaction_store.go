package core

import (
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/types"
	"github.com/vmihailenco/msgpack"
)

func (pool *TxPool) saveReceipt(txHash common.Hash, dataBytes []byte) error {
	return pool.receiptdb.AddKv(pool.batch, txHash.Bytes(), dataBytes)
}

func (pool *TxPool) saveReceipts(bhash common.Hash, receipts types.Receipts) error {
	if nil == receipts || 0 == len(receipts) {
		return nil
	}
	for _, receipt := range receipts {
		executedTxBytes, err := msgpack.Marshal(receipt)
		if nil != err {
			return err
		}
		if err := pool.saveReceipt(receipt.TxHash, executedTxBytes); err != nil {
			return err
		}
	}
	return nil
}

func (pool *TxPool) deleteReceipts(txs []common.Hash) error {
	if nil == txs || 0 == len(txs) {
		return nil
	}
	var err error
	for _, tx := range txs {
		err = pool.saveReceipt(tx, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetTransactionStatus returns the execute result status by hash
func (pool *TxPool) GetTransactionStatus(hash common.Hash) (uint, error) {
	executedTx := pool.loadReceipt(hash)
	if executedTx == nil {
		return 0, ErrNil
	}
	return executedTx.Status, nil
}

func (pool *TxPool) loadReceipt(hash common.Hash) *types.Receipt {
	txBytes, _ := pool.receiptdb.Get(hash.Bytes())
	if txBytes == nil {
		return nil
	}

	var rs types.Receipt
	err := msgpack.Unmarshal(txBytes, &rs)
	if err != nil {
		return nil
	}
	return &rs
}

func (pool *TxPool) hasReceipt(hash common.Hash) bool {
	ok, _ := pool.receiptdb.Has(hash.Bytes())
	return ok
}

// GetReceipt returns the transaction's recipe by hash
func (pool *TxPool) GetReceipt(hash common.Hash) *types.Receipt {
	rs := pool.loadReceipt(hash)
	if rs == nil {
		return nil
	}
	return rs
}
