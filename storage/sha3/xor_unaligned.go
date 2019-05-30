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

// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build amd64 386 ppc64le
// +build !appengine

package sha3

import "unsafe"

func xorInUnaligned(d *state, buf []byte) {
	bw := (*[maxRate / 8]uint64)(unsafe.Pointer(&buf[0]))
	n := len(buf)
	if n >= 72 {
		d.a[0] ^= bw[0]
		d.a[1] ^= bw[1]
		d.a[2] ^= bw[2]
		d.a[3] ^= bw[3]
		d.a[4] ^= bw[4]
		d.a[5] ^= bw[5]
		d.a[6] ^= bw[6]
		d.a[7] ^= bw[7]
		d.a[8] ^= bw[8]
	}
	if n >= 104 {
		d.a[9] ^= bw[9]
		d.a[10] ^= bw[10]
		d.a[11] ^= bw[11]
		d.a[12] ^= bw[12]
	}
	if n >= 136 {
		d.a[13] ^= bw[13]
		d.a[14] ^= bw[14]
		d.a[15] ^= bw[15]
		d.a[16] ^= bw[16]
	}
	if n >= 144 {
		d.a[17] ^= bw[17]
	}
	if n >= 168 {
		d.a[18] ^= bw[18]
		d.a[19] ^= bw[19]
		d.a[20] ^= bw[20]
	}
}

func copyOutUnaligned(d *state, buf []byte) {
	ab := (*[maxRate]uint8)(unsafe.Pointer(&d.a[0]))
	copy(buf, ab[:])
}

var (
	xorIn   = xorInUnaligned
	copyOut = copyOutUnaligned
)

const xorImplementationUnaligned = "unaligned"
