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
	"math/big"
)

type Sign struct {
	r big.Int
	s big.Int
}

//数据签名结构 for message casting
type SignData struct {
	DataHash   Hash        //哈希值
	DataSign   Sign				 //签名
	Id		   string            //用户ID
}

//签名构造函数
func (s *Sign) Set(_r, _s *big.Int) {
	s.r = *_r
	s.s = *_s
}

//检查签名是否有效
func (s Sign) Valid() bool {
	return s.r.BitLen() != 0 && s.s.BitLen() != 0
}

//获取R值
func (s Sign) GetR() big.Int {
	return s.r
}

//获取S值
func (s Sign) GetS() big.Int {
	return s.s
}

func (s Sign) Bytes() []byte {
	rb := s.r.Bytes()
	sb := s.s.Bytes()
	r := make([]byte, len(rb)+len(sb))
	copy(r, rb)
	copy(r[len(rb):], sb)
	return r
}

func BytesToSign(b []byte) *Sign {
	var r, s big.Int
	br := b[:32]
	r = *r.SetBytes(br)

	sr := b[32:]
	s = *s.SetBytes(sr)
	return &Sign{r, s}
}

func (s Sign) GetHexString() string {
	buf := s.Bytes()
	str := PREFIX + hex.EncodeToString(buf)
	return str
}

//导入函数
func HexStringToSign(s string) (si *Sign) {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return
	}
	buf, _ := hex.DecodeString(s[len(PREFIX):])
	si = BytesToSign(buf)
	return si
}

