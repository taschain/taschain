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
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"strings"
	"common/ecies"
)

type PrivateKey struct {
	PrivKey ecdsa.PrivateKey
}

//私钥签名函数
func (pk PrivateKey) Sign(hash []byte) Sign {
	var sign Sign
	sig, err := secp256k1.Sign(hash, pk.PrivKey.D.Bytes())
	if err == nil {
		sign = *BytesToSign(sig)
	} else {
		panic(fmt.Sprintf("Sign Failed, reason : %v.\n", err.Error()))
	}

	return sign
}

//私钥生成函数
func GenerateKey(s string) PrivateKey {
	var r io.Reader
	if len(s) > 0 {
		r = strings.NewReader(s)
	} else {
		r = rand.Reader
	}
	var pk PrivateKey
	_pk, err := ecdsa.GenerateKey(getDefaultCurve(), r)
	if err == nil {
		pk.PrivKey = *_pk
	} else {
		panic(fmt.Sprintf("GenKey Failed, reason : %v.\n", err.Error()))
	}
	return pk
}

//由私钥萃取公钥函数
func (pk *PrivateKey) GetPubKey() PublicKey {
	var pubk PublicKey
	pubk.PubKey = pk.PrivKey.PublicKey
	return pubk
}

//导出函数
func (pk *PrivateKey) GetHexString() string {
	buf := pk.ToBytes()
	str := PREFIX + hex.EncodeToString(buf)
	return str
}

//导入函数
func HexStringToSecKey(s string) (sk *PrivateKey) {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return
	}
	buf, _ := hex.DecodeString(s[len(PREFIX):])
	sk = BytesToSecKey(buf)
	return
}

func (pk *PrivateKey) ToBytes() []byte {
	//fmt.Printf("begin seckey ToBytes...\n")
	pubk := pk.GetPubKey() //取得公钥
	buf := pubk.ToBytes()  //公钥序列化
	//fmt.Printf("pub key tobytes, len=%v, data=%v.\n", len(buf), buf)
	d := pk.PrivKey.D.Bytes() //D序列化
	buf = append(buf, d...)   //叠加公钥和D的序列化
	//fmt.Printf("sec key tobytes, len=%v, data=%v.\n", len(buf), buf)
	//fmt.Printf("end seckey ToBytes.\n")
	return buf
}

func BytesToSecKey(data []byte) (sk *PrivateKey) {
	//fmt.Printf("begin bytesToSecKey, len=%v, data=%v.\n", len(data), data)
	if len(data) < SecKeyLength {
		return nil
	}
	sk = new(PrivateKey)
	buf_pub := data[:PubKeyLength]
	buf_d := data[PubKeyLength:]
	sk.PrivKey.PublicKey = BytesToPublicKey(buf_pub).PubKey
	sk.PrivKey.D = new(big.Int).SetBytes(buf_d)
	if sk.PrivKey.X != nil && sk.PrivKey.Y != nil && sk.PrivKey.D != nil {
		return sk
	}
	return nil
}

//私钥解密消息
func (pk *PrivateKey) Decrypt(rand io.Reader, ct[]byte) (m []byte, err error) {
	prv := ecies.ImportECDSA(&pk.PrivKey)
	return prv.Decrypt(rand, ct, nil, nil )
}