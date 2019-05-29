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

var InstanceIndex int
var BootID int

type AccountData struct {
	sk   []byte //secure key
	pk   []byte //public key
	addr []byte //address
}

//创建一个交易账户
//s：种子字符串，为空则采用系统默认强随机数作为种子。
//返回：成功返回账户信息（私钥，公钥和地址），该私钥请妥善保管（最高优先级）。
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
