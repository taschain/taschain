package trie

import (
	"common"
)

type PublicTrie struct {
	db           *Database
	RootNode         node
	originalRoot common.Hash
	cachegen, cachelimit uint16
}


func (t *PublicTrie) TryGet(key []byte) ([]byte, error) {
	return nil,nil
}

func (t *PublicTrie) TryUpdate(key, value []byte) error {
	return nil
}

func (t *PublicTrie) TryDelete(key []byte) error {
	return nil
}

func (t *PublicTrie) Commit(onleaf LeafCallback) (root common.Hash, err error) {
	panic("not expect enter here")
}

func (t *PublicTrie) Hash() common.Hash {
	panic("not expect enter here")
}

func (t *PublicTrie) NodeIterator(start []byte) NodeIterator {
	panic("not expect enter here")
}

func (t *PublicTrie) Fstring() string{
	if t.RootNode == nil{
		return ""
	}
	return t.RootNode.fstring("")
}
