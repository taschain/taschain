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
	"encoding/hex"
	"math/big"

	"github.com/taschain/taschain/common/secp256k1"
)

//Data struct of signature
type Sign struct {
	r     big.Int
	s     big.Int
	recid byte
}

//data struct for message casting
type SignData struct {
	DataHash Hash   //哈希值
	DataSign Sign   //签名
	ID       string //用户ID
}

//signature construct function
func (s *Sign) Set(_r, _s *big.Int, recid int) {
	s.r = *_r
	s.s = *_s
	s.recid = byte(recid)
}

//Check the signature is valid
func (s Sign) Valid() bool {
	return s.r.BitLen() != 0 && s.s.BitLen() != 0 && s.recid < 4
}

//Get r value
func (s Sign) GetR() big.Int {
	return s.r
}

//Get s value
func (s Sign) GetS() big.Int {
	return s.s
}

//Export the signature into a byte array.
func (s Sign) Bytes() []byte {
	rb := s.r.Bytes()
	sb := s.s.Bytes()
	r := make([]byte, SignLength)
	copy(r[32-len(rb):32], rb)
	copy(r[64-len(sb):64], sb)
	r[64] = s.recid
	return r
}

//Construct a signature with the byte array imported. The length of the byte array must be 65.
func BytesToSign(b []byte) *Sign {
	if len(b) == 65 {
		var r, s big.Int
		br := b[:32]
		r = *r.SetBytes(br)

		sr := b[32:64]
		s = *s.SetBytes(sr)

		recid := b[64]
		return &Sign{r, s, recid}
	}
	panic("BytesToSign must input 65 bytes!")
}

//Export the signature into a hex string
func (s Sign) Hex() string {
	return ToHex(s.Bytes())
}

//Construct a signature with the hex string imported
func HexToSign(s string) (si *Sign) {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return
	}
	buf, _ := hex.DecodeString(s[len(PREFIX):])
	si = BytesToSign(buf)
	return si
}

//Recover the public key from the signature
func (s Sign) RecoverPubkey(msg []byte) (pk *PublicKey, err error) {
	pubkey, err := secp256k1.RecoverPubkey(msg, s.Bytes())
	if err != nil {
		return nil, err
	}
	pk = BytesToPublicKey(pubkey)
	return
}
