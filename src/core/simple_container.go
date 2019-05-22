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

func (c *simpleContainer) push(tx *types.Transaction) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, exist := c.AllTxs[tx.Hash]; exist {
		return
	}

	pendingTxsCount := c.getPendingTxsLen()

	// 清除空间
	if pendingTxsCount >= c.pendingLimit {

		// 得到gas最低的交易（相同gas得到nonce最大的交易）
		e := c.sortedTxsByPrice.ByPosition(0)
		headTx := e.(*types.Transaction)
		headTxPrice := headTx.GasPrice
		headTxSource := *headTx.Source

		oldPending, _ := c.pending[headTxSource]
		pending, isExistInPending := c.pending[*tx.Source]

		// pending 中不存在
		if !isExistInPending {
			if tx.GasPrice <= headTxPrice {
				return
			}

			if ok, txs := oldPending.removeEnd(); ok && len(txs) > 0 {
				for _, txx := range txs {
					c.insertQueue(txx)

				}
			}
			if len(oldPending.items) == 0 {
				delete(c.pending, headTxSource)
			}

		} else {
			if tx.Nonce < pending.getMinNonce() {
				if !pending.isNonceContinuous(tx.Nonce) {
					return
				}
				if ok, txs := pending.removeEnd(); ok && len(txs) > 0 {
					for _, tx := range txs {
						c.insertQueue(tx)
					}
				}

			}

			if tx.Nonce > pending.getMaxNonce() {
				c.insertQueue(tx)
				c.AllTxs[tx.Hash] = tx
				c.sortedTxsByPrice.Insert(tx)
				return
			}

			if find, oldTx := pending.FindSameNonce(tx.Nonce); find {
				if oldTx.GasPrice >= tx.GasPrice {
					return
				}

				//pending.remove(oldTx)
				//
				//c.sortedTxsByPrice.Delete(oldTx)
				//delete(c.AllTxs, oldTx.Hash)

				c.replace(oldTx, tx)
			}
		}
	}

	if pending := c.pending[*tx.Source]; pending != nil {
		old, exist := pending.items[tx.Nonce]
		if exist && old.GasPrice < tx.GasPrice {
			c.replace(old, tx)
			return
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
	for i, nonce := range *c.pending[*tx.Source].indexes {
		if nonce == tx.Nonce {
			heap.Remove(c.pending[*tx.Source].indexes, i)
			delete(c.pending[*tx.Source].items, nonce)
		}
	}
}


func (c *simpleContainer) replace(old, newTx *types.Transaction) {
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
		if newTx.Nonce > c.pending[*newTx.Source].getMaxNonce() {
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

func newSortedTxsByNonce() *sortedTxsByNonce {
	s := &sortedTxsByNonce{
		items:   make(map[uint64]*types.Transaction),
		indexes: new(nonceHeap),
	}
	return s
}

func (c *simpleContainer) insertQueue(newTx *types.Transaction) {

	queueTxsCount := c.getQueueTxsLen()
	if queueTxsCount >= c.queueLimit {
		return
	}

	addr := *newTx.Source
	nonce := newTx.Nonce

	if c.queue[*newTx.Source] == nil {
		c.queue[addr] = newSortedTxsByNonce()
		heap.Init(c.queue[addr].indexes)
	}

	heap.Push(c.queue[addr].indexes, nonce)
	c.queue[addr].items[nonce] = newTx
	c.sortedTxsByPrice.Insert(newTx)
	c.AllTxs[newTx.Hash] = newTx
}

// TODO 查找其它需要调用这个方法的地方--lei
func (c *simpleContainer) promoteQueueToPending() {
	c.lock.Lock()
	defer c.lock.Unlock()

	var accounts []common.Address
	for addr := range c.queue {
		accounts = append(accounts, addr)
	}

	for _, addr := range accounts {
		queue := c.queue[addr]
		for i := 0; i < queue.indexes.Len(); i++ {
			//TODO:目前这代码有问题，要重写过
			iter := uint64((*queue.indexes)[i])
			c.insertPending(queue.items[iter])
			delete(queue.items, iter)
			heap.Pop(queue.indexes)
		}
	}
}

func (c *simpleContainer) getQueueTxsLen() int {
	var queueTxsLen int
	for _, v := range c.pending {
		queueTxsLen += v.indexes.Len()
	}
	return queueTxsLen
}

func (s *sortedTxsByNonce) isNonceContinuous(nonce uint64) bool {
	if s != nil {
		maxNonce := s.getMaxNonce()
		minNonce := s.getMinNonce()
		if nonce != minNonce-1 && nonce != maxNonce+1 {
			return false
		}
	}
	return true
}

func (s *sortedTxsByNonce) getMaxNonce() uint64 {
	var maxNonce uint64
	for nonce := range s.items {
		if nonce > maxNonce {
			maxNonce = nonce
		}
	}
	return maxNonce
}

func (s *sortedTxsByNonce) getMinNonce() uint64 {
	var minNonce uint64
	for nonce := range s.items {
		if nonce < minNonce {
			minNonce = nonce
		}
	}
	return minNonce
}

func (s *sortedTxsByNonce) FindSameNonce(targetNonce uint64) (bool, *types.Transaction) {

	if _, ok := s.items[targetNonce]; !ok {
		return false, nil
	}
	return true, s.items[targetNonce]
}

func (s *sortedTxsByNonce) removeEnd() (bool, []*types.Transaction) {
	removedNonce := heap.Remove(s.indexes, s.indexes.Len()-1).(uint64)
	return true, s.filter(func(tx *types.Transaction) bool { return tx.Nonce >= removedNonce })
}

func (s *sortedTxsByNonce) remove(tx *types.Transaction) (bool, []*types.Transaction) {

	nonce := tx.Nonce

	if _, ok := s.items[nonce]; !ok {
		return false, nil
	}
	for i := 0; i < s.indexes.Len(); i++ {
		if (*s.indexes)[i] == nonce {
			// 删除index的内容
			heap.Remove(s.indexes, i)
			break
		}
	}

	delete(s.items, nonce)

	return true, s.filter(func(tx *types.Transaction) bool { return tx.Nonce > nonce })
}

func (s *sortedTxsByNonce) filter(compare func(*types.Transaction) bool) []*types.Transaction {
	var removed []*types.Transaction
	if len(s.items) == 0 && s.indexes.Len() == 0 {
		return nil
	}

	for nonce, tx := range s.items {
		if compare(tx) {
			removed = append(removed, tx)
			// 从 items 中删除
			delete(s.items, nonce)
		}
	}
	// 更新堆(以态坊也是这么写的)  为什么不直接用heap.Remove()?
	if len(removed) > 0 {
		*s.indexes = make([]uint64, 0, len(s.items))
		for nonce := range s.items {
			*s.indexes = append(*s.indexes, nonce)
		}
		// 更新index堆
		heap.Init(s.indexes)
	}
	return removed
}
