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

// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sha3

// This file defines the ShakeHash interface, and provides
// functions for creating SHAKE instances, as well as utility
// functions for hashing bytes to arbitrary-length output.

import (
	"io"
)

// ShakeHash defines the interface to hash functions that
// support arbitrary-length output.
type ShakeHash interface {
	// Write absorbs more data into the hash's state. It panics if input is
	// written to it after output has been read from it.
	io.Writer

	// Read reads more output from the hash; reading affects the hash's
	// state. (ShakeHash.Read is thus very different from Hash.Sum)
	// It never returns an error.
	io.Reader

	// Clone returns a copy of the ShakeHash in its current state.
	Clone() ShakeHash

	// Reset resets the ShakeHash to its initial state.
	Reset()
}

func (d *state) Clone() ShakeHash {
	return d.clone()
}

// NewShake128 creates a new SHAKE128 variable-output-length ShakeHash.
// Its generic security strength is 128 bits against all attacks if at
// least 32 bytes of its output are used.
func NewShake128() ShakeHash { return &state{rate: 168, dsbyte: 0x1f} }

// NewShake256 creates a new SHAKE128 variable-output-length ShakeHash.
// Its generic security strength is 256 bits against all attacks if
// at least 64 bytes of its output are used.
func NewShake256() ShakeHash { return &state{rate: 136, dsbyte: 0x1f} }

// ShakeSum128 writes an arbitrary-length digest of data into hash.
func ShakeSum128(hash, data []byte) {
	h := NewShake128()
	h.Write(data)
	h.Read(hash)
}

// ShakeSum256 writes an arbitrary-length digest of data into hash.
func ShakeSum256(hash, data []byte) {
	h := NewShake256()
	h.Write(data)
	h.Read(hash)
}
