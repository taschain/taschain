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
	"github.com/taschain/taschain/common"
)

type PublicTrie struct {
	db                   *NodeDatabase
	RootNode             node
	originalRoot         common.Hash
	cachegen, cachelimit uint16
}

func (t *PublicTrie) TryGet(key []byte) ([]byte, error) {
	return nil, nil
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

func (t *PublicTrie) Fstring() string {
	if t.RootNode == nil {
		return ""
	}
	return t.RootNode.fstring("")
}
