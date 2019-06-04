package core

import (
	"github.com/taschain/taschain/middleware/types"
	"testing"
)

func TestCalTree(t *testing.T) {
	tx1 := getRandomTxs()
	tree1 := calcTxTree(tx1)

	if tree1.Hex() != "0x5a312281df4bd8dfbb4d4a94ad0bf44d01bb8cfced1206b90e21b4ca0568cdb1" {
		t.Errorf("mismatch, expect 0x5a312281df4bd8dfbb4d4a94ad0bf44d01bb8cfced1206b90e21b4ca0568cdb1 but got get %s ", tree1.Hex())
	}
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
