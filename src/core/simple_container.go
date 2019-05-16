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
	"fmt"
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
	//sortedTxs sortedTxsByPrice
	pending map[common.Address]*sortedTxsByNonce
	queue   []*types.Transaction

	AllTxs map[common.Hash]*types.Transaction

	lock sync.RWMutex
}

//type txList struct {
//	sortedTxsByPrice *skip.SkipList
//}

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
	cache   []uint64
}

func newSimpleContainer(lp, lq int) *simpleContainer {
	c := &simpleContainer{
		lock:         sync.RWMutex{},
		limit:        lp + lq,
		pendingLimit: lp,
		queueLimit:   lq,
		AllTxs:       make(map[common.Hash]*types.Transaction),
		pending:      make(map[common.Address]*sortedTxsByNonce),
		queue:        make([]*types.Transaction, 0),

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

	//return c.AllTxs[key] != nil
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

	count := 0
	for _, v := range c.pending {
		nonce := heap.Pop(v.indexes).(uint64)
		v.cache = append(v.cache, nonce)
		tx := v.items[nonce]
		if !f(tx) || len(c.pending) == count {
			return
		}
		count++
	}
	//for i := 0; i < len(c.pending); i++  {
	//	if f()
	//}
	//
	//
	//
	//iter := c.sortedTxsByPrice.IterAtPosition(0)
	//for iter.Next() {
	//	if !f(iter.Value().(*types.Transaction)) {
	//		break
	//	}
	//}
}

func (c *simpleContainer) recoverFromCache() {
	// 把缓存内容再放入indexes中
	for _, v := range c.pending {
		for i := 0; i < len(v.cache); i++ {
			heap.Push(v.indexes, v.cache[i])
		}
		v.cache = nil
	}
}

func (c *simpleContainer) push(tx *types.Transaction) {
	c.lock.Lock()
	defer c.lock.Unlock()
	fmt.Printf("Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", tx.Hash, tx.GasPrice, tx.Nonce, tx.Source)

	if _, exist := c.AllTxs[tx.Hash]; exist {
		return
	}

	// TODO 长度定义 应为 pending+queue 长度 ??
	var pendingTxsCount int
	for _, v := range c.pending {
		pendingTxsCount += v.indexes.Len()
	}

	// 清除空间
	if pendingTxsCount >= c.pendingLimit {

		e := c.sortedTxsByPrice.ByPosition(0)
		headTx := e.(*types.Transaction)
		headTxPrice := headTx.GasPrice
		//nonce := headTx.Nonce
		//headTxHash := headTx.Hash
		//headTxSource := *headTx.Source

		// TODO 本地堆是否存在
		pending, isExistInPending := c.pending[*tx.Source]
		// 还需判断queue是否存在以及是否满了
		//var isExistInQueue bool
		//for _, v := range c.queue{
		//	if bytes.Equal((*tx.Source)[:],v.Source[:]){
		//		isExistInQueue = true
		//	}
		//}
		//
		if !isExistInPending && tx.GasPrice <= headTxPrice {
			return
		}

		// pending 中是否存在
		if isExistInPending {
			//fmt.Println("Instart")

			if tx.Nonce < pending.getMinNonce() {
				if ok, txs := pending.removeEnd(pending); ok && len(txs) >= 0 {
					//fmt.Println("pending in removeEnd")
					//TODO 放到queue中
					//for _, willInQueue := range txs{
					c.queue = append(c.queue, txs...)
					//	fmt.Printf("Will in queue 2>> Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", willInQueue.Hash, willInQueue.GasPrice,willInQueue.Nonce,willInQueue.Source)
					//}
				}
				if len(pending.items) == 0 {
					//fmt.Println("del")
					delete(c.pending, *tx.Source)
				}
				//c.sortedTxsByPrice.Delete(oldTx)
				//delete(c.AllTxs, oldTx.Hash)
			}

			if tx.Nonce > pending.getMaxNonce() {
				c.queue = append(c.queue, tx)
				c.AllTxs[tx.Hash] = tx
				c.sortedTxsByPrice.Insert(tx)
				return
			}

			if tx.Nonce > pending.getMinNonce() && tx.Nonce < pending.getMaxNonce() {
				if ok, txs := pending.removeEnd(pending); ok && len(txs) >= 0 {
					fmt.Println("pending in removeEnd")
					fmt.Println(len(txs))
					//TODO 放到queue中
					//for _, willInQueue := range txs{
					c.queue = append(c.queue, txs...)
					//	fmt.Printf("Will in queue 2>> Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", willInQueue.Hash, willInQueue.GasPrice,willInQueue.Nonce,willInQueue.Source)
					//}
				}
				if len(pending.items) == 0 {
					//fmt.Println("del")
					delete(c.pending, *tx.Source)
				}
			}

			if find, oldTx := pending.FindSameNonce(tx.Nonce); find {
				if oldTx.GasPrice >= tx.GasPrice {
					return
				}
				if ok, txs := pending.remove(oldTx); ok && len(txs) >= 0 {
					//TODO 放到queue中
					//for _, willInQueue := range txs{
					c.queue = append(c.queue, txs...)
					//	fmt.Printf("Will in queue 1>> Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", willInQueue.Hash, willInQueue.GasPrice,willInQueue.Nonce,willInQueue.Source)
					//}
				}

				if len(pending.items) == 0 {
					delete(c.pending, *tx.Source)
				}
				c.sortedTxsByPrice.Delete(oldTx)
				delete(c.AllTxs, oldTx.Hash)
			}

			//for nonce, oldTx := range pending.items{
			//	//fmt.Println(nonce)
			//	// 从pending中删除
			//	if tx.Nonce == nonce {
			//		//fmt.Println("Instart1")
			//		if oldTx.GasPrice >= tx.GasPrice {
			//			return
			//		}
			//		if ok, txs := pending.remove(oldTx); ok && len(txs) >=0{
			//			//TODO 放到queue中
			//			for _, willInQueue := range txs{
			//				c.queue=append(c.queue, txs...)
			//				fmt.Printf("Will in queue 1>> Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", willInQueue.Hash, willInQueue.GasPrice,willInQueue.Nonce,willInQueue.Source)
			//			}
			//		}
			//
			//		if len(pending.items) == 0 {
			//			delete(c.pending, *tx.Source)
			//		}
			//		c.sortedTxsByPrice.Delete(oldTx)
			//		delete(c.AllTxs, oldTx.Hash)
			//		break
			//	}
			//
			//	// 移除小于交易nonce的该地址pending最大nonce交易到queue
			//	if tx.Nonce < nonce {
			//		//fmt.Println("Instart2")
			//		if ok, txs := pending.removeEnd(pending); ok && len(txs) >= 0{
			//			//fmt.Println("pending in removeEnd")
			//			//TODO 放到queue中
			//			for _, willInQueue := range txs{
			//				c.queue=append(c.queue, txs...)
			//				fmt.Printf("Will in queue 2>> Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", willInQueue.Hash, willInQueue.GasPrice,willInQueue.Nonce,willInQueue.Source)
			//			}
			//		}
			//		if len(pending.items) == 0 {
			//			//fmt.Println("del")
			//			delete(c.pending, *tx.Source)
			//		}
			//		//c.sortedTxsByPrice.Delete(oldTx)
			//		//delete(c.AllTxs, oldTx.Hash)
			//		break
			//	}
			//
			//	if tx.Nonce > pending.getMaxNonce(){
			//		//fmt.Println("directly in queue")
			//		c.queue=append(c.queue,tx)
			//		c.AllTxs[tx.Hash]=tx
			//		c.sortedTxsByPrice.Insert(tx)
			//		return
			//	}
			//
			//}
			//fmt.Println("InEnd")
		}

		// TODO 插入queue中,若有空间……
		//if !isExistInPending {
		// TODO 去queue中寻找
		//fmt.Println("notInStart")
		//if ok, txs := pending.remove(e.(*types.Transaction)); ok && len(txs) >= 0{
		//	// 放到queue中
		//	for _, willInQueue := range txs{
		//		fmt.Printf("Will in queue 3>> Hash:%x,\tGas:%d,\tNonce:%d,\tSource:%s\n", willInQueue.Hash, willInQueue.GasPrice,willInQueue.Nonce,willInQueue.Source)
		//	}
		//}
		//
		//if len(pending.items) == 0 {
		//	delete(c.pending, headTxSource)
		//}
		//c.sortedTxsByPrice.Delete(e)
		//delete(c.AllTxs, headTxHash)
		//fmt.Println("notInEnd")
		//fmt.Println("queue>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")

		//}
		if !isExistInPending {
			c.queue = append(c.queue, tx)
			c.AllTxs[tx.Hash] = tx
			c.sortedTxsByPrice.Insert(tx)
			fmt.Println("not in pending")
			return
		}

	}

	if pending := c.pending[*tx.Source]; pending != nil {

		old, exist := pending.items[tx.Nonce]
		if !exist {
			//fmt.Println("CommonInsertNotExist")
			c.insertPending(tx, false)
			return
		}

		if exist && old.GasPrice < tx.GasPrice {
			//fmt.Println("CommonInsertExistReplace")
			c.replace(old, tx)
			return

		}

	}
	c.insertPending(tx, true)
	//fmt.Println("AllLastInsert")
}

func (c *simpleContainer) remove(key common.Hash) {
	if !c.contains(key) {
		return
	}
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
			c.pending[*tx.Source].cache = nil
		}
	}

	//for i, tx := range c.txs {
	//	if tx.Hash == key {
	//		heap.Remove(&c.txs, i)
	//		break
	//	}
	//}
}

func (c *simpleContainer) replace(old, new *types.Transaction) {
	if _, exist := c.AllTxs[old.Hash]; !exist {
		return
	}

	delete(c.AllTxs, old.Hash)
	c.sortedTxsByPrice.Delete(old)

	c.pending[*old.Source].items[old.Nonce] = new
	c.AllTxs[new.Hash] = new
	c.sortedTxsByPrice.Insert(new)

}

func (c *simpleContainer) insertPending(new *types.Transaction, isFirstTime bool) {
	// TODO 决定是放在queue中还是pending中  现在是不验证nonce，直接放在pending中
	addr := *new.Source
	nonce := new.Nonce

	if isFirstTime {
		c.pending[addr] = newSortedTxsByNonce()
		heap.Init(c.pending[addr].indexes)
	}

	heap.Push(c.pending[addr].indexes, nonce)
	c.pending[addr].items[nonce] = new
	c.sortedTxsByPrice.Insert(new)
	c.AllTxs[new.Hash] = new
}

func newSortedTxsByNonce() *sortedTxsByNonce {
	s := &sortedTxsByNonce{
		items:   make(map[uint64]*types.Transaction),
		indexes: new(nonceHeap),
		cache:   make([]uint64, 0),
	}
	return s
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

func (s *sortedTxsByNonce) removeEnd(pending *sortedTxsByNonce) (bool, []*types.Transaction) {
	removedNonce := heap.Remove(pending.indexes, pending.indexes.Len()-1).(uint64)
	fmt.Println(removedNonce)

	// 放到queue中
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

func (s *sortedTxsByNonce) filter(filter func(*types.Transaction) bool) []*types.Transaction {
	var removed []*types.Transaction

	if len(s.items) == 0 && s.indexes.Len() == 0 {
		return nil
	}
	// Collect all the transactions to filter out
	//position := 0
	for nonce, tx := range s.items {
		if filter(tx) {
			removed = append(removed, tx)
			// 从 items 中删除
			delete(s.items, nonce)
		}
	}
	// If transactions were removed, the heap are ruined
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
