package trie

import (
	"testing"
	"storage/tasdb"
	"common"
	"fmt"
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
	<- channel
}

func TestTrie_Get(t *testing.T) {
	diskdb, _ := tasdb.NewLDBDatabase("/Volumes/sda1/work", 0, 0)
	triedb := NewDatabase(diskdb)
	trie, _ := NewTrie(common.HexToHash("0x124e32fbe112a9fb8d73abb01c275f3f8ba809fb9347ca381b3a45dd28d5c5df"), triedb)
	fmt.Println(string(getString(trie,"xogglesw")))
	//fmt.Println(getString(trie,"xogee"))
	//fmt.Println(getString(trie,"xogef"))
	//updateString(trie, "xogef1", "cat12")
	//trie.Commit(nil)
	//root, _ := trie.Commit(nil)
	//triedb.Commit(root, false)
	//fmt.Println(root.Hex())
}