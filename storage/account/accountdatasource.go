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

package account

import (
	"sync"

	"fmt"
	"github.com/hashicorp/golang-lru"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/storage/tasdb"
	"github.com/taschain/taschain/storage/trie"
)

const (
	codeSizeCacheSize = 100000
)

type AccountDatabase interface {
	OpenTrie(root common.Hash) (Trie, error)

	//OpenTrieWithMap(root common.Hash, nodes map[string]*[]byte) (Trie, error)

	OpenStorageTrie(addrHash, root common.Hash) (Trie, error)

	CopyTrie(Trie) Trie

	PushTrie(root common.Hash, t Trie)

	ContractCode(addrHash, codeHash common.Hash) ([]byte, error)

	ContractCodeSize(addrHash, codeHash common.Hash) (int, error)

	TrieDB() *trie.NodeDatabase
}

type Trie interface {
	TryGet(key []byte) ([]byte, error)
	TryUpdate(key, value []byte) error
	TryDelete(key []byte) error
	Commit(onleaf trie.LeafCallback) (common.Hash, error)
	Hash() common.Hash
	NodeIterator(startKey []byte) trie.NodeIterator
	//Fstring() string
	//GetAllNodes(nodes map[string]*[]byte)
}

func NewDatabase(db tasdb.Database) AccountDatabase {
	csc, _ := lru.New(codeSizeCacheSize)
	return &storageDB{
		publicStorageDB: publicStorageDB{
			db:            trie.NewDatabase(db),
			codeSizeCache: csc,
		},
	}
}

func NewLightDatabase(db tasdb.Database) AccountDatabase {
	csc, _ := lru.New(codeSizeCacheSize)
	return &lightStorageDB{
		publicStorageDB: publicStorageDB{
			db:            trie.NewDatabase(db),
			codeSizeCache: csc,
		},
	}
}

type publicStorageDB struct {
	db *trie.NodeDatabase
	mu sync.Mutex
	//pastTries     []Trie
	codeSizeCache *lru.Cache
}

type storageDB struct {
	publicStorageDB
}

type lightStorageDB struct {
	publicStorageDB
}

func (db *publicStorageDB) TrieDB() *trie.NodeDatabase {
	return db.db
}

func (db *publicStorageDB) ContractCode(addrHash, codeHash common.Hash) ([]byte, error) {
	code, err := db.db.Node(codeHash)
	if err == nil {
		db.codeSizeCache.Add(codeHash, len(code))
	}
	return code, err
}

func (db *publicStorageDB) ContractCodeSize(addrHash, codeHash common.Hash) (int, error) {
	if cached, ok := db.codeSizeCache.Get(codeHash); ok {
		return cached.(int), nil
	}
	code, err := db.ContractCode(addrHash, codeHash)
	if err == nil {
		db.codeSizeCache.Add(codeHash, len(code))
	}
	return len(code), err
}

//func (db *storageDB) OpenTrieWithMap(root common.Hash, nodes map[string]*[]byte) (Trie, error) {
//	db.mu.Lock()
//	defer db.mu.Unlock()
//
//	tr, err := trie.NewTrieWithMap(root, db.db, nodes)
//	if err != nil {
//		return nil, err
//	}
//	return tr, nil
//}

func (db *storageDB) OpenTrie(root common.Hash) (Trie, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	//for i := len(db.pastTries) - 1; i >= 0; i-- {
	//	if db.pastTries[i].Hash() == root {
	//		return db.pastTries[i].Copy(),nil
	//	}
	//}

	tr, err := trie.NewTrie(root, db.db)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

func (db *storageDB) PushTrie(root common.Hash, t Trie) {
	//db.mu.Lock()
	//defer db.mu.Unlock()

	//if len(db.pastTries) >= maxPastTries {
	//	copy(db.pastTries, db.pastTries[1:])
	//	db.pastTries[len(db.pastTries)-1] = t
	//} else {
	//	db.pastTries = append(db.pastTries, t)
	//}
}

func (db *storageDB) OpenStorageTrie(addrHash, root common.Hash) (Trie, error) {
	return trie.NewTrie(root, db.db)
}

func (db *storageDB) CopyTrie(t Trie) Trie {
	switch t := t.(type) {
	case *trie.Trie:
		newTrie, _ := trie.NewTrie(t.Hash(), db.db)
		return newTrie
	default:
		panic(fmt.Errorf("unknown trie type %T", t))
	}
}

//func (db *storageDB) RestoreTrie(root common.Hash,hn []byte) []trie.node {
//	return nilsss
//}

func (db *lightStorageDB) OpenTrieWithMap(root common.Hash, nodes map[string]*[]byte) (Trie, error) {
	panic("not support")
}

func (db *lightStorageDB) OpenTrie(root common.Hash) (Trie, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	tr, err := trie.NewLightTrie(root, db.db)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

func (db *lightStorageDB) PushTrie(root common.Hash, t Trie) {
	//db.mu.Lock()
	//defer db.mu.Unlock()

	//if len(db.pastTries) >= maxPastTries {
	//	copy(db.pastTries, db.pastTries[1:])
	//	db.pastTries[len(db.pastTries)-1] = t
	//} else {
	//	db.pastTries = append(db.pastTries, t)
	//}
}

func (db *lightStorageDB) OpenStorageTrie(addrHash, root common.Hash) (Trie, error) {
	return trie.NewTrie(root, db.db)
}

func (db *lightStorageDB) CopyTrie(t Trie) Trie {
	switch t := t.(type) {
	case *trie.Trie:
		newTrie, _ := trie.NewTrie(t.Hash(), db.db)
		return newTrie
	default:
		panic(fmt.Errorf("unknown trie type %T", t))
	}
}