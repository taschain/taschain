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

package rand

import (
	"testing"
	"math/big"
	"math/rand"
)

/*
**  Creator: pxf
**  Date: 2018/6/15 下午4:39
**  Description: 
*/

func TestRandSeed(t *testing.T) {
	b := big.NewInt(100000)

	r := NewFromSeed(b.Bytes())
	t.Log(r)

	r = NewFromSeed(b.Bytes())
	t.Log(r)

	r = NewFromSeed(b.Bytes())
	t.Log(r)
}

func TestMathRand(t *testing.T) {
	s := rand.NewSource(1000000)
	r := rand.New(s)


	t.Log(r.Uint64())
	t.Log(r.Uint64())
	t.Log(r.Uint64())
}

func TestMathRand2(t *testing.T) {
	rand.Seed(1000000)
	t.Log(rand.Uint64())
	t.Log(rand.Uint64())
	t.Log(rand.Uint64())
	t.Log(rand.Uint64())
}