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
	"crypto/rand"
	"fmt"
	"github.com/taschain/taschain/common/ecies"
	"github.com/taschain/taschain/common/secp256k1"
	"io"
	"math/big"
	"strings"
)

type PrivateKey struct {
	PrivKey ecdsa.PrivateKey
}

//Sign message using the private key
func (pk PrivateKey) Sign(hash []byte) Sign {
	var sign Sign

	pribytes := pk.PrivKey.D.Bytes()
	seckbytes := pribytes
	if len(pribytes) < 32 {
		seckbytes = make([]byte, 32)
		copy(seckbytes[32-len(pribytes):32], pribytes) //make sure that the length of seckey is 32 bytes
	}

	sig, err := secp256k1.Sign(hash, seckbytes)
	if err == nil {
		sign = *BytesToSign(sig)
	} else {
		panic(fmt.Sprintf("Sign Failed, reason : %v.\n", err.Error()))
	}

	return sign
}

//Generate Private key by the specified string
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

//Get public key from the data struct of private key
func (pk *PrivateKey) GetPubKey() PublicKey {
	var pubk PublicKey
	pubk.PubKey = pk.PrivKey.PublicKey
	return pubk
}

//export the private key into a hex string
func (pk *PrivateKey) Hex() string {
	return ToHex(pk.Bytes())
}

//construct a private key with the hex string imported.
func HexToSecKey(s string) (sk *PrivateKey) {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return
	}
	sk = BytesToSecKey(FromHex(s))
	return
}

//export the private key into a byte array
func (pk *PrivateKey) Bytes() []byte {
	buf := make([]byte, SecKeyLength)
	copy(buf[:PubKeyLength], pk.GetPubKey().Bytes())
	d := pk.PrivKey.D.Bytes()
	if len(d) > 32 {
		panic("privateKey data length error: D length is more than 32!")
	}
	copy(buf[SecKeyLength-len(d):SecKeyLength], d)
	return buf
}

//construct a private key with the byte array imported
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

//Decrypt function using the private key
func (pk *PrivateKey) Decrypt(rand io.Reader, ct []byte) (m []byte, err error) {
	prv := ecies.ImportECDSA(&pk.PrivKey)
	return prv.Decrypt(rand, ct, nil, nil)
}
