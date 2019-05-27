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
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/storage/tasdb"
	"testing"
)

func TestDatabase_Insert(t *testing.T) {
	diskdb, _ := tasdb.NewLDBDatabase("/Volumes/sda1/work", 0, 0)
	triedb := NewDatabase(diskdb)
	trie, _ := NewTrie(common.Hash{}, triedb)
	updateString(trie, "xogglesw", "cat")
	updateString(trie, "xogee", "cat11")
	updateString(trie, "xogef", "cat12")
	trie.Commit(nil)
	root, _ := trie.Commit(nil)
	triedb.Commit(root, false)
	fmt.Println(root.Hex())
	channel := make(chan struct{})
	<-channel
}

func TestTrie_Get(t *testing.T) {
	diskdb, _ := tasdb.NewLDBDatabase("/Volumes/sda1/work", 0, 0)
	triedb := NewDatabase(diskdb)
	trie, _ := NewTrie(common.HexToHash("0x124e32fbe112a9fb8d73abb01c275f3f8ba809fb9347ca381b3a45dd28d5c5df"), triedb)
	fmt.Println(string(getString(trie, "xogglesw")))
	//fmt.Println(getString(trie,"xogee"))
	//fmt.Println(getString(trie,"xogef"))
	//updateString(trie, "xogef1", "cat12")
	//trie.Commit(nil)
	//root, _ := trie.Commit(nil)
	//triedb.Commit(root, false)
	//fmt.Println(root.Hex())
}
