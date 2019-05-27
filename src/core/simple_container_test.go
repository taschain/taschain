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
	"common"
	"container/heap"
	"middleware/types"
	"reflect"
	"testing"
)

var container = newSimpleContainer(6, 2)

const testTxCountPerBlock = 3

var (
	source1 = "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f71111111111111"
	source2 = "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f71222222222222"
	source3 = "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f71333333333333"
	source4 = "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f74444444444444"
	source5 = "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f75555555555555"
	source0 = "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f00000000000000"

	addr1 = common.BytesToAddress(common.Hex2Bytes(source1))
	addr2 = common.BytesToAddress(common.Hex2Bytes(source2))
	addr3 = common.BytesToAddress(common.Hex2Bytes(source3))
	addr4 = common.BytesToAddress(common.Hex2Bytes(source4))
	addr5 = common.BytesToAddress(common.Hex2Bytes(source5))
	addr0 = common.BytesToAddress(common.Hex2Bytes(source0))

	tx1  = &types.Transaction{Hash: common.HexToHash("ab454fdea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7ff4"), Nonce: 4, GasPrice: 10000, Source: &addr0}
	tx2  = &types.Transaction{Hash: common.HexToHash("d3b14a7bab3c68e9369d0e433e5be9a514e843593f0f149cb0906e7bc085d88d"), Nonce: 4, GasPrice: 20000, Source: &addr1}
	tx3  = &types.Transaction{Hash: common.HexToHash("d1f1134223133d8ab88897b3ffc68c4797697b4e8603a7fd6a76722e3cc615ae"), Nonce: 1, GasPrice: 17000, Source: &addr2}
	tx4  = &types.Transaction{Hash: common.HexToHash("b4f213b67242f9439d62549fc128e98efe21b935b4a211b52b9b0b1812a57165"), Nonce: 3, GasPrice: 10000, Source: &addr3}
	tx5  = &types.Transaction{Hash: common.HexToHash("80aa134ea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7123"), Nonce: 5, GasPrice: 11000, Source: &addr0}
	tx6  = &types.Transaction{Hash: common.HexToHash("d3b14a7bab3c68e9369d0e433e5be9a514e843593f0f149cb0906e7bc085d31a"), Nonce: 6, GasPrice: 21000, Source: &addr1}
	tx7  = &types.Transaction{Hash: common.HexToHash("d1f1134223133d8ab88897b3ffc68c4797697b4e8603a7fd6a76722e3cc617fa"), Nonce: 2, GasPrice: 9000, Source: &addr2}
	tx8  = &types.Transaction{Hash: common.HexToHash("3761a47f2b6745f1fefff25d529d18bd92ca460892f929b749e3995c4baac2d2"), Nonce: 2, GasPrice: 10000, Source: &addr0}
	tx9  = &types.Transaction{Hash: common.HexToHash("6d0edf5dc9d37e79d248b0f31796cfed580604b4ca1bcdd5aa696da6765a6054"), Nonce: 3, GasPrice: 9000, Source: &addr0}
	tx10 = &types.Transaction{Hash: common.HexToHash("49892838a63742cc522ad7a8c8be0f4360b13e83062a808a042c0b65b1fa096a"), Nonce: 2, GasPrice: 11000, Source: &addr0}
	tx11 = &types.Transaction{Hash: common.HexToHash("e41fe4ff98d0fc7df69686e79fa920bdfad6180d5162ce5324863f580522980a"), Nonce: 4, GasPrice: 11000, Source: &addr0}
	tx12 = &types.Transaction{Hash: common.HexToHash("b57b9520513eac56dc83af561d606340b8ac041b97f1741ccd11fc9c0cc098bd"), Nonce: 5, GasPrice: 8000, Source: &addr4}
	tx13 = &types.Transaction{Hash: common.HexToHash("1a375c639553f66d0ae4316bde2fc82a7b04a688ec63df04d63ff7f2b8d467ca"), Nonce: 2, GasPrice: 10000, Source: &addr5}
	tx14 = &types.Transaction{Hash: common.HexToHash("ca1896f3507580ef6f3c43d76bb097540f9281c5529c968f3e8f7328276ffe11"), Nonce: 4, GasPrice: 21000, Source: &addr1}
	tx15 = &types.Transaction{Hash: common.HexToHash("ba2c2944f27aeaa03ef97b42909b43e0ead02cf08d0c20433dda1a2e8b3c2e5a"), Nonce: 2, GasPrice: 10000, Source: &addr5}
)

func Test_simpleContainer_push(t *testing.T) {

	txs := []*types.Transaction{
		tx1, tx2, tx3, tx4, tx5, tx6, tx7, tx8, tx9, tx10, tx11, tx12, tx13, tx14, tx15,
	}

	for _, tx := range txs {
		container.push(tx)
	}

	t.Run("ByPrice", func(t *testing.T) {

		idealTxs := []*types.Transaction{
			tx9, tx7, tx4, tx13, tx10, tx3, tx6, tx14,
		}

		iter1 := container.sortedTxsByPrice.IterAtPosition(0)
		for j := 0; iter1.Next() && j < container.Len(); j++ {
			//fmt.Printf("Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", iter1.Value().(*types.Transaction).Hash, iter1.Value().(*types.Transaction).GasPrice, iter1.Value().(*types.Transaction).Nonce, *iter1.Value().(*types.Transaction).Source)

			for _, tx := range idealTxs {
				if reflect.DeepEqual(iter1, tx) {
					t.Error("sortedTxsByPrice err")
				}
			}

		}
	})

	t.Run("AllTxsMap", func(t *testing.T) {

		idealTxs := []*types.Transaction{
			tx9, tx7, tx4, tx13, tx10, tx3, tx6, tx14,
		}

		//for k, v := range container.AllTxs {
		//	if v != nil {
		//		fmt.Printf("Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", k, v.GasPrice, v.Nonce, v.Source)
		//	} else {
		//		fmt.Printf("Hash:%x,\t not in txsmap\n", k)
		//	}
		//}

		if len(idealTxs) != len(container.AllTxs) {
			t.Error("allmaps length doesn't match")
		}

		for _, tx := range idealTxs {
			if _, ok := container.AllTxs[tx.Hash]; !ok {
				t.Error("txs doesn't match")
			}
		}

	})

	t.Run("GetFromPendingAll", func(t *testing.T) {

		idealTxs := []*types.Transaction{
			tx10, tx14, tx3, tx7, tx4, tx13,
		}

		tmp := make([]*types.Transaction, 0)

		for _, v := range container.pending {
			for v.indexes.Len() > 0 {
				tx := v.items[heap.Pop(v.indexes).(uint64)]
				tmp = append(tmp, tx)
				//fmt.Printf("Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", tx.Hash, tx.GasPrice, tx.Nonce, *tx.Source)
			}
		}
		for _, tx := range tmp {
			heap.Push(container.pending[*tx.Source].indexes, tx.Nonce)
			//fmt.Printf(">>>>>Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", tx.Hash, tx.GasPrice, tx.Nonce, *tx.Source)

		}

		count := 0
		for _, v1 := range tmp {
			for _, v2 := range idealTxs {
				if reflect.DeepEqual(v1, v2) {
					count++
					continue
				}
			}
		}

		if count != len(idealTxs) {
			t.Error("pending err")
		}
	})

	//t.Run("PopFromPendingAll---------------------------", func(t *testing.T) {
	//	for _, v := range container.pending {
	//		for v.indexes.Len() > 0 {
	//			tx := v.items[heap.Pop(v.indexes).(uint64)]
	//			fmt.Printf("Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", tx.Hash, tx.GasPrice, tx.Nonce, *tx.Source)
	//		}
	//	}
	//})

	t.Run("GetFromQueue", func(t *testing.T) {

		idealTxs := []*types.Transaction{
			tx6, tx9,
		}
		tmp2 := make([]*types.Transaction, 0)

		for _, v := range container.queue {
			for v.indexes.Len() > 0 {
				tx := v.items[heap.Pop(v.indexes).(uint64)]
				tmp2 = append(tmp2, tx)
				//fmt.Printf("Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", tx.Hash, tx.GasPrice, tx.Nonce, *tx.Source)
			}
		}
		for _, tx := range tmp2 {
			heap.Push(container.queue[*tx.Source].indexes, tx.Nonce)
		}

		count := 0
		for _, v1 := range tmp2 {
			for _, v2 := range idealTxs {
				if reflect.DeepEqual(v1, v2) {
					count++
					continue
				}
			}
		}

		if count != len(idealTxs) {
			t.Error("queue err")
		}
	})
}

func Test_simpleContainer_forEach(t *testing.T) {

	Test_simpleContainer_push(t)

	idealTxs := []*types.Transaction{
		tx14, tx3, tx10,
	}

	txsFromPending := make([]*types.Transaction, 0, testTxCountPerBlock)

	t.Run("forEachPending", func(t *testing.T) {
		container.forEach(func(tx *types.Transaction) bool {
			txsFromPending = append(txsFromPending, tx)
			return len(txsFromPending) < testTxCountPerBlock
		})
	})

	//for _, tx := range txsFromPending {
	//	fmt.Printf("Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", tx.Hash, tx.GasPrice, tx.Nonce, *tx.Source)
	//}

	if !reflect.DeepEqual(idealTxs, txsFromPending) {
		t.Error("foreach err，txs doesn't match")
	}

}

func Test_simpleContainer_promoteQueueToPending(t *testing.T) {

	Test_simpleContainer_remove(t)

	tmp := make([]*types.Transaction, 0)
	t.Run("promoteQueueToPending", func(t *testing.T) {

		idealTxs := []*types.Transaction{
			tx6, tx9,
		}

		container.promoteQueueToPending()
		for _, v := range container.pending {
			for v.indexes.Len() > 0 {
				tx := v.items[heap.Pop(v.indexes).(uint64)]
				tmp = append(tmp, tx)
				//fmt.Printf("Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", tx.Hash, tx.GasPrice, tx.Nonce, *tx.Source)
			}
		}
		//fmt.Println("lentmp", len(tmp))
		for _, tx := range tmp {
			heap.Push(container.pending[*tx.Source].indexes, tx.Nonce)
		}

		count := 0
		for _, v1 := range tmp {
			for _, v2 := range idealTxs {
				if reflect.DeepEqual(v1, v2) {
					count++
				}
			}
		}

		if count != len(idealTxs) {
			t.Error("promote queue to pending err, txs doesn't match")
		}
	})

}

func Test_simpleContainer_remove(t *testing.T) {
	Test_simpleContainer_push(t)
	t.Run("removeTxs", func(t *testing.T) {

		idealTxs := []*types.Transaction{
			tx10, tx14, tx3, tx7, tx4, tx13,
		}

		for i := 0; i < len(idealTxs); i++ {
			container.remove(idealTxs[i].Hash)
		}

		if container.getPendingTxsLen() != 0 {
			t.Error("remove failure")
		}
	})
}

func Test_simpleContainer_contains(t *testing.T) {
	Test_simpleContainer_push(t)
	t.Run("contains", func(t *testing.T) {

		idealTxs := []*types.Transaction{
			tx10, tx11,
		}

		if !container.contains(idealTxs[0].Hash) {
			t.Errorf("tx:%s should in map", common.Bytes2Hex(tx10.Hash[:]))
		}

		if container.contains(idealTxs[1].Hash) {
			t.Errorf("tx:%s shouldn't in map", common.Bytes2Hex(tx11.Hash[:]))
		}
	})
}

func Test_simpleContainer_get(t *testing.T) {
	Test_simpleContainer_push(t)

	t.Run("get", func(t *testing.T) {
		idealTx := tx10
		if !reflect.DeepEqual(idealTx, container.get(idealTx.Hash)) {
			t.Errorf("tx:%s can't get the tx in map", common.Bytes2Hex(idealTx.Hash[:]))
		}
	})

}
