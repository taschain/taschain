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

package common

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"runtime/debug"
	"testing"
)

// BigPow returns a ** b as a big integer.
func BigPow(a, b int64) *big.Int {
	r := big.NewInt(a)
	return r.Exp(r, big.NewInt(b), nil)
}

// Various big integer limit values.
var (
	tt255     = BigPow(2, 255)
	tt256     = BigPow(2, 256)
	tt256m1   = new(big.Int).Sub(tt256, big.NewInt(1))
	tt63      = BigPow(2, 63)
	MaxBig256 = new(big.Int).Set(tt256m1)
	MaxBig63  = new(big.Int).Sub(tt63, big.NewInt(1))
)

/*
**  Creator: pxf
**  Date: 2018/9/30 下午3:11
**  Description:
 */

func TestHash_Hex(t *testing.T) {
	var h Hash
	h = HexToHash("0x1234")
	t.Log(h.Hex())

	s := "0xf3be4592802e6bfa85bf449c41eea1fc7a695220590c677c46d84339a13eec1a"
	h = HexToHash(s)
	t.Log(h.Hex())
}

func TestAddress_MarshalJSON(t *testing.T) {
	addr := HexToAddress("0x123")

	bs, _ := json.Marshal(&addr)
	log.Printf(string(bs))
}

func a(bs []byte) {
	fmt.Println(bs[0])
}

func TestDebug(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("捕获到的错误：%s\n", r)
			bs := debug.Stack()
			fmt.Println(string(bs))
		}
	}()
	var bs []byte
	a(bs)
}

func getmap() map[string]int {
	m := make(map[string]int)
	m["111"] = 1
	m2 := m
	log.Printf("%p %p\n", m, m2)
	return m2
}

func TestMap(t *testing.T) {

	m := getmap()

	t.Logf("%p\n", m)

}

func TestHashEqual(t *testing.T) {
	h1 := HexToHash("0x123")
	h2 := HexToHash("0123")
	t.Log(h1 == h2)
	t.Logf("%p %p", &h1, &h2)
}

func TestLen(t *testing.T) {
	var arr []int = nil
	t.Log(len(arr))
}

func TestBigMarshal(t *testing.T) {
	bi := MaxBig256
	bs, _ := (bi.MarshalText())
	t.Log(len(bs), len(bi.Bytes()))

	bi = big.NewInt(1000)
	t.Log(len(bi.Bytes()))
}
