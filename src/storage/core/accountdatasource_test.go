package core

import (
	"testing"
	"strconv"
	"storage/tasdb"
	"common"
	"storage/trie"

	"encoding/json"
)
func getString(trie *trie.Trie, k string) []byte {
	return trie.Get([]byte(k))
}

func updateString(trie *trie.Trie, k, v string) {
	trie.Update([]byte(k), []byte(v))
}

func deleteString(trie *trie.Trie, k string) {
	trie.Delete([]byte(k))
}
func TestExpandTrie(t *testing.T) {
	diskdb, _ := tasdb.NewMemDatabase()
	triedb := NewDatabase(diskdb)
	trie1, _ := trie.NewTrie(common.Hash{}, triedb.TrieDB())

	for i:=0;i<100;i++{
		updateString(trie1, strconv.Itoa(i), strconv.Itoa(i))
	}
	trie1.SetCacheLimit(10)
	for i:= 0;i<11;i++{
		trie1.Commit(nil)
	}

	root,_:=trie1.Commit(nil)
	triedb.TrieDB().Commit(root,false)

	for i:=0;i<100;i++{
		vl:= string(getString(trie1,strconv.Itoa(i)))
		if vl != strconv.Itoa(i){
			t.Errorf("wrong value: %v", vl)
		}
	}
	data := encode(trie1)


	newTrie:=decode(data)
	for i:=0;i<100;i++{
		vl:= string(getString(newTrie,strconv.Itoa(i)))
		if vl != strconv.Itoa(i){
			t.Errorf("wrong value: %v", vl)
		}
	}

}


func decode(data []byte)*trie.Trie{
	//buffer := bytes.NewBuffer(data)
	//decoder := gob.NewDecoder(buffer)
	var n trie.Trie
	err := json.Unmarshal(data,&n)
	if  err != nil{
		panic(err)
	}
	return &n
}

func encode(n Trie)[]byte{
	b , err := json. Marshal (n)

	if err != nil{
		panic(err)
	}
	return b
}

