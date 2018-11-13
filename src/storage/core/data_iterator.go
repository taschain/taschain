package core

import (
	"storage/trie"
)

type DataIterator struct {
	*trie.Iterator
	object *accountObject
}

func (di *DataIterator) GetValue() []byte {
	if v,ok := di.object.dirtyStorage[string(di.Key)];ok{
		return v
	} else {
		return di.Value
	}
}