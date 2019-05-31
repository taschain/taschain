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
	"crypto/ecdsa"
	"crypto/elliptic"
	"github.com/taschain/taschain/common/ecies"
	"github.com/taschain/taschain/common/secp256k1"
	"golang.org/x/crypto/sha3"
	"io"
)

//Data struct of the public key
type PublicKey struct {
	PubKey ecdsa.PublicKey
}

//Verify the signature and message (hash) by the public key
func (pk PublicKey) Verify(hash []byte, s *Sign) bool {
	return secp256k1.VerifySignature(pk.Bytes(), hash, s.Bytes()[:64])
}

//Get the address mapped from the public key
func (pk PublicKey) GetAddress() Address {
	x := pk.PubKey.X.Bytes()
	y := pk.PubKey.Y.Bytes()
	x = append(x, y...)

	addr_buf := sha3.Sum256(x)
	if len(addr_buf) != AddressLength {
		panic("地址长度错误")
	}
	return BytesToAddress(addr_buf[:])
}

//Export the public key into a byte array
func (pk PublicKey) Bytes() []byte {
	buf := elliptic.Marshal(pk.PubKey.Curve, pk.PubKey.X, pk.PubKey.Y)
	//fmt.Printf("end pub key marshal, len=%v, data=%v\n", len(buf), buf)
	return buf
}

//Construct a public key with the byte array imported
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

//Export the public key into a hex string
func (pk PublicKey) Hex() string {
	return ToHex(pk.Bytes())
}

//Encrypt the message using the public key
func (pk *PublicKey) Encrypt(rand io.Reader, msg []byte) ([]byte, error) {
	return Encrypt(rand, pk, msg)
}

//Construct a public key with the hex string imported
func HexToPubKey(s string) (pk *PublicKey) {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return
	}
	pk = BytesToPublicKey(FromHex(s))
	return
}

//Encrypt the message using the ECIES method
func Encrypt(rand io.Reader, pub *PublicKey, msg []byte) (ct []byte, err error) {
	pubECIES := ecies.ImportECDSAPublic(&pub.PubKey)
	return ecies.Encrypt(rand, pubECIES, msg, nil, nil)
}
