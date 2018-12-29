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
	"common/secp256k1"
	"fmt"
	"math/big"
	"encoding/hex"
)

type Sign struct {
	r big.Int
	s big.Int
	recid byte
}

//数据签名结构 for message casting
type SignData struct {
	DataHash   Hash        //哈希值
	DataSign   Sign		   //签名
	Id		   string      //用户ID
}

//签名构造函数
func (s *Sign) Set(_r, _s *big.Int, recid int) {
	s.r = *_r
	s.s = *_s
	s.recid = byte(recid)
}

//检查签名是否有效
func (s Sign) Valid() bool {
	return s.r.BitLen() != 0 && s.s.BitLen() != 0 && s.recid >= 0 && s.recid <= 4
}

//获取R值
func (s Sign) GetR() big.Int {
	return s.r
}

//获取S值
func (s Sign) GetS() big.Int {
	return s.s
}

//Sign必须65 bytes
func (s Sign) Bytes() []byte {
	rb := s.r.Bytes()
	sb := s.s.Bytes()
	r := make([]byte, SignLength)
	copy(r[32-len(rb):32], rb)
	copy(r[64-len(sb):64], sb)
	r[64] = s.recid
	return r
}

//Sign必须65 bytes
func BytesToSign(b []byte) *Sign {
	if len(b) == 65 {
		var r, s big.Int
		br := b[:32]
		r = *r.SetBytes(br)

		sr := b[32:64]
		s = *s.SetBytes(sr)

		recid := b[64]
		return &Sign{r, s, recid}
	} else {
		panic("BytesToSign must input 65 bytes!")
	}
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
	//achates testing <<
	if len(buf) != 65 {
		fmt.Println("sign length wrong in HexStringToSign! len = ", len(buf))
	}
	//>>achates testing
	si = BytesToSign(buf)
	return si
}

func (s Sign) RecoverPubkey(msg []byte) (pk *PublicKey, err error) {
	pubkey, err :=  secp256k1.RecoverPubkey(msg, s.Bytes())
	if err != nil {
		return nil, err
	}
	pk = BytesToPublicKey(pubkey)
	return
}

