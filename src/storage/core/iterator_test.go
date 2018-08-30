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
	"bytes"
	"testing"

	"common"
	"storage/tasdb"
	"fmt"
)

func TestNodeIterator(t *testing.T)  {
	diskdb, _ := tasdb.NewMemDatabase()
	db := NewDatabase(diskdb)
	state, _ := NewAccountDB(common.Hash{}, db)
	state.CreateAccount(common.StringToAddress("1"))
	state.SetCode(common.StringToAddress("1"),[]byte("hello world"))

	state.SetData(common.StringToAddress("1"),"a",[]byte("b"))
	state.SetData(common.StringToAddress("1"),"c",[]byte("d"))
	state.Commit(true)
	stateObject := state.getAccountObject(common.StringToAddress("1"))
	for it2 := stateObject.DataIterator(nil); it2.Next();{
		fmt.Printf("%s %s\n",string(it2.Key), it2.Value)
	}

	for it := NewNodeIterator(state); it.Next(); {
		if it.Hash != (common.Hash{}) {
			fmt.Println(string(it.code))
		}
	}
}

// Tests that the node iterator indeed walks over the entire database contents.
func TestNodeIteratorCoverage(t *testing.T) {
	// Create some arbitrary test state to iterate
	db, root, _ := makeTestState()

	state, err := NewAccountDB(root, db)
	if err != nil {
		t.Fatalf("failed to create state trie at %x: %v", root, err)
	}
	// Gather all the node hashes found by the iterator
	hashes := make(map[common.Hash]struct{})
	for it := NewNodeIterator(state); it.Next(); {
		if it.Hash != (common.Hash{}) {
			hashes[it.Hash] = struct{}{}
		}
	}
	// Cross check the iterated hashes and the database/nodepool content
	for hash := range hashes {
		if _, err := db.TrieDB().Node(hash); err != nil {
			t.Errorf("failed to retrieve reported node %x", hash)
		}
	}
	for _, hash := range db.TrieDB().Nodes() {
		if _, ok := hashes[hash]; !ok {
			t.Errorf("state entry not reported %x", hash)
		}
	}
	for _, key := range db.TrieDB().DiskDB().(*tasdb.MemDatabase).Keys() {
		if bytes.HasPrefix(key, []byte("secure-key-")) {
			continue
		}
		if _, ok := hashes[common.BytesToHash(key)]; !ok {
			t.Errorf("state entry not reported %x", key)
		}
	}
}
