package common

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha1"
)

type PublicKey struct {
	PubKey ecdsa.PublicKey
}

//公钥验证函数
func (pk PublicKey) Verify(hash []byte, s *Sign) bool {
	return ecdsa.Verify(&pk.PubKey, hash, &s.r, &s.s)
}

//由公钥萃取地址函数
func (pk PublicKey) GetAddress() Address {
	x := pk.PubKey.X.Bytes()
	y := pk.PubKey.Y.Bytes()
	x = append(x, y...)
	fix_buf := sha1.Sum(x)
	var addr_buf []byte = fix_buf[0:]
	if len(addr_buf) != AddressLength {
		panic("地址长度错误")
	}
	Addr := BytesToAddress(addr_buf)
	return Addr
}

//把公钥转换成字节切片
func (pk PublicKey) ToBytes() []byte {
	return elliptic.Marshal(pk.PubKey.Curve, pk.PubKey.X, pk.PubKey.Y)
	//x := pk.PubKey.X.Bytes()
	//y := pk.PubKey.Y.Bytes()
	//x = append(x, y...)
	//return x
}

//从字节切片转换到公钥
func BytesToPublicKey(data []byte) (pk PublicKey) {
	pk.PubKey.Curve = elliptic.P256()
	x, y := elliptic.Unmarshal(pk.PubKey.Curve, data)
	if x == nil || y == nil {
		panic("unmarshal public key failed.")
	}
	pk.PubKey.X = x
	pk.PubKey.Y = y
	return
}
