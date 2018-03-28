package common

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
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
func GenerateKey() PrivateKey {
	var pk PrivateKey
	_pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
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

func (pk PrivateKey) ToBytes() []byte {
	pubk := pk.GetPubKey()
	buf := pubk.ToBytes()
	D := pk.PrivKey.D.Bytes()
	buf = append(buf, D...)
	return buf
}
