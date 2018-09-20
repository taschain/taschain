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

	"common"
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
	gob.Register(shortNode{})
	gob.Register(fullNode{})
}

type node interface {
	fstring(string) string
	cache() (hashNode, bool)
	canUnload(cachegen, cachelimit uint16) bool
	magic() byte
	encode(io.Writer) error
	print()string
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

func newNodeFlag(hash  hashNode,gen   uint16)nodeFlag{
	return nodeFlag{hash:hash,gen:gen}
}
func (n *fullNode) copy() *fullNode   { copy := *n; return &copy }
func (n *shortNode) copy() *shortNode { copy := *n; return &copy }

type nodeFlag struct {
	hash  hashNode // cached hash of the node (may be nil)
	gen   uint16   // cache generation counter
	dirty bool     // whether the node has changes that must be written to the database
}

func (n *fullNode) print()string  {
	str:= fmt.Sprintf("fullnode:flag=%v,value=[",n.flags)
	for index,d:=range n.Children{
		switch dn:=d.(type) {
		case hashNode:
			str1 := fmt.Sprintf("hashnode:index=%d,value=%v",index,(common.ToHex([]byte(dn))))
			str=fmt.Sprintf("%s,%s",str,str1)
		case valueNode:
			str1 := fmt.Sprintf(str,"valueNode:index=%d,value=%v",index,([]byte(dn)))
			str=fmt.Sprintf("%s,%s",str,str1)
		default:
			if d != nil{
				str1 := fmt.Sprintf("otherNode:index=%d",index)
				str=fmt.Sprintf("%s,%s",str,str1)
			}

		}
	}
	str=fmt.Sprintf("%s,%s",str,"]\n")
	return str

}
func (n *shortNode) print()string   {
	var str string= ""
	hashnode,isok :=n.Val.(hashNode)
	if isok{
		str = fmt.Sprintf("[hashnode:value=%v]",hashnode)
	}else{
		valuenode,isok :=n.Val.(valueNode)
		if isok{
			str = fmt.Sprintf("[valuenode:value=%v]",([]byte)(valuenode))
		}else{
			str = fmt.Sprintf("[othernode:value=%v]",n.Val)
		}
	}
	return fmt.Sprintf("shortNode:key=%v,value=%s,flag=%v\n",n.Key,str,n.flags)
	}
func (n hashNode) print()string   {return fmt.Sprintf("hashNode:value=%v\n",n) }
func (n valueNode) print()string  {return fmt.Sprintf("valueNode:value=%v\n",n) }

func (n *fullNode) cache() (hashNode, bool)  { return n.flags.hash, n.flags.dirty }
func (n *shortNode) cache() (hashNode, bool) { return n.flags.hash, n.flags.dirty }
func (n hashNode) cache() (hashNode, bool)   { return nil, true }
func (n valueNode) cache() (hashNode, bool)  { return nil, true }

func (n *nodeFlag) canUnload(cachegen, cachelimit uint16) bool {
	return !n.dirty && cachegen-n.gen >= cachelimit
}

func (n *fullNode) canUnload(gen, limit uint16) bool  { return n.flags.canUnload(gen, limit) }
func (n *shortNode) canUnload(gen, limit uint16) bool { return n.flags.canUnload(gen, limit) }
func (n hashNode) canUnload(uint16, uint16) bool      { return false }
func (n valueNode) canUnload(uint16, uint16) bool     { return false }

func (n *fullNode) magic() byte  { return magicFull }
func (n *shortNode) magic() byte { return magicShort }
func (n hashNode) magic() byte   { return magicHash }
func (n valueNode) magic() byte  { return magicValue }

func (n *fullNode) encode(w io.Writer) error  {
	return encode(w, n.magic(), n.Children)
}

func (n *shortNode) encode(w io.Writer) error  {
	return encode(w, n.magic(), n.Key,n.getValueType(),n.Val)
}

func (n *shortNode) getValueType() int8 {
	switch n.Val.(type) {
	case *fullNode:
		return 1
	case valueNode:
		return 2
	case hashNode:
		return 3
	case *shortNode:
		return 4
	}
	panic(fmt.Sprintf("unknow value type%v",n.Val))
}

func (n hashNode) encode(w io.Writer) error  { return encode(w, n.magic(), n) }
func (n valueNode) encode(w io.Writer) error  { return encode(w, n.magic(), n) }


// Pretty printing.
func (n *fullNode) String() string  { return n.fstring("") }
func (n *shortNode) String() string { return n.fstring("") }
func (n hashNode) String() string   { return n.fstring("") }
func (n valueNode) String() string  { return n.fstring("") }

func encode(w io.Writer,magic byte,vl ...interface{}) error{
	encoder := gob.NewEncoder(w)
	encoder.Encode(encodeVersion)
	encoder.Encode(magic)
	for _,data := range vl{
		er :=encoder.Encode(data)
		if er!=nil{
			return er
		}
	}
	return nil
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
	n, err := decodeNode(hash, buf, cachegen)
	if err != nil {
		panic(fmt.Sprintf("node %x: %v", hash, err))
	}
	return n
}

func mustDecodeNode2(hash, buf []byte) node {
	n, err := decodeNode2(hash, buf)
	if err != nil {
		panic(fmt.Sprintf("node %x: %v", hash, err))
	}
	return n
}

func decodeNode2(hash, buf []byte) (node, error)  {
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

		return &n,err
	case magicShort:
		var n shortNode
		err := decoder.Decode(&n)
		n.Key = compactToHex(n.Key)
		n.flags.hash = hash

		return &n,err
	}
	return nil, fmt.Errorf("type mismatch")
}

func decodeNode(hash, buf []byte, cachegen uint16) (node, error)  {
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
			var Children [17]node
			err := decoder.Decode(&Children)
			return &fullNode{Children:Children,flags:newNodeFlag(hash,cachegen)},err
		case magicShort:
			var key  []byte
			var valueType int8
			var vl node
			err:=decoder.Decode(&key)
			if err != nil{
				return nil,err
			}
			err=decoder.Decode(&valueType)
			if err != nil{
				return nil,err
			}
			if valueType == 1{
				var fn fullNode
				err=decoder.Decode(&fn)
				vl = &fn
			}else if valueType==2{
				var fn valueNode
				err=decoder.Decode(&fn)
				vl = fn
			}else if valueType==3{
				var fn hashNode
				err=decoder.Decode(&fn)
				vl = fn
			}else if valueType==4{
				var fn shortNode
				err=decoder.Decode(&fn)
				vl = &fn
			}
			if err != nil{
				return nil,err
			}
			key = compactToHex(key)
			return  &shortNode{Key:key,Val:vl,flags:newNodeFlag(hash,cachegen)},nil
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
