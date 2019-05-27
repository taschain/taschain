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
	"github.com/Workiva/go-datastructures/slice/skip"
	"middleware/types"
	"sync"
)

type simpleContainer struct {
	limit        int
	pendingLimit int
	queueLimit   int

	//txs    types.PriorityTransactions
	sortedTxsByPrice *skip.SkipList
	pending          map[common.Address]*sortedTxsByNonce
	queue            map[common.Address]*sortedTxsByNonce

	AllTxs map[common.Hash]*types.Transaction

	lock sync.RWMutex
}

type nonceHeap []uint64

func (h nonceHeap) Len() int           { return len(h) }
func (h nonceHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h nonceHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *nonceHeap) Push(x interface{}) {
	*h = append(*h, x.(uint64))
}

func (h *nonceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type sortedTxsByNonce struct {
	items   map[uint64]*types.Transaction
	indexes *nonceHeap
}

func newSimpleContainer(lp, lq int) *simpleContainer {
	c := &simpleContainer{
		lock:         sync.RWMutex{},
		limit:        lp + lq,
		pendingLimit: lp,
		queueLimit:   lq,
		AllTxs:       make(map[common.Hash]*types.Transaction),
		pending:      make(map[common.Address]*sortedTxsByNonce),
		queue:        make(map[common.Address]*sortedTxsByNonce),

		sortedTxsByPrice: skip.New(uint16(16)),
	}
	return c
}

func (c *simpleContainer) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.AllTxs)
}

//func (c *simpleContainer) sort() {
//	c.lock.Lock()
//	defer c.lock.Unlock()
//
//	sort.Sort(c.txs)
//}

func (c *simpleContainer) contains(key common.Hash) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	_, ok := c.AllTxs[key]
	return ok
}

func (c *simpleContainer) get(key common.Hash) *types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.AllTxs[key]
}

func (c *simpleContainer) forEach(f func(tx *types.Transaction) bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var willPackedTx *types.Transaction
	sortedList := skip.New(uint16(16))

	// 计数器：记录每个地址已经取出来交易个数
	count := make(map[common.Address]int)

	for _, sortedTxs := range c.pending {
		nonce := uint64((*sortedTxs.indexes)[0])
		willPackedTx = sortedTxs.items[nonce]
		sortedList.Insert(willPackedTx)
		count[*willPackedTx.Source]++
	}

	for sortedList.Len() > 0 {
		tx := sortedList.IterAtPosition(sortedList.Len() - 1).Value().(*types.Transaction)

		// 把每个地址对应的pending中的交易添加到sortedList中：添加的是gas最大交易同一发起者的nonce最小的交易
		if count[*tx.Source] < c.pending[*tx.Source].indexes.Len() {
			nextTxNonce := uint64((*c.pending[*tx.Source].indexes)[count[*tx.Source]])
			nextTx := c.pending[*tx.Source].items[nextTxNonce]
			sortedList.Insert(nextTx)
			count[*tx.Source]++
		}

		if !f(tx) {
			count = nil
			break
		}
		sortedList.Delete(tx)
	}
}

func (c *simpleContainer) getTargetTxByPrice() *types.Transaction {
	// 得到可用的gas最低的交易（相同gas得到nonce最大的交易）
	var i uint64
	for ; i < c.sortedTxsByPrice.Len(); i++ {
		e := c.sortedTxsByPrice.ByPosition(i)
		targetTx := e.(*types.Transaction)

		pending := c.pending[*targetTx.Source]
		if pending != nil && pending.items[targetTx.Nonce] != nil {
			return targetTx
		} else {
			continue
		}
	}
	return nil
}

func (c *simpleContainer) push(tx *types.Transaction) {

	c.lock.Lock()
	defer c.lock.Unlock()

	if _, exist := c.AllTxs[tx.Hash]; exist {
		return
	}

	pendingTxsCount := c.getPendingTxsLen()

	// 清除空间
	if pendingTxsCount >= c.pendingLimit {

		// 得到pending 中 gas最低的交易（相同gas得到nonce最大的交易）
		targetTx := c.getTargetTxByPrice()
		if targetTx == nil {
			return
		}
		targetPrice := targetTx.GasPrice
		targetSource := *targetTx.Source
		oldPending := c.pending[targetSource]

		newPending, isExistInPending := c.pending[*tx.Source]

		// pending 中不存在
		if !isExistInPending {
			if tx.GasPrice <= targetPrice {
				c.insertQueue(tx)
				return
			}

			c.enEndPendingToQueue(oldPending)

			if len(oldPending.items) == 0 || oldPending.indexes.Len() == 0 {
				delete(c.pending, targetSource)
			}

		} else {
			find, oldTx := newPending.FindSameNonce(tx.Nonce)
			switch {
			case tx.Nonce < newPending.getMinNonce():
				if !newPending.isNonceContinuous(tx.Nonce) {
					return
				}
				c.enEndPendingToQueue(newPending)
			case tx.Nonce > newPending.getMaxNonce(0):

				c.insertQueue(tx)
				return
			case find && newPending != nil:

				if oldTx.GasPrice >= tx.GasPrice {
					return
				}
				c.replaceThroughPending(oldTx, tx)
				return
			}

		}
	}

	if pending := c.pending[*tx.Source]; pending != nil {
		old, exist := pending.items[tx.Nonce]
		if exist && old.GasPrice < tx.GasPrice {
			c.replaceThroughPending(old, tx)
		}
	}
	c.insertPending(tx)

}

func (c *simpleContainer) remove(key common.Hash) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.AllTxs[key] == nil {
		return
	}
	tx := c.AllTxs[key]
	delete(c.AllTxs, key)
	c.sortedTxsByPrice.Delete(tx)

	// from pending
	for _, nonce := range *c.pending[*tx.Source].indexes {
		//fmt.Print("\t>>>nonce:",nonce)
		//fmt.Print("\tindexLen:",c.pending[*tx.Source].indexes.Len())
		for j := 0; j < c.pending[*tx.Source].indexes.Len() && (*c.pending[*tx.Source].indexes)[j] == nonce; j++ {
			//fmt.Println("\t>>>j:",j)
			// 删除index的内容
			delete(c.pending[*tx.Source].items, nonce)

			heap.Remove(c.pending[*tx.Source].indexes, j)
		}
		break
	}

	if len(c.pending[*tx.Source].items) == 0 {
		delete(c.pending, *tx.Source)
	}

}

func (c *simpleContainer) replaceThroughPending(old, newTx *types.Transaction) {
	if _, exist := c.AllTxs[old.Hash]; !exist {
		return
	}

	delete(c.AllTxs, old.Hash)
	c.sortedTxsByPrice.Delete(old)

	c.pending[*old.Source].items[old.Nonce] = newTx
	c.AllTxs[newTx.Hash] = newTx
	c.sortedTxsByPrice.Insert(newTx)
}

func (c *simpleContainer) insertPending(newTx *types.Transaction) {
	addr := *newTx.Source
	nonce := newTx.Nonce
	if !c.pending[*newTx.Source].isNonceContinuous(newTx.Nonce) && c.pending[*newTx.Source] != nil {
		if newTx.Nonce > c.pending[*newTx.Source].getMaxNonce(0) {
			c.insertQueue(newTx)
		}
		return
	}

	if c.pending[*newTx.Source] == nil {
		c.pending[addr] = newSortedTxsByNonce()
		heap.Init(c.pending[addr].indexes)
	}

	heap.Push(c.pending[addr].indexes, nonce)
	c.pending[addr].items[nonce] = newTx

	if c.AllTxs[newTx.Hash] == nil {
		c.AllTxs[newTx.Hash] = newTx
		c.sortedTxsByPrice.Insert(newTx)
	}
}

func (c *simpleContainer) getPendingTxsLen() int {
	var pendingTxsLen int
	for _, v := range c.pending {
		pendingTxsLen += v.indexes.Len()
	}
	return pendingTxsLen
}

func (c *simpleContainer) insertQueue(newTx *types.Transaction) {

	if c.getQueueTxsLen() >= c.queueLimit {
		// if newTx source has appeared in queue, remove the max nonce
		if queue, isExistInQueue := c.queue[*newTx.Source]; isExistInQueue {
			if queue.getMinNonce() > newTx.Nonce {
				// 删除最大的 nonce
				maxNonce := queue.getMaxNonce(1)
				for i := 0; i < queue.indexes.Len(); i++ {
					if (*queue.indexes)[i] == maxNonce {
						// 删除index的内容
						heap.Remove(queue.indexes, i)
						break
					}
				}
				oldTx := queue.items[maxNonce]
				delete(queue.items, maxNonce)
				delete(c.AllTxs, oldTx.Hash)
				c.sortedTxsByPrice.Delete(oldTx)

			} else if find, _ := queue.FindSameNonce(newTx.Nonce); find {
				//  删除gas小的
				oldTx := queue.items[newTx.Nonce]
				if oldTx.GasPrice < newTx.GasPrice {
					// 新的替换旧的
					//c.replaceThroughQueue(oldTx, newTx)
					c.discardTxByQueue(oldTx)
				} else {
					return
				}
			} else {
				return
			}
		} else {
			return
		}
	}

	addr := *newTx.Source
	nonce := newTx.Nonce

	if c.queue[*newTx.Source] == nil {
		c.queue[addr] = newSortedTxsByNonce()
		heap.Init(c.queue[addr].indexes)
	} else {
		if find, targetTx := c.queue[*newTx.Source].FindSameNonce(newTx.Nonce); find {
			if targetTx.GasPrice < newTx.GasPrice {
				c.discardTxByQueue(targetTx)
			} else {
				return
			}
		}
	}

	if c.queue[*newTx.Source] == nil {
		c.queue[addr] = newSortedTxsByNonce()
		heap.Init(c.queue[addr].indexes)
	}

	if c.AllTxs[newTx.Hash] == nil {
		c.sortedTxsByPrice.Insert(newTx)
		c.AllTxs[newTx.Hash] = newTx
	}
	heap.Push(c.queue[addr].indexes, nonce)
	c.queue[addr].items[nonce] = newTx
}

// TODO 查找其它需要调用这个方法的地方--lei
func (c *simpleContainer) promoteQueueToPending() {
	c.lock.Lock()
	defer c.lock.Unlock()

	var accounts []common.Address
	for addr := range c.queue {
		accounts = append(accounts, addr)
	}

	ready := make([]*types.Transaction, 0)

	for _, addr := range accounts {
		if queue := c.queue[addr]; queue != nil {
			for next := (*queue.indexes)[0]; queue.indexes.Len() > 0 && (*queue.indexes)[0] == next; next++ {
				ready = append(ready, queue.items[next])
				delete(queue.items, next)
				heap.Pop(queue.indexes)
			}
		}
	}

	for i := 0; i < len(ready) && c.getPendingTxsLen() < c.pendingLimit; i++ {
		c.insertPending(ready[i])
	}
}

func (c *simpleContainer) getQueueTxsLen() int {
	var queueTxsLen int
	for _, v := range c.queue {
		queueTxsLen += v.indexes.Len()
	}
	return queueTxsLen
}

func (c *simpleContainer) discardTxByQueue(tx *types.Transaction) {

	for i := 0; i < c.queue[*tx.Source].indexes.Len(); i++ {
		if (*c.queue[*tx.Source].indexes)[i] == tx.Nonce {
			// 删除index的内容
			heap.Remove(c.queue[*tx.Source].indexes, i)
			break
		}
	}
	delete(c.queue[*tx.Source].items, tx.Nonce)
	delete(c.AllTxs, tx.Hash)
	c.sortedTxsByPrice.Delete(tx)

	if c.queue[*tx.Source].indexes.Len() == 0 {
		delete(c.queue, *tx.Source)
	}

}

func (c *simpleContainer) enEndPendingToQueue(pending *sortedTxsByNonce) {
	removedNonce := pending.getMaxNonce(0)
	removedTx := pending.items[removedNonce]
	for i := 0; i < pending.indexes.Len(); i++ {
		if (*pending.indexes)[i] == removedNonce {
			// 删除index的内容
			heap.Remove(pending.indexes, i)
			break
		}
	}

	delete(pending.items, removedNonce)
	c.insertQueue(removedTx)
}

func newSortedTxsByNonce() *sortedTxsByNonce {
	s := &sortedTxsByNonce{
		items:   make(map[uint64]*types.Transaction),
		indexes: new(nonceHeap),
	}
	return s
}

func (s *sortedTxsByNonce) isNonceContinuous(nonce uint64) bool {
	if s != nil {
		maxNonce := s.getMaxNonce(0)
		minNonce := s.getMinNonce()
		if nonce != minNonce-1 && nonce != maxNonce+1 {
			return false
		}
	}
	return true
}

func (s *sortedTxsByNonce) getMaxNonce(from int) uint64 {

	// case = 1  from queue
	// default   from pending
	switch from {
	case 1:
		var maxNonce uint64
		for nonce := range s.items {
			if nonce > maxNonce {
				maxNonce = nonce
			}
		}
		return maxNonce
	default:
		return s.getMinNonce() + uint64(s.indexes.Len()) - 1
	}
}

func (s *sortedTxsByNonce) getMinNonce() uint64 {
	return (*s.indexes)[0]
}

func (s *sortedTxsByNonce) FindSameNonce(targetNonce uint64) (bool, *types.Transaction) {

	if _, ok := s.items[targetNonce]; !ok {
		return false, nil
	}
	return true, s.items[targetNonce]
}

//func (s *sortedTxsByNonce) removeEnd() (bool, *types.Transaction, []*types.Transaction) {
//	removedNonce := s.getMaxNonce(0)
//	discardTx := s.items[removedNonce]
//
//	for i := 0; i < s.indexes.Len(); i++ {
//		if (*s.indexes)[i] == removedNonce {
//			// 删除index的内容
//			heap.Remove(s.indexes, i)
//			break
//		}
//	}
//
//	delete(s.items, removedNonce)
//	return true, discardTx, s.filter(func(tx *types.Transaction) bool { return tx.Nonce > removedNonce })
//}
//
//func (s *sortedTxsByNonce) filter(compare func(*types.Transaction) bool) []*types.Transaction {
//	var removed []*types.Transaction
//	if len(s.items) == 0 && s.indexes.Len() == 0 {
//		return nil
//	}
//
//	for nonce, tx := range s.items {
//		if compare(tx) {
//			removed = append(removed, tx)
//			// 从 items 中删除
//			delete(s.items, nonce)
//		}
//	}
//	// 更新堆
//	if len(removed) > 0 {
//		*s.indexes = make([]uint64, 0, len(s.items))
//		for nonce := range s.items {
//			*s.indexes = append(*s.indexes, nonce)
//		}
//		// 更新index堆
//		heap.Init(s.indexes)
//	}
//	return removed
//}
