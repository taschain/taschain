package core

import (
	"testing"
	"middleware/types"
	"log"
)

func TestCalTree(t *testing.T) {
	tx1 := getRandomTxs()
	tree1 := calcTxTree(tx1)
	log.Printf("tree1:%v",tree1.Hex())
	tree2 := calcTxTree(tx1)
	log.Printf("tree2:%v",tree2.Hex())

}


func getRandomTxs()[]*types.Transaction{
	result := make([]*types.Transaction,0)
	var i uint64
	for i=0;i<100;i++{
		tx := types.Transaction{Nonce:i,Value:100-i}
		result = append(result,&tx)
	}
	return result
}