package core

import (
	"storage/trie"
	"strings"
)

type DataIterator struct {
	*trie.Iterator
	object *accountObject
	prefix string
}

func (di *DataIterator) Next() bool{
	if len(di.prefix) == 0{
		return di.Iterator.Next()
	}
	for di.Iterator.Next(){
		if strings.HasPrefix(string(di.Key), di.prefix){
			return true
		}
	}
	return false
}

func (di *DataIterator) GetValue() []byte {
	if v,ok := di.object.dirtyStorage[string(di.Key)];ok{
		return v
	} else {
		return di.Value
	}
}