package common

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"strings"
	"utility/ecies"
)

type PrivateKey struct {
	PrivKey ecdsa.PrivateKey
}

//私钥签名函数
func (pk PrivateKey) Sign(hash []byte) Sign {
	var sign Sign
	r, s, err := ecdsa.Sign(rand.Reader, &pk.PrivKey, hash)
	if err == nil {
		sign.Set(r, s)
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
func (pk PrivateKey) GetPubKey() PublicKey {
	var pubk PublicKey
	pubk.PubKey = pk.PrivKey.PublicKey
	return pubk
}

//导出函数
func (pk *PrivateKey) GetHexString() string {
	buf := pk.toBytes()
	str := PREFIX + hex.EncodeToString(buf)
	return str
}

//导入函数
func HexStringToSecKey(s string) (sk *PrivateKey) {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return
	}
	buf, _ := hex.DecodeString(s[len(PREFIX):])
	sk = bytesToSecKey(buf)
	return
}

func (pk *PrivateKey) toBytes() []byte {
	fmt.Printf("begin seckey ToBytes...\n")
	pubk := pk.GetPubKey() //取得公钥
	buf := pubk.toBytes()  //公钥序列化
	fmt.Printf("pub key tobytes, len=%v, data=%v.\n", len(buf), buf)
	d := pk.PrivKey.D.Bytes() //D序列化
	buf = append(buf, d...)   //叠加公钥和D的序列化
	fmt.Printf("sec key tobytes, len=%v, data=%v.\n", len(buf), buf)
	fmt.Printf("end seckey ToBytes.\n")
	return buf
}

func bytesToSecKey(data []byte) (sk *PrivateKey) {
	fmt.Printf("begin bytesToSecKey, len=%v, data=%v.\n", len(data), data)
	if len(data) < SecKeyLength {
		return nil
	}
	sk = new(PrivateKey)
	buf_pub := data[:PubKeyLength]
	buf_d := data[PubKeyLength:]
	sk.PrivKey.PublicKey = bytesToPublicKey(buf_pub).PubKey
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