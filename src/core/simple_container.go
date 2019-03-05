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
	"sort"
	"sync"
	"time"

	"common"
	"middleware/types"
)

type simpleContainer struct {
	limit  int
	txs    types.PriorityTransactions
	txsMap map[common.Hash]*types.Transaction

	sortTicker *time.Ticker
	lock       sync.RWMutex
}

func newSimpleContainer(l int) *simpleContainer {
	c := &simpleContainer{
		lock:       sync.RWMutex{},
		limit:      l,
		txsMap:     map[common.Hash]*types.Transaction{},
		txs:        types.PriorityTransactions{},
		sortTicker: time.NewTicker(time.Millisecond * 500),
	}
	go c.loop()
	return c
}

func (c *simpleContainer) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.txs)
}

func (c *simpleContainer) loop() {
	for {
		<-c.sortTicker.C
		c.sort()
	}
}

func (c *simpleContainer) sort() {
	c.lock.Lock()
	defer c.lock.Unlock()

	sort.Sort(c.txs)
}

func (c *simpleContainer) contains(key common.Hash) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.txsMap[key] != nil
}

func (c *simpleContainer) get(key common.Hash) *types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.txsMap[key]
}

func (c *simpleContainer) asSlice() []*types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.txs
}

func (c *simpleContainer) push(tx *types.Transaction) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.txs.Len() < c.limit {
		c.txs = append(c.txs, tx)
		c.txsMap[tx.Hash] = tx
		return
	}

	for i, oldTx := range c.txs {
		if tx.GasPrice >= oldTx.GasPrice {
			delete(c.txsMap, oldTx.Hash)
			c.txs[i] = tx
			c.txsMap[tx.Hash] = tx
			break
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
