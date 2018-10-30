package trie

import (
	"testing"
	"common"
	"core/datasource"
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
	db, _ := datasource.NewLRUMemDatabase(10)
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
