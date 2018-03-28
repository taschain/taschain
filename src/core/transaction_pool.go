package core

type TransactionPool struct {
	pending []Transaction
}

func (pool *TransactionPool) getPendingTransactions() *[]Transaction {

	return &pool.pending
}