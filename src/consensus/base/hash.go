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
package base

import (
	"common"
	"hash"

	"golang.org/x/crypto/sha3"
)

// Hash helpers

//生成多维字节数组的SHA3_256位哈希
func HashBytes(b ...[]byte) hash.Hash {
	d := sha3.New256()
	for _, bi := range b {
		d.Write(bi)
	}
	return d
}

//生成数据的256位common.Hash
func Data2CommonHash(data []byte) common.Hash {
	var h common.Hash
	sha3_hash := sha3.Sum256(data)
	if len(sha3_hash) == common.HashLength {
		copy(h[:], sha3_hash[:])
	} else {
		panic("Data2Hash failed, size error.")
	}
	return h
}

func String2CommonHash(s string) common.Hash {
	return Data2CommonHash([]byte(s))
}
