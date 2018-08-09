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
	"fmt"
	"io"
	"strings"

	//"storage/common"
	"encoding/gob"
	"bytes"

)

const encodeVersion byte = 1

const (
	magicFull byte = 0x1
	magicShort byte = 0x2
	magicHash byte = 0x3
	magicValue byte = 0x4
)

var indices = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f", "[17]"}

func init()  {
	gob.Register(valueNode{})
	gob.Register(hashNode{})
	gob.Register(nodeFlag{})
}

type node interface {
	fstring(string) string
	cache() (hashNode, bool)
	canUnload(cachegen, cachelimit uint16) bool
	magic() byte
	encode(io.Writer) error
}

type (
	fullNode struct {
		Children [17]node // Actual trie node data to serialize/decode (needs custom encoder)
		flags    nodeFlag
	}
	shortNode struct {
		Key   []byte
		Val   node
		flags nodeFlag
	}
	hashNode  []byte
	valueNode []byte
)

func (n *fullNode) copy() *fullNode   { copy := *n; return &copy }
func (n *shortNode) copy() *shortNode { copy := *n; return &copy }

type nodeFlag struct {
	hash  hashNode // cached hash of the node (may be nil)
	gen   uint16   // cache generation counter
	dirty bool     // whether the node has changes that must be written to the database
}

func (n *nodeFlag) canUnload(cachegen, cachelimit uint16) bool {
	return !n.dirty && cachegen-n.gen >= cachelimit
}

func (n *fullNode) canUnload(gen, limit uint16) bool  { return n.flags.canUnload(gen, limit) }
func (n *shortNode) canUnload(gen, limit uint16) bool { return n.flags.canUnload(gen, limit) }
func (n hashNode) canUnload(uint16, uint16) bool      { return false }
func (n valueNode) canUnload(uint16, uint16) bool     { return false }

func (n *fullNode) cache() (hashNode, bool)  { return n.flags.hash, n.flags.dirty }
func (n *shortNode) cache() (hashNode, bool) { return n.flags.hash, n.flags.dirty }
func (n hashNode) cache() (hashNode, bool)   { return nil, true }
func (n valueNode) cache() (hashNode, bool)  { return nil, true }

func (n *fullNode) magic() byte  { return magicFull }
func (n *shortNode) magic() byte { return magicShort }
func (n hashNode) magic() byte   { return magicHash }
func (n valueNode) magic() byte  { return magicValue }

func (n *fullNode) encode(w io.Writer) error  {
	return encode(w, n.magic(), n)
}

func (n *shortNode) encode(w io.Writer) error  {
	return encode(w, n.magic(), n)
}
func (n hashNode) encode(w io.Writer) error  { return encode(w, n.magic(), n) }
func (n valueNode) encode(w io.Writer) error  { return encode(w, n.magic(), n) }


// Pretty printing.
func (n *fullNode) String() string  { return n.fstring("") }
func (n *shortNode) String() string { return n.fstring("") }
func (n hashNode) String() string   { return n.fstring("") }
func (n valueNode) String() string  { return n.fstring("") }

func encode(w io.Writer,magic byte,node node) error{
	encoder := gob.NewEncoder(w)
	encoder.Encode(encodeVersion)
	encoder.Encode(magic)
	return encoder.Encode(node)
}

func (n *fullNode) fstring(ind string) string {
	resp := fmt.Sprintf("[\n%s  ", ind)
	for i, node := range n.Children {
		if node == nil {
			resp += fmt.Sprintf("%s: <nil> ", indices[i])
		} else {
			resp += fmt.Sprintf("%s: %v", indices[i], node.fstring(ind+"  "))
		}
	}
	return resp + fmt.Sprintf("\n%s] ", ind)
}
func (n *shortNode) fstring(ind string) string {
	return fmt.Sprintf("{short %x: %v} ", n.Key, n.Val.fstring(ind+"  "))
}
func (n hashNode) fstring(ind string) string {
	return fmt.Sprintf("hash <%x> ", []byte(n))
}
func (n valueNode) fstring(ind string) string {
	return fmt.Sprintf("value %x ", []byte(n))
}

func mustDecodeNode(hash, buf []byte, cachegen uint16) node {
	n, err := decodeNode2(hash, buf, cachegen)
	if err != nil {
		panic(fmt.Sprintf("node %x: %v", hash, err))
	}
	return n
}

func decodeNode2(hash, buf []byte, cachegen uint16) (node, error)  {
	if len(buf) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	buffer := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(buffer)
	var version,magic byte
	decoder.Decode(&version)
	decoder.Decode(&magic)
	switch magic {
		case magicFull:
		var n fullNode
			err := decoder.Decode(&n)
			n.flags.hash = hash
			n.flags.dirty = false
			return &n,err
		case magicShort:
			var n shortNode
			err := decoder.Decode(&n)
			//n.Key = compactToHex(n.Key)
			n.flags.hash = hash
			n.flags.dirty = false
			return &n,err
		case magicHash:
			var n hashNode
			err := decoder.Decode(&n)
			return &n,err
		case magicValue:
			var n valueNode
			err := decoder.Decode(&n)
			if len(n) == 0{
				return nil, err
			}
			return &n,err
	}
	return nil, fmt.Errorf("type mismatch")
}

type decodeError struct {
	what  error
	stack []string
}

func wrapError(err error, ctx string) error {
	if err == nil {
		return nil
	}
	if decErr, ok := err.(*decodeError); ok {
		decErr.stack = append(decErr.stack, ctx)
		return decErr
	}
	return &decodeError{err, []string{ctx}}
}

func (err *decodeError) Error() string {
	return fmt.Sprintf("%v (decode path: %s)", err.what, strings.Join(err.stack, "<-"))
}
