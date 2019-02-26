package core

import (
	"fmt"
	"log"
	"middleware/types"
	"testing"
)

func TestCalTree(t *testing.T) {
	tx1 := getRandomTxs()
	tree1 := calcTxTree(tx1)
	log.Printf("tree1:%v", tree1.Hex())
	tree2 := calcTxTree(tx1)
	log.Printf("tree2:%v", tree2.Hex())

}

func getRandomTxs() []*types.Transaction {
	result := make([]*types.Transaction, 0)
	var i uint64
	for i = 0; i < 100; i++ {
		tx := types.Transaction{Nonce: i, Value: 100 - i}
		result = append(result, &tx)
	}
	return result
}

func TestHeap(t *testing.T) {
	con1 := newContainer(2)
	tx1 := &types.Transaction{
		GasPrice: 1,
		Value:    1,
	}
	tx2 := &types.Transaction{
		GasPrice: 1,
		Value:    2,
	}
	tx3 := &types.Transaction{
		GasPrice: 1,
		Value:    3,
	}
	con1.add(tx1)
	con1.add(tx2)
	con1.add(tx3)
	slice := con1.AsSlice()
	for _, tx := range slice {
		fmt.Println(tx)
	}

}

func TestConsistencyMark(t *testing.T) {
	fmt.Println("In")
	panic("Test panic!")
	fmt.Println("Out")
}
