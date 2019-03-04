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
	"container/heap"
	"sync"
	"middleware/types"
	"common"
)

type container struct {
	lock   sync.RWMutex
	txs    types.PriorityTransactions
	txsMap map[common.Hash]*types.Transaction
	limit  int
	inited bool
}

func newContainer(l int) *container {
	return &container{
		lock:   sync.RWMutex{},
		limit:  l,
		txsMap: map[common.Hash]*types.Transaction{},
		txs:    types.PriorityTransactions{},
		inited: false,
	}

}

func (c *container) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.txs)
}

func (c *container) Contains(key common.Hash) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.txsMap[key] != nil
}

func (c *container) Get(key common.Hash) *types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.txsMap[key]
}

func (c *container) AsSlice() []*types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var result []*types.Transaction
	len := c.txs.Len()
	if len > 3000 {
		result = make([]*types.Transaction, 3000)
		copy(result, c.txs[len-3000:])
	} else {
		result = make([]*types.Transaction, len)
		copy(result, c.txs)
	}

	return result
}

func (c *container) Push(tx *types.Transaction) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.add(tx)

}

func (c *container) PushTxs(txs []*types.Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	for _, tx := range txs {
		c.add(tx)
	}

}

func (c *container) add(tx *types.Transaction) {
	if !c.inited {
		heap.Init(&c.txs)
		c.inited = true
	}

	if c.txs.Len() < c.limit {
		//c.txs = append(c.txs, tx)
		heap.Push(&c.txs, tx)
		c.txsMap[tx.Hash] = tx
		return
	}

	//Logger.Debugf("tx pool size:%d great than max size,ignore tx!", c.txs.Len())
	heap.Push(&c.txs, tx)
	c.txsMap[tx.Hash] = tx
	evicted := heap.Pop(&c.txs).(*types.Transaction)
	delete(c.txsMap, evicted.Hash)
}

func (c *container) Remove(keys []common.Hash) {
	if nil == keys || 0 == len(keys) {
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	//Logger.Debugf("[Remove111:]tx pool container remove tx len:%d,contain tx map len %d,contain txs len %d",len(keys),len(c.txsMap),len(c.txs))
	//if len(keys) < 50 {
	for _, key := range keys {
		if c.txsMap[key] == nil {
			continue
		}
		delete(c.txsMap, key)
		//Logger.Debugf("txsMap delete Value contain tx map len %d",len(c.txsMap))

		index := -1
		for i, tx := range c.txs {
			if tx.Hash == key {
				index = i
				break
			}
		}
		heap.Remove(&c.txs, index)
	}
	//} else {
	//	queryMap := make(map[common.Hash]struct{})
	//	var num int
	//	for _, key := range keys {
	//		transaction := c.txsMap[key]
	//		if transaction != nil{
	//			transaction.Type = types.TransactionTypeToBeRemoved
	//			num++
	//			queryMap[key] = struct{}{}
	//		} else {
	//			for i := len(c.txs) - 1;i >= 0;i--{
	//				if c.txs[i].Hash == key {
	//					c.txs[i].Type = types.TransactionTypeToBeRemoved
	//					num++
	//					break
	//				}
	//			}
	//		}
	//		delete(c.txsMap, key)
	//	}
	//	heap.Init(&c.txs)
	//	toRemove := c.txs[:num]
	//	c.txs = c.txs[num:]
	//
	//	for _,item := range toRemove{
	//		if _,ok := queryMap[item.Hash];!ok{
	//			for i,tx := range c.txs{
	//				if _,ok := queryMap[tx.Hash];ok{
	//					c.txs[i] = item
	//					break
	//				}
	//			}
	//		}
	//	}
	//	heap.Init(&c.txs)
	//}
	//Logger.Debugf("[Remove111:]After remove,contain tx map len %d,contain txs len %d",len(c.txsMap),len(c.txs))
}
