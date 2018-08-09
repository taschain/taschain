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

package utility

import (
	"bytes"
	"encoding/binary"
)

func UInt32ToByte(i uint32) []byte {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, i)
	return buf.Bytes()
}

func ByteToUInt32(b []byte) uint32 {
	buf := bytes.NewBuffer(b)
	var x uint32
	binary.Read(buf, binary.BigEndian, &x)
	return x
}

func IntToByte(i int) []byte {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, i)
	return buf.Bytes()
}

func ByteToInt(b []byte) int {
	buf := bytes.NewBuffer(b)
	var x int
	binary.Read(buf, binary.BigEndian, &x)
	return x
}

func UInt64ToByte(i uint64) []byte {
buf := bytes.NewBuffer([]byte{})
binary.Write(buf, binary.BigEndian, i)
return buf.Bytes()
}

func ByteToUInt64(b []byte) uint64 {
	buf := bytes.NewBuffer(b)
	var x uint64
	binary.Read(buf, binary.BigEndian, &x)
	return x
}

