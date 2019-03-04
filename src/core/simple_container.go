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
	"middleware/types"
	"sort"
	"sync"
	"time"
)

type simpleContainer struct {
	lock   sync.RWMutex
	txs    types.PriorityTransactions
	txsMap map[common.Hash]*types.Transaction
	limit  int
	ticker *time.Ticker
}

func newSimpleContainer(l int) *simpleContainer {
	c := &simpleContainer{
		lock:   sync.RWMutex{},
		limit:  l,
		txsMap: map[common.Hash]*types.Transaction{},
		txs:    types.PriorityTransactions{},
		ticker: time.NewTicker(time.Millisecond * 500),
	}
	go c.loop()
	return c
}

func (c *simpleContainer) loop() {
	for {
		<-c.ticker.C
		c.sort()
	}
}

func (c *simpleContainer) sort() {
	c.lock.Lock()
	defer c.lock.Unlock()
	sort.Sort(c.txs)
}

func (c *simpleContainer) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.txs)
}

func (c *simpleContainer) Contains(key common.Hash) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.txsMap[key] != nil
}

func (c *simpleContainer) Get(key common.Hash) *types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.txsMap[key]
}

func (c *simpleContainer) AsSlice() []*types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.txs
}

func (c *simpleContainer) Push(tx *types.Transaction) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.add(tx)

}

func (c *simpleContainer) PushTxs(txs []*types.Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	for _, tx := range txs {
		c.add(tx)
	}

}

func (c *simpleContainer) add(tx *types.Transaction) {
	if c.txs.Len() < c.limit {
		c.txs = append(c.txs, tx)
		c.txsMap[tx.Hash] = tx
		return
	}

	for i, oldtx := range c.txs {
		if tx.GasPrice >= oldtx.GasPrice {
			delete(c.txsMap, oldtx.Hash)
			c.txs[i] = tx
			c.txsMap[tx.Hash] = tx
			break
		}
	}
}

func (c *simpleContainer) Remove(keys []common.Hash) {
	if nil == keys || 0 == len(keys) {
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	//Logger.Debugf("[Remove111:]tx pool container remove tx len:%d,contain tx map len %d,contain txs len %d",len(keys),len(c.txsMap),len(c.txs))
	for _, key := range keys {
		if c.txsMap[key] == nil {
			continue
		}
		delete(c.txsMap, key)
		//Logger.Debugf("txsMap delete Value contain tx map len %d",len(c.txsMap))

		for i, tx := range c.txs {
			if tx.Hash == key {
				c.txs = append(c.txs[:i], c.txs[i+1:]...)
				break
			}
		}
	}
}

func (c *simpleContainer) remove(key common.Hash) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.txsMap[key] == nil {
		return
	}

	delete(c.txsMap, key)
	for i, tx := range c.txs {
		if tx.Hash == key {
			c.txs = append(c.txs[:i], c.txs[i+1:]...)
			break
		}
	}
}
