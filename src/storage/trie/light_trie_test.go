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
	"testing"
	"common"

	"storage/tasdb"
)

func TestLightInsertAndDel(t *testing.T) {
	db, _ := tasdb.NewMemDatabase()
	triedb := NewDatabase(db)
	trie,_:=NewLightTrie(common.Hash{}, triedb)
	updateLightString(trie,"key1","2")
	vl:=string(getLightString(trie,"key1"))
	if vl !="2"{
		t.Errorf("Wrong error: %v", vl)
	}
	updateLightString(trie,"key1","1")
	vl =string(getLightString(trie,"key1"))
	if vl !="1"{
		t.Errorf("Wrong error: %v", vl)
	}
	updateLightString(trie,"key12","12")
	vl =string(getLightString(trie,"key12"))
	if vl !="12"{
		t.Errorf("Wrong error: %v", vl)
	}
	deleteLightString(trie,"key12")
	vlb:=getLightString(trie,"key12")
	if vlb != nil{
		t.Errorf("Wrong error: %v", vl)
	}
	updateLightString(trie,"key2","2")
	updateLightString(trie,"key3","3")
	updateLightString(trie,"key4","4")
	root,_:=trie.Commit(nil)
	triedb.Commit(root,false)
	trie,_=NewLightTrie(root, triedb)
}


func TestLRU(t *testing.T) {
	db, _ := tasdb.NewLRUMemDatabase(10)
	triedb := NewDatabase(db)
	trie,_:=NewLightTrie(common.Hash{}, triedb)
	updateLightString(trie,"key1","1")
	vl:=string(getLightString(trie,"key1"))
	if vl !="1"{
		t.Errorf("Wrong error: %v", vl)
	}

	root,_:=trie.Commit(nil)
	triedb.Commit(root,false)
}



func getLightString(trie *LightTrie, k string) []byte {
	return trie.Get([]byte(k))
}

func updateLightString(trie *LightTrie, k, v string) {
	trie.Update([]byte(k), []byte(v))
}

func deleteLightString(trie *LightTrie, k string) {
	trie.Delete([]byte(k))
}
