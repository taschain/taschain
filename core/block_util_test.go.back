package core

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/taschain/taschain/middleware/types"
	"log"
	"math/big"
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

//func TestHeap(t *testing.T) {
//	con1 := newContainer(2)
//	tx1 := &types.Transaction{
//		GasPrice: 1,
//		Value:    1,
//	}
//	tx2 := &types.Transaction{
//		GasPrice: 1,
//		Value:    2,
//	}
//	tx3 := &types.Transaction{
//		GasPrice: 1,
//		Value:    3,
//	}
//	con1.add(tx1)
//	con1.add(tx2)
//	con1.add(tx3)
//	slice := con1.AsSlice()
//	for _, tx := range slice {
//		fmt.Println(tx)
//	}
//
//}

func TestConsistencyMark(t *testing.T) {
	fmt.Println("In")
	panic("Test panic!")
	fmt.Println("Out")
}

func TestShrinkPV(t *testing.T) {

	max256, _ := big.NewInt(0).SetString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
	max192, _ := big.NewInt(0).SetString("2ffffffffffffffffffffffffffffffffffffffffffffffff", 16)
	rat256 := big.NewFloat(1).SetInt(max256)
	rat192 := big.NewFloat(1).SetInt(max192)
	t.Log(math.MaxBig256, max256)

	f, _ := rat256.Float64()
	t.Log(f)

	z := new(big.Float).Quo(rat256, rat192)
	ff, _ := z.Float64()
	t.Log(z, ff, uint64(ff))
	t.Log(z.Int64())
}
