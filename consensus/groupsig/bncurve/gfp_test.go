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

package bncurve

import (
	"fmt"
	"testing"
)

// Tests that negation works the same way on both assembly-optimized and pure Go
// implementation.
func TestGFpNeg(t *testing.T) {
	n := &gfP{0x0123456789abcdef, 0xfedcba9876543210, 0xdeadbeefdeadbeef, 0xfeebdaedfeebdaed}
	w := &gfP{0xfedcba9876543211, 0x0123456789abcdef, 0x2152411021524110, 0x0114251201142512}
	h := &gfP{}

	gfpNeg(h, n)
	if *h != *w {
		t.Errorf("negation mismatch: have %#x, want %#x", *h, *w)
	}
}

// Tests that addition works the same way on both assembly-optimized and pure Go
// implementation.
func TestGFp(t *testing.T) {
	a := &gfP{}
	b := &gfP{0x3c208c16d87cfd47, 0x97816a916871ca8d, 0xb85045b68181585d, 0x30644e72e131a029}
	//w := &gfP{0xc3df73e9278302b8, 0x687e956e978e3572, 0x254954275c18417f, 0xad354b6afc67f9b4}
	h := &gfP{}

	a.Invert(b)
	fmt.Println("b:", b.String())

	gfpMul(h, a, b)
	fmt.Println("h:", h.String())
	//if *h != *h {
	//	t.Errorf("addition mismatch: have %#x, want %#x", *h, *w)
	//}
}

// Tests that addition works the same way on both assembly-optimized and pure Go
// implementation.
func TestGFpAdd(t *testing.T) {
	a := &gfP{0x0123456789abcdef, 0xfedcba9876543210, 0xdeadbeefdeadbeef, 0xfeebdaedfeebdaed}
	b := &gfP{0xfedcba9876543210, 0x0123456789abcdef, 0xfeebdaedfeebdaed, 0xdeadbeefdeadbeef}
	w := &gfP{0xc3df73e9278302b8, 0x687e956e978e3572, 0x254954275c18417f, 0xad354b6afc67f9b4}
	h := &gfP{}

	gfpAdd(h, a, b)
	if *h != *w {
		t.Errorf("addition mismatch: have %#x, want %#x", *h, *w)
	}
}

// Tests that subtraction works the same way on both assembly-optimized and pure Go
// implementation.
func TestGFpSub(t *testing.T) {
	a := &gfP{0x0123456789abcdef, 0xfedcba9876543210, 0xdeadbeefdeadbeef, 0xfeebdaedfeebdaed}
	b := &gfP{0xfedcba9876543210, 0x0123456789abcdef, 0xfeebdaedfeebdaed, 0xdeadbeefdeadbeef}
	w := &gfP{0x02468acf13579bdf, 0xfdb97530eca86420, 0xdfc1e401dfc1e402, 0x203e1bfe203e1bfd}
	h := &gfP{}

	gfpSub(h, a, b)
	if *h != *w {
		t.Errorf("subtraction mismatch: have %#x, want %#x", *h, *w)
	}
}

// Tests that multiplication works the same way on both assembly-optimized and pure Go
// implementation.
func TestGFpMul(t *testing.T) {
	a := &gfP{0x0123456789abcdef, 0xfedcba9876543210, 0xdeadbeefdeadbeef, 0xfeebdaedfeebdaed}
	b := &gfP{0xfedcba9876543210, 0x0123456789abcdef, 0xfeebdaedfeebdaed, 0xdeadbeefdeadbeef}
	w := &gfP{0xcbcbd377f7ad22d3, 0x3b89ba5d849379bf, 0x87b61627bd38b6d2, 0xc44052a2a0e654b2}
	h := &gfP{}

	gfpMul(h, a, b)
	if *h != *w {
		t.Errorf("multiplication mismatch: have %#x, want %#x", *h, *w)
	}
}
