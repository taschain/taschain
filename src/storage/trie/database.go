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

package trie

import (
	"sync"
	"time"

	"storage/tasdb"
	"common"
)

type DatabaseReader interface {
	Get(key []byte) (value []byte, err error)

	Has(key []byte) (bool, error)
}

type NodeDatabase struct {
	diskdb tasdb.Database

	nodes map[common.Hash]*cachedNode

	gctime  time.Duration
	gcnodes uint64
	gcsize  common.StorageSize

	nodesSize common.StorageSize

	lock sync.RWMutex
}

type cachedNode struct {
	blob     []byte
	parents  int
	children map[common.Hash]int
}

func NewDatabase(diskdb tasdb.Database) *NodeDatabase {
	return &NodeDatabase{
		diskdb: diskdb,
		nodes: map[common.Hash]*cachedNode{
			{}: {children: make(map[common.Hash]int)},
		},
	}
}

func (db *NodeDatabase) Node(hash common.Hash) ([]byte, error) {
	db.lock.RLock()
	node := db.nodes[hash]
	db.lock.RUnlock()
	if node != nil {
		return node.blob, nil
	}
	return db.diskdb.Get(hash[:])
}

func (db *NodeDatabase) Nodes() []common.Hash {
	db.lock.RLock()
	defer db.lock.RUnlock()

	var hashes = make([]common.Hash, 0, len(db.nodes))
	for hash := range db.nodes {
		if hash != (common.Hash{}) { // Special case for "root" references/nodes
			hashes = append(hashes, hash)
		}
	}
	return hashes
}

func (db *NodeDatabase) Reference(child common.Hash, parent common.Hash) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	db.reference(child, parent)
}

func (db *NodeDatabase) reference(child common.Hash, parent common.Hash) {

	node, ok := db.nodes[child]
	if !ok {
		return
	}

	if _, ok = db.nodes[parent].children[child]; ok && parent != (common.Hash{}) {
		return
	}
	node.parents++
	db.nodes[parent].children[child]++
}

func (db *NodeDatabase) Dereference(child common.Hash, parent common.Hash) {
	db.lock.Lock()
	defer db.lock.Unlock()

	nodes, storage, start := len(db.nodes), db.nodesSize, time.Now()
	db.dereference(child, parent)

	db.gcnodes += uint64(nodes - len(db.nodes))
	db.gcsize += storage - db.nodesSize
	db.gctime += time.Since(start)

	common.DefaultLogger.Debug("Dereferenced trie from memory database", "nodes", nodes-len(db.nodes), "size", storage-db.nodesSize, "time", time.Since(start),
		"gcnodes", db.gcnodes, "gcsize", db.gcsize, "gctime", db.gctime, "livenodes", len(db.nodes), "livesize", db.nodesSize)
}

func (db *NodeDatabase) dereference(child common.Hash, parent common.Hash) {

	node := db.nodes[parent]

	node.children[child]--
	if node.children[child] == 0 {
		delete(node.children, child)
	}

	node, ok := db.nodes[child]
	if !ok {
		return
	}

	node.parents--
	if node.parents == 0 {
		for hash := range node.children {
			db.dereference(hash, child)
		}
		delete(db.nodes, child)
		db.nodesSize -= common.StorageSize(common.HashLength + len(node.blob))
	}
}

func (db *NodeDatabase) DiskDB() DatabaseReader {
	return db.diskdb
}

func (db *NodeDatabase) Insert(hash common.Hash, blob []byte) {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.insert(hash, blob)
}

func (db *NodeDatabase) insert(hash common.Hash, blob []byte) {
	if _, ok := db.nodes[hash]; ok {
		return
	}
	db.nodes[hash] = &cachedNode{
		blob:     common.CopyBytes(blob),
		children: make(map[common.Hash]int),
	}
	db.nodesSize += common.StorageSize(common.HashLength + len(blob))
}

func (db *NodeDatabase) Commit(node common.Hash, report bool) error {
	db.lock.RLock()

	start := time.Now()
	batch := db.diskdb.NewBatch()

	nodes, storage := len(db.nodes), db.nodesSize
	if err := db.commit(node, batch); err != nil {
		common.DefaultLogger.Error("Failed to commit trie from trie database", "err", err)
		db.lock.RUnlock()
		return err
	}

	if err := batch.Write(); err != nil {
		common.DefaultLogger.Error("Failed to write trie to disk", "err", err)
		db.lock.RUnlock()
		return err
	}
	db.lock.RUnlock()

	db.lock.Lock()
	defer db.lock.Unlock()

	db.uncache(node)

	logger := common.DefaultLogger.Info
	if !report {
		logger = common.DefaultLogger.Debug
	}
	logger("Persisted trie from memory database", "nodes", nodes-len(db.nodes), "size", storage-db.nodesSize, "time", time.Since(start),
		"gcnodes", db.gcnodes, "gcsize", db.gcsize, "gctime", db.gctime, "livenodes", len(db.nodes), "livesize", db.nodesSize)

	db.gcnodes, db.gcsize, db.gctime = 0, 0, 0

	return nil
}

func (db *NodeDatabase) commit(hash common.Hash, batch tasdb.Batch) error {

	node, ok := db.nodes[hash]
	if !ok {
		return nil
	}
	for child := range node.children {
		if err := db.commit(child, batch); err != nil {
			return err
		}
	}
	if err := batch.Put(hash[:], node.blob); err != nil {
		return err
	}

	if batch.ValueSize() >= tasdb.IdealBatchSize {
		if err := batch.Write(); err != nil {
			return err
		}
		batch.Reset()
	}
	return nil
}

func (db *NodeDatabase) uncache(hash common.Hash) {

	node, ok := db.nodes[hash]
	if !ok {
		return
	}

	for child := range node.children {
		db.uncache(child)
	}
	delete(db.nodes, hash)
	db.nodesSize -= common.StorageSize(common.HashLength + len(node.blob))
}

func (db *NodeDatabase) Size() common.StorageSize {
	db.lock.RLock()
	defer db.lock.RUnlock()

	return db.nodesSize
}
