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
	"common/ecies"
	"common/secp256k1"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"golang.org/x/crypto/sha3"
	"io"
)

//用户公钥
type PublicKey struct {
	PubKey ecdsa.PublicKey
}

//公钥验证函数
func (pk PublicKey) Verify(hash []byte, s *Sign) bool {
	//return ecdsa.Verify(&pk.PubKey, hash, &s.r, &s.s)
	rb := s.r.Bytes()
	sb := s.s.Bytes()
	sig := make([]byte, len(rb)+len(sb))
	copy(sig, rb)
	copy(sig[len(rb):], sb)
	return secp256k1.VerifySignature(pk.ToBytes(), hash, sig)
}

//由公钥萃取地址函数
func (pk PublicKey) GetAddress() Address {
	x := pk.PubKey.X.Bytes()
	y := pk.PubKey.Y.Bytes()
	x = append(x, y...)

	addr_buf := sha3.Sum256(x)
	if len(addr_buf) != AddressLength {
		panic("地址长度错误")
	}
	Addr := BytesToAddress(addr_buf[:])
	return Addr
}

//把公钥转换成字节切片
func (pk PublicKey) ToBytes() []byte {
	buf := elliptic.Marshal(pk.PubKey.Curve, pk.PubKey.X, pk.PubKey.Y)
	//fmt.Printf("end pub key marshal, len=%v, data=%v\n", len(buf), buf)
	return buf
}

//从字节切片转换到公钥
func BytesToPublicKey(data []byte) (pk *PublicKey) {
	pk = new(PublicKey)
	pk.PubKey.Curve = getDefaultCurve()
	//fmt.Printf("begin pub key unmarshal, len=%v, data=%v.\n", len(data), data)
	x, y := elliptic.Unmarshal(pk.PubKey.Curve, data)
	if x == nil || y == nil {
		panic("unmarshal public key failed.")
	}
	pk.PubKey.X = x
	pk.PubKey.Y = y
	return
}

//导出函数
func (pk PublicKey) GetHexString() string {
	buf := pk.ToBytes()
	str := PREFIX + hex.EncodeToString(buf)
	return str
}

//导入函数
func HexStringToPubKey(s string) (pk *PublicKey) {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return
	}
	buf, _ := hex.DecodeString(s[len(PREFIX):])
	pk = BytesToPublicKey(buf)
	return
}

//公钥加密消息
func Encrypt(rand io.Reader, pub *PublicKey, msg []byte) (ct []byte, err error) {
	pubECIES := ecies.ImportECDSAPublic(&pub.PubKey)
	return ecies.Encrypt(rand, pubECIES, msg, nil, nil)
}