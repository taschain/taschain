package core

import (
	"testing"
	"fmt"
	"common"
)

func TestCreatePool(t *testing.T) {

	pool := NewTransactionPool()

	fmt.Printf("received: %d transactions\n", length(pool.received))

	transaction := &Transaction{
		GasPrice: 1234,
	}

	pool.Add(transaction)
	fmt.Printf("received: %d transactions\n", length(pool.received))

	h := common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

	transaction = &Transaction{
		GasPrice: 12345,
		Hash:     h,
	}

	pool.Add(transaction)
	fmt.Printf("received: %d transactions\n", length(pool.received))

	tGet, error := pool.GetTransaction(h)
	if nil == error {
		fmt.Printf("%d\n", tGet.GasPrice)
	}

	casting := pool.GetTransactionsForCasting()
	fmt.Printf("%d\n", len(casting))

	fmt.Printf("%d\n", casting[0])
	fmt.Printf("%d\n", casting[1])
	//fmt.Printf("%d\n", casting[2].gasprice)
	//fmt.Printf("%d\n", casting[3].gasprice)

}
