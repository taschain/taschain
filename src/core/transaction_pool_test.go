////   Copyright (C) 2018 TASChain
////
////   This program is free software: you can redistribute it and/or modify
////   it under the terms of the GNU General Public License as published by
////   the Free Software Foundation, either version 3 of the License, or
////   (at your option) any later version.
////
////   This program is distributed in the hope that it will be useful,
////   but WITHOUT ANY WARRANTY; without even the implied warranty of
////   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
////   GNU General Public License for more details.
////
////   You should have received a copy of the GNU General Public License
////   along with this program.  If not, see <https://www.gnu.org/licenses/>.
//
package core

import (
	"common"
	"container/heap"
	"crypto/sha256"
	"fmt"
	"github.com/hashicorp/golang-lru"
	"math/rand"
	"middleware/types"
	"strconv"
	"testing"
	"time"
)

//
//import (
//	"testing"
//	"fmt"
//	"common"
//	"middleware/types"
//	"math/rand"
//)
//
//func TestCreatePool(t *testing.T) {
//
//	pool := NewTransactionPool()
//
//	fmt.Printf("received: %d transactions\n")
//
//	transaction := &types.Transaction{
//		GasPrice: 1234,
//	}
//
//	pool.AddTransaction(transaction)
//	fmt.Printf("received: %d transactions\n")
//
//	h := common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
//
//	transaction = &types.Transaction{
//		GasPrice: 12345,
//		Hash:     h,
//	}
//
//	pool.AddTransaction(transaction)
//	fmt.Printf("received: %d transactions\n")
//
//	tGet, error := pool.GetTransaction(h)
//	if nil == error {
//		fmt.Printf("GasPrice: %d\n", tGet.GasPrice)
//	}
//
//	casting := pool.GetTransactionsForCasting()
//	fmt.Printf("length for casting: %d\n", len(casting))
//
//	fmt.Printf("%d\n", casting[0])
//	//fmt.Printf("%d\n", casting[1])
//	//fmt.Printf("%d\n", casting[2].gasprice)
//	//fmt.Printf("%d\n", casting[3].gasprice)
//
//}
//
//func TestContainer(t *testing.T) {
//	pool := NewTransactionPool()
//	//pool.received.limit = 1
//	h := common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
//	e := common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b42")
//
//	transaction := &types.Transaction{
//		GasPrice: 1234,
//		Hash:     e,
//	}
//
//	pool.AddTransaction(transaction)
//	fmt.Printf("received: %d transactions\n")
//
//	transaction = &types.Transaction{
//		GasPrice: 12345,
//		Hash:     h,
//	}
//
//	pool.AddTransaction(transaction)
//	fmt.Printf("received: %d transactions\n")
//
//	tGet, error := pool.GetTransaction(h)
//	if nil == error {
//		fmt.Printf("%d\n", tGet.GasPrice)
//	}
//
//	tGet, _ = pool.GetTransaction(e)
//	if nil != tGet {
//		fmt.Printf("%d\n", tGet.GasPrice)
//	} else {
//		fmt.Printf("success %x\n", e)
//	}
//}
//
//func TestMaxTxsPerBlock(t *testing.T) {
//	pool := NewTransactionPool()
//	for i := 0; i < 100000; i++ {
//		transaction := &types.Transaction{
//			GasPrice: rand.Uint64(),
//			Nonce:    rand.Uint64(),
//		}
//
//		transaction.Hash = transaction.GenHash()
//		pool.AddTransaction(transaction)
//
//		//fmt.Printf("%d\n", pool.received.Len())
//	}
//
//	casting := pool.GetTransactionsForCasting()
//	fmt.Printf("length for casting: %d\n", len(casting))
//}

type TransactionPoolTest interface {
	Add(tx *types.Transaction) (bool, error)
}

const Count = 30

var Addresses []*common.Address

func (pool *TxPool) Add(tx *types.Transaction) (bool, error) {
	if tx.Type == types.TransactionTypeBonus {
		pool.bonPool.add(tx)
	} else {
		pool.received.push(tx)
	}

	return true, nil
}

func TestPush(t *testing.T) {

	hex1 := "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f71111111111111"
	hex2 := "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f73333333333333"
	hex3 := "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f71777777777777"
	hex4 := "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f70000000000000"
	hex5 := "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f72222222222222"
	hex6 := "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f79999999999999"
	hex7 := "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f73333333333333"
	hex8 := "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f7aaaaaaaaaaaaa"
	hex9 := "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f7fffffffffffff"
	hex0 := "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f7eeeeeeeeeeeee"

	addr1 := common.BytesToAddress(common.Hex2Bytes(hex1))
	addr2 := common.BytesToAddress(common.Hex2Bytes(hex2))
	addr3 := common.BytesToAddress(common.Hex2Bytes(hex3))
	addr4 := common.BytesToAddress(common.Hex2Bytes(hex4))
	addr5 := common.BytesToAddress(common.Hex2Bytes(hex5))
	addr6 := common.BytesToAddress(common.Hex2Bytes(hex6))
	addr7 := common.BytesToAddress(common.Hex2Bytes(hex7))
	addr8 := common.BytesToAddress(common.Hex2Bytes(hex8))
	addr9 := common.BytesToAddress(common.Hex2Bytes(hex9))
	addr0 := common.BytesToAddress(common.Hex2Bytes(hex0))

	Addresses = []*common.Address{&addr1, &addr2, &addr3, &addr4, &addr5, &addr6, &addr7, &addr8, &addr9, &addr0}

	cache, _ := lru.New(txCountPerBlock * blockResponseSize)
	pool := &TxPool{
		//broadcastTimer:  time.NewTimer(broadcastTimerInterval),
		//oldTxBroadTimer: time.NewTimer(oldTxBroadcastTimerInterval),
		asyncAdds: cache,
	}
	pool.received = newSimpleContainer(Count/2, Count/2)
	var hash common.Hash
	var txs []*types.Transaction
	var hashes []common.Hash
	var usedAddr []*common.Address

	for i := 0; i < Count*2; i++ {
		txhash := common.Hash(sha256.Sum256([]byte(strconv.Itoa(int(time.Now().Nanosecond())))))
		copy(hash[:], txhash[:32])
		tx := &types.Transaction{
			//GasPrice: uint64(rand.Intn(500000) + 1000),
			GasPrice: uint64(rand.Intn(5) + 1000),
			//GasPrice: 110,
			Hash: hash,
			Type: types.TransactionTypeContractCreate,
			Data: []byte(strconv.Itoa(i)),
			//Nonce:    uint64(i)+1,
			Nonce: uint64(rand.Intn(10)),
			//Nonce:    uint64(50-i),
			Source: Addresses[rand.Intn(8)],
		}
		txs = append(txs, tx)
		hashes = append(hashes, txhash)
		usedAddr = append(usedAddr, tx.Source)
		//for _,v := range usedAddr{
		//	if !bytes.Equal(v[:],tx.Source[:]) {
		//		usedAddr = append(usedAddr, tx.Source)
		//	}
		//}

	}
	fmt.Println(usedAddr)

	for i := 0; i < Count; i++ {
		pool.Add(txs[i])
	}

	///////////
	fmt.Println("ByPrice---------------------------")
	iter1 := pool.received.sortedTxsByPrice.IterAtPosition(0)
	for j := 0; iter1.Next() && j < pool.received.Len(); j++ {
		fmt.Printf("Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", iter1.Value().(*types.Transaction).Hash, iter1.Value().(*types.Transaction).GasPrice, iter1.Value().(*types.Transaction).Nonce, *iter1.Value().(*types.Transaction).Source)
	}
	///////////

	///////////
	fmt.Println("AllTxsMap---------------------------")
	for i := 0; i < len(hashes); i++ {
		if tx := pool.received.AllTxs[hashes[i]]; tx != nil {
			fmt.Printf("Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", hashes[i], tx.GasPrice, tx.Nonce, tx.Source)
		} else {
			fmt.Printf("Hash:%x,\t\n", hashes[i])
		}
	}
	///////////

	///////////
	var pendingTxsCount int
	for _, v := range pool.received.pending {
		pendingTxsCount += v.indexes.Len()
	}
	///////////

	///////////
	fmt.Println("GetFromPendingAll---------------------------")
	for _, v := range pool.received.pending {
		for v.indexes.Len() > 0 {
			tx := v.items[heap.Pop(v.indexes).(uint64)]
			fmt.Printf("Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", tx.Hash, tx.GasPrice, tx.Nonce, *tx.Source)
		}

	}
	//
	fmt.Println("CountOfPending---------------------------")
	fmt.Println(pendingTxsCount)
	///////////

	///////////
	// 从交易池取出交易
	// 将交易添加至新的切片
	//txsNew := make([]*types.Transaction, 0)
	//sortedList := skip.New(uint16(16))
	//for _, v := range pool.received.pending {
	//	if v.indexes.Len() >0 {
	//		nonce := heap.Pop(v.indexes).(uint64)
	//		v.cache = append(v.cache, nonce)
	//		fmt.Println(v.cache)
	//		fmt.Println(*v.indexes)
	//		tx:= v.items[nonce]
	//		sortedList.Insert(tx)
	//		pool.received.recoverFromCache()
	//	}
	//	fmt.Println(*v.indexes)
	//
	//
	//}
	//for i := sortedList.Len() ; i > 0; i--{
	//	tx := sortedList.IterAtPosition(i-1).Value().(*types.Transaction)
	//	txsNew = append(txsNew, tx)
	//}
	//fmt.Println(sortedList.Len())
	//fmt.Println(len(txsNew))
	//
	//
	//fmt.Println("GetFromPending---------------------------")
	//for _,value := range txsNew{
	//	fmt.Printf("Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", value.Hash, value.GasPrice,value.Nonce,*value.Source)
	//}
	///////////

	fmt.Println("GetFromQueue---------------------------")
	queueTxs := pool.received.queue
	for i := 0; i < len(queueTxs); i++ {
		fmt.Printf("Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", queueTxs[i].Hash, queueTxs[i].GasPrice, queueTxs[i].Nonce, *queueTxs[i].Source)

	}
	fmt.Println(len(queueTxs))

}
