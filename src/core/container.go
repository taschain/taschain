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
	if len > 1000{
		result = make([]*types.Transaction, 1000)
		copy(result, c.txs[len - 1000:])
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
	if c.txs.Len() < c.limit {
		c.txs = append(c.txs, tx)
		c.txsMap[tx.Hash] = tx
		return
	}

	if !c.inited {
		heap.Init(&c.txs)
		c.inited = true

	}

	evicted := heap.Pop(&c.txs).(*types.Transaction)
	delete(c.txsMap, evicted.Hash)
	heap.Push(&c.txs, tx)
	c.txsMap[tx.Hash] = tx
}

func (c *container) Remove(keys []common.Hash) {
	if nil == keys || 0 == len(keys) {
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	//Logger.Debugf("[Remove111:]tx pool container remove tx len:%d,contain tx map len %d,contain txs len %d",len(keys),len(c.txsMap),len(c.txs))
	if len(keys) < 50 {
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
	} else {
		queryMap := make(map[common.Hash]struct{})
		var num int
		for _, key := range keys {
			transaction := c.txsMap[key]
			if transaction != nil{
				transaction.Type = types.TransactionTypeToBeRemoved
				num++
				queryMap[key] = struct{}{}
			} else {
				for i := len(c.txs) - 1;i >= 0;i--{
					if c.txs[i].Hash == key {
						c.txs[i].Type = types.TransactionTypeToBeRemoved
						num++
						break
					}
				}
			}
			delete(c.txsMap, key)
		}
		heap.Init(&c.txs)
		toRemove := c.txs[:num]
		c.txs = c.txs[num:]

		for _,item := range toRemove{
			if _,ok := queryMap[item.Hash];!ok{
				for i,tx := range c.txs{
					if _,ok := queryMap[tx.Hash];ok{
						c.txs[i] = item
						break
					}
				}
			}
		}
		heap.Init(&c.txs)
	}
	//Logger.Debugf("[Remove111:]After remove,contain tx map len %d,contain txs len %d",len(c.txsMap),len(c.txs))
}