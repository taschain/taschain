package common

import (

)

//创建一个交易账户
//s：种子字符串，为空则采用系统默认强随机数作为种子。
//返回：成功返回私钥，该私钥请妥善保管（最高优先级）。
func NewAccount(s string) (byteSk []byte) {
	sk := GenerateKey(s)
	byteSk = sk.ToBytes()
	return
}

//由交易私钥取得交易公钥
func GenAccountPubKey(sk []byte) (bytePk []byte) {
	seckey := BytesToSecKey(sk)
	if seckey != nil {
		pubk := seckey.GetPubKey()
		bytePk = pubk.ToBytes()
	}
	return
}

//由交易公钥取得交易地址
func GenAccountAddress(pk []byte) (byteAddr []byte) {
	pubk := BytesToPublicKey(pk)
	addr := pubk.GetAddress()
	byteAddr = addr.Bytes()
	return
}
