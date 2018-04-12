package mediator

import (
	"common"
)

//所有私钥，公钥，地址，ID的对外格式均为“0xa19d...854e”的加前缀十六进制格式
//创建一个交易账户
//s：种子字符串，为空则采用系统默认强随机数作为种子。
//返回：成功返回私钥，该私钥请妥善保管。

type UserSecKey struct 
func NewUser(s string) ([]byte, bool) {
	sk := common.GenerateKey(s)
	buf := sk.ToBytes()
	return buf, true
}

/*
//由交易私钥取得交易公钥
func GenUserPKI(sk []byte) ([]byte, bool) {

}

//由交易公钥取得交易地址
func GenUserAddressI(pk []byte) ([]byte, bool) {

}

//创建一个矿工
//s：种子字符串，为空则采用系统默认强随机数作为种子。
//返回：成功返回私钥，该私钥请妥善保管。
func NewMinerI(s string) ([]byte, bool) {

}

//由矿工私钥取得矿工公钥
func GenMinerPKI(sk []byte) ([]byte, bool) {

}

//由矿工公钥取得矿工ID
func GenMinerIDI(pk []byte) ([]byte, bool) {

}

//共识初始化
//uid: 矿工ID
//返回：true初始化成功，可以启动铸块。内部会和链进行交互，进行初始数据加载和预处理。失败返回false。
func ConsensusInitI(uid []byte) bool {

}
*/
//启动矿工进程，参与铸块。
//成功返回true，失败返回false。
func StartMiner() bool {
	return true
}
