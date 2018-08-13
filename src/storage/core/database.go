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
	"sync"

	"storage/tasdb"
	"storage/trie"
	"github.com/hashicorp/golang-lru"
	"common"
	"fmt"
)

var MaxTrieCacheGen = uint16(120)

const (
	maxPastTries = 12

	codeSizeCacheSize = 100000
)

type Database interface {
	OpenTrie(root common.Hash) (Trie, error)

	OpenStorageTrie(addrHash, root common.Hash) (Trie, error)

	CopyTrie(Trie) Trie

	PushTrie(root common.Hash,t Trie)

	ContractCode(addrHash, codeHash common.Hash) ([]byte, error)

	ContractCodeSize(addrHash, codeHash common.Hash) (int, error)

	TrieDB() *trie.Database
}

type Trie interface {
	TryGet(key []byte) ([]byte, error)
	TryUpdate(key, value []byte) error
	TryDelete(key []byte) error
	Commit(onleaf trie.LeafCallback) (common.Hash, error)
	Hash() common.Hash
	Fstring() string
}

func NewDatabase(db tasdb.Database) Database {
	csc, _ := lru.New(codeSizeCacheSize)
	csc2, _ := lru.New(maxPastTries)
	return &storageDB{
		db:            trie.NewDatabase(db),
		codeSizeCache: csc,
		pastTriesCache: csc2,
	}
}

type storageDB struct {
	db            *trie.Database
	mu            sync.Mutex
	pastTriesCache    *lru.Cache
	codeSizeCache 	  *lru.Cache
}

func (db *storageDB) OpenTrie(root common.Hash) (Trie, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if tr,ok := db.pastTriesCache.Get(root);ok{
		return tr.(Trie),nil
	}

	tr, err := trie.NewTrie(root, db.db)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

func (db *storageDB) PushTrie(root common.Hash, t Trie) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.pastTriesCache.Add(root, t)
}

func (db *storageDB) OpenStorageTrie(addrHash, root common.Hash) (Trie, error) {
	return trie.NewTrie(root, db.db)
}

func (db *storageDB) CopyTrie(t Trie) Trie {
	switch t := t.(type) {
	case *trie.Trie:
		newTrie,_ := trie.NewTrie(t.Hash(), db.db)
		return newTrie
	default:
		panic(fmt.Errorf("unknown trie type %T", t))
	}
}

func (db *storageDB) ContractCode(addrHash, codeHash common.Hash) ([]byte, error) {
	code, err := db.db.Node(codeHash)
	if err == nil {
		db.codeSizeCache.Add(codeHash, len(code))
	}
	return code, err
}

func (db *storageDB) ContractCodeSize(addrHash, codeHash common.Hash) (int, error) {
	if cached, ok := db.codeSizeCache.Get(codeHash); ok {
		return cached.(int), nil
	}
	code, err := db.ContractCode(addrHash, codeHash)
	if err == nil {
		db.codeSizeCache.Add(codeHash, len(code))
	}
	return len(code), err
}

func (db *storageDB) TrieDB() *trie.Database {
	return db.db
}
