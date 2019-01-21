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
	"common"
	"fmt"
	"bytes"
)

type LightTrie struct {
	PublicTrie
}

func NewLightTrie(root common.Hash, db *NodeDatabase) (*LightTrie, error) {
	if db == nil {
		panic("trie.NewTrie called without a database")
	}
	trie := &LightTrie{
		PublicTrie: PublicTrie{
			db:           db,
			originalRoot: root,
		},
	}
	if (root != common.Hash{}) && root != emptyRoot {
		rootnode, err := trie.resolveHash(root[:], nil)
		if err != nil {
			return nil, err
		}
		trie.RootNode = rootnode
	}
	return trie, nil
}

func (t *LightTrie) Get(key []byte) []byte {
	res, err := t.TryGet(key)
	if err != nil {
		common.DefaultLogger.Error(fmt.Sprintf("Unhandled trie error: %v", err))
		panic("Light tri get key:" + common.BytesToAddress(key).GetHexString() + err.Error())
	}
	return res
}

func (t *LightTrie) Hash2(nodes map[string]*[]byte, isInit bool) {

}

func (t *LightTrie) Root() []byte { return t.Hash().Bytes() }

func (t *LightTrie) Hash() common.Hash {
	hash, cached, _ := t.hashRoot(nil, nil)
	t.RootNode = cached
	return common.BytesToHash(hash.(hashNode))
}

func (t *LightTrie) hashRoot(db *NodeDatabase, onleaf LeafCallback) (node, node, error) {
	if t.RootNode == nil {
		return hashNode(emptyRoot.Bytes()), nil, nil
	}
	h := newHasher(t.cachegen, t.cachelimit, onleaf)
	defer returnHasherToPool(h)
	return h.hash(t.RootNode, db, true)
}

func (t *LightTrie) Commit(onleaf LeafCallback) (root common.Hash, err error) {
	if t.db == nil {
		panic("commit called on trie with nil database")
	}
	hash, cached, err := t.hashRoot(t.db, onleaf)
	if err != nil {
		return common.Hash{}, err
	}
	t.RootNode = cached
	t.cachegen++
	return common.BytesToHash(hash.(hashNode)), nil
}

func (t *LightTrie) TryDelete(key []byte) error {
	k := keybytesToHex(key)
	_, n, err := t.delete(t.RootNode, nil, k)
	if err != nil {
		return err
	}
	t.RootNode = n
	return nil
}

func (t *LightTrie) newFlag() nodeFlag {
	return nodeFlag{dirty: true, gen: t.cachegen}
}

func (t *LightTrie) delete(n node, prefix, key []byte) (bool, node, error) {
	switch n := n.(type) {
	case *shortNode:
		matchlen := prefixLen(key, n.Key)
		if matchlen < len(n.Key) {
			return false, n, nil
		}
		if matchlen == len(key) {
			return true, nil, nil
		}

		dirty, child, err := t.delete(n.Val, append(prefix, key[:len(n.Key)]...), key[len(n.Key):])
		if !dirty || err != nil {
			return false, n, err
		}
		switch child := child.(type) {
		case *shortNode:
			return true, &shortNode{concat(n.Key, child.Key...), child.Val, t.newFlag()}, nil
		default:
			return true, &shortNode{n.Key, child, t.newFlag()}, nil
		}

	case *fullNode:
		dirty, nn, err := t.delete(n.Children[key[0]], append(prefix, key[0]), key[1:])
		if !dirty || err != nil {
			return false, n, err
		}
		n = n.copy()
		n.flags = t.newFlag()
		n.Children[key[0]] = nn

		pos := -1
		for i, cld := range n.Children {
			if cld != nil {
				if pos == -1 {
					pos = i
				} else {
					pos = -2
					break
				}
			}
		}
		if pos >= 0 {
			if pos != 16 {
				cnode, err := t.resolve(n.Children[pos], prefix)
				if err != nil {
					return false, nil, err
				}
				if cnode, ok := cnode.(*shortNode); ok {
					k := append([]byte{byte(pos)}, cnode.Key...)
					return true, &shortNode{k, cnode.Val, t.newFlag()}, nil
				}
			}
			return true, &shortNode{[]byte{byte(pos)}, n.Children[pos], t.newFlag()}, nil
		}
		return true, n, nil

	case valueNode:
		return true, nil, nil

	case nil:
		return false, nil, nil

	case hashNode:
		rn, err := t.resolveHash(n, prefix)
		if err != nil {
			return false, nil, err
		}
		dirty, nn, err := t.delete(rn, prefix, key)
		if !dirty || err != nil {
			return false, rn, err
		}
		return true, nn, nil

	default:
		panic(fmt.Sprintf("%T: invalid node: %v (%v)", n, n, key))
	}
}

func (t *LightTrie) resolve(n node, prefix []byte) (node, error) {
	if n, ok := n.(hashNode); ok {
		return t.resolveHash(n, prefix)
	}
	return n, nil
}

func (t *LightTrie) resolveHash(n hashNode, prefix []byte) (node, error) {
	hash := common.BytesToHash(n)
	enc, err := t.db.Node(hash)
	if err != nil {
		return nil, err
	}
	if enc == nil {
		return nil, nil
	}
	return mustDecodeNode(n, enc, t.cachegen), nil
}

func (t *LightTrie) Delete(key []byte) {
	if err := t.TryDelete(key); err != nil {
		common.DefaultLogger.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

func (t *LightTrie) TryGet(key []byte) ([]byte, error) {
	key = keybytesToHex(key)
	value, newroot, didResolve, err := t.tryGet(t.RootNode, key, 0)
	if err == nil && didResolve {
		t.RootNode = newroot
	}
	return value, err
}

func (t *LightTrie) tryGet(origNode node, key []byte, pos int) (value []byte, newnode node, didResolve bool, err error) {
	switch n := (origNode).(type) {
	case nil:
		return nil, nil, false, nil
	case valueNode:
		return n, n, false, nil
	case *shortNode:
		if len(key)-pos < len(n.Key) || !bytes.Equal(n.Key, key[pos:pos+len(n.Key)]) {
			return nil, n, false, nil
		}
		value, newnode, didResolve, err = t.tryGet(n.Val, key, pos+len(n.Key))
		if err == nil && didResolve {
			n = n.copy()
			n.Val = newnode
			n.flags.gen = t.cachegen
		}
		return value, n, didResolve, err
	case *fullNode:
		value, newnode, didResolve, err = t.tryGet(n.Children[key[pos]], key, pos+1)
		if err == nil && didResolve {
			n = n.copy()
			n.flags.gen = t.cachegen
			n.Children[key[pos]] = newnode
		}
		return value, n, didResolve, err
	case hashNode:
		child, err := t.resolveHash(n, key[:pos])
		if err != nil {
			return nil, n, true, err
		}
		value, newnode, _, err := t.tryGet(child, key, pos)
		return value, newnode, true, err
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", origNode, origNode))
	}
}

func (t *LightTrie) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil {
		common.DefaultLogger.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

func (t *LightTrie) TryUpdate(key, value []byte) error {
	k := keybytesToHex(key)
	if len(value) != 0 {
		_, n, err := t.insert(t.RootNode, nil, k, valueNode(value))
		if err != nil {
			return err
		}
		t.RootNode = n
	} else {
		_, n, err := t.delete(t.RootNode, nil, k)
		if err != nil {
			return err
		}
		t.RootNode = n
	}
	return nil
}

func (t *LightTrie) insert(n node, prefix, key []byte, value node) (bool, node, error) {
	if len(key) == 0 {
		if v, ok := n.(valueNode); ok {
			return !bytes.Equal(v, value.(valueNode)), value, nil
		}
		return true, value, nil
	}
	switch n := n.(type) {
	case *shortNode:
		matchlen := prefixLen(key, n.Key)

		if matchlen == len(n.Key) {
			dirty, nn, err := t.insert(n.Val, append(prefix, key[:matchlen]...), key[matchlen:], value)
			if !dirty || err != nil {
				return false, n, err
			}
			return true, &shortNode{n.Key, nn, t.newFlag()}, nil
		}

		branch := &fullNode{flags: t.newFlag()}
		var err error
		_, branch.Children[n.Key[matchlen]], err = t.insert(nil, append(prefix, n.Key[:matchlen+1]...), n.Key[matchlen+1:], n.Val)
		if err != nil {
			return false, nil, err
		}
		_, branch.Children[key[matchlen]], err = t.insert(nil, append(prefix, key[:matchlen+1]...), key[matchlen+1:], value)
		if err != nil {
			return false, nil, err
		}

		if matchlen == 0 {
			return true, branch, nil
		}

		return true, &shortNode{key[:matchlen], branch, t.newFlag()}, nil

	case *fullNode:
		dirty, nn, err := t.insert(n.Children[key[0]], append(prefix, key[0]), key[1:], value)
		if !dirty || err != nil {
			return false, n, err
		}
		n = n.copy()
		n.flags = t.newFlag()
		n.Children[key[0]] = nn
		return true, n, nil

	case nil:
		return true, &shortNode{key, value, t.newFlag()}, nil

	case hashNode:
		rn, err := t.resolveHash(n, prefix)
		if err != nil {
			return false, nil, err
		}
		dirty, nn, err := t.insert(rn, prefix, key, value)
		if !dirty || err != nil {
			return false, rn, err
		}
		return true, nn, nil

	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}
}

func (t *LightTrie) NodeIterator(start []byte) NodeIterator {
	//return newLightNodeIterator(t, start)
	return nil
}

func (t *LightTrie) GetAllNodes(nodes map[string]*[]byte) {
	panic("Not support!")
}
