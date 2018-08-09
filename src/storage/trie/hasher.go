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
	"bytes"
	"hash"
	"sync"

	"golang.org/x/crypto/sha3"
	"common"
)

type hasher struct {
	tmp        *bytes.Buffer
	sha        hash.Hash
	cachegen   uint16
	cachelimit uint16
	onleaf     LeafCallback
}

var hasherPool = sync.Pool{
	New: func() interface{} {
		return &hasher{tmp: new(bytes.Buffer), sha: sha3.New256()}
	},
}

func newHasher(cachegen, cachelimit uint16, onleaf LeafCallback) *hasher {
	h := hasherPool.Get().(*hasher)
	h.cachegen, h.cachelimit, h.onleaf = cachegen, cachelimit, onleaf
	return h
}

func returnHasherToPool(h *hasher) {
	hasherPool.Put(h)
}

func (h *hasher) hash(n node, db *Database, force bool) (node, node, error) {
	if hash, dirty := n.cache(); hash != nil {
		if db == nil {
			return hash, n, nil
		}
		if n.canUnload(h.cachegen, h.cachelimit) {
			cacheUnloadCounter.Inc(1)
			return hash, hash, nil
		}
		if !dirty {
			return hash, n, nil
		}
	}

	collapsed, cached, err := h.hashChildren(n, db)
	if err != nil {
		return hashNode{}, n, err
	}
	hashed, err := h.store(collapsed, db, force)
	if err != nil {
		return hashNode{}, n, err
	}

	cachedHash, _ := hashed.(hashNode)
	switch cn := cached.(type) {
	case *shortNode:
		cn.flags.hash = cachedHash
		if db != nil {
			cn.flags.dirty = false
		}
	case *fullNode:
		cn.flags.hash = cachedHash
		if db != nil {
			cn.flags.dirty = false
		}
	}
	return hashed, cached, nil
}


func (h *hasher) hashChildren(original node, db *Database) (node, node, error) {
	var err error

	switch n := original.(type) {
	case *shortNode:

		collapsed, cached := n.copy(), n.copy()

		cached.Key = common.CopyBytes(n.Key)

		if _, ok := n.Val.(valueNode); !ok {
			collapsed.Val, cached.Val, err = h.hash(n.Val, db, false)
			if err != nil {
				return original, original, err
			}
		}
		//if collapsed.Val == nil {
		//	collapsed.Val = valueNode(nil)
		//}
		return collapsed, cached, nil

	case *fullNode:

		collapsed, cached := n.copy(), n.copy()

		for i := 0; i < 16; i++ {
			if n.Children[i] != nil {
				collapsed.Children[i], cached.Children[i], err = h.hash(n.Children[i], db, false)
				if err != nil {
					return original, original, err
				}
			}
			//else {
			//	collapsed.Children[i] = valueNode(nil) // Ensure that nil children are encoded as empty strings.
			//}
		}
		cached.Children[16] = n.Children[16]
		//if collapsed.Children[16] == nil {
		//	collapsed.Children[16] = valueNode(nil)
		//}
		return collapsed, cached, nil

	default:

		return n, original, nil
	}
}

func (h *hasher) store(n node, db *Database, force bool) (node, error) {

	if _, isHash := n.(hashNode); n == nil || isHash {
		return n, nil
	}

	h.tmp.Reset()
	if err := n.encode(h.tmp); err != nil {
		panic("serialize error: " + err.Error())
	}
	if h.tmp.Len() < 32 && !force {
		return n, nil
	}

	hash, _ := n.cache()
	if hash == nil {
		h.sha.Reset()
		h.sha.Write(h.tmp.Bytes())
		hash = hashNode(h.sha.Sum(nil))
	}
	if db != nil {

		db.lock.Lock()

		hash := common.BytesToHash(hash)
		db.insert(hash, h.tmp.Bytes())

		switch n := n.(type) {
		case *shortNode:
			if child, ok := n.Val.(hashNode); ok {
				db.reference(common.BytesToHash(child), hash)
			}
		case *fullNode:
			for i := 0; i < 16; i++ {
				if child, ok := n.Children[i].(hashNode); ok {
					db.reference(common.BytesToHash(child), hash)
				}
			}
		}
		db.lock.Unlock()

		if h.onleaf != nil {
			switch n := n.(type) {
			case *shortNode:
				if child, ok := n.Val.(valueNode); ok {
					h.onleaf(child, hash)
				}
			case *fullNode:
				for i := 0; i < 16; i++ {
					if child, ok := n.Children[i].(valueNode); ok {
						h.onleaf(child, hash)
					}
				}
			}
		}
	}
	return hash, nil
}
