package common

import (

)

type AccountData struct {
	sk   []byte   //secure key
	pk 	 []byte   //public key
	addr []byte   //address
}

//创建一个交易账户
//s：种子字符串，为空则采用系统默认强随机数作为种子。
//返回：成功返回私钥，该私钥请妥善保管（最高优先级）。
func NewAccount(s string) (account AccountData) {
	seckey := GenerateKey(s)
	pubkey := seckey.GetPubKey()
	addr := pubkey.GetAddress()

	account.sk = seckey.ToBytes()
	account.pk = pubkey.ToBytes()
	account.addr = addr.Bytes()
	return account
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
