package mediator

import (
	"consensus/groupsig"
	"consensus/logical"
	"consensus/rand"
)

///////////////////////////////////////////////////////////////////////////////
//共识模块提供给主框架的接口

//所有私钥，公钥，地址，ID的对外格式均为“0xa19d...854e”的加前缀十六进制格式

var Proc logical.Processer

//创建一个矿工
//id:矿工id，需要全网唯一性。
//secret：种子字符串，为空则采用系统默认强随机数作为种子。种子字符串越复杂，则矿工私钥的安全系数越高。
//返回：成功返回矿工结构，该结构包含挖矿私钥信息，请妥善保管。
func NewMiner(id string, secret string) (mi logical.MinerInfo, ok bool) {
	mi = logical.NewMinerInfo(id, secret)
	ok = true
	return
}

func NewMinerEx(id groupsig.ID, secret string) (mi logical.MinerInfo, ok bool) {
	mi.Init(id, rand.RandFromString(secret))
	ok = true
	return
}

//共识初始化
//mid: 矿工ID
//返回：true初始化成功，可以启动铸块。内部会和链进行交互，进行初始数据加载和预处理。失败返回false。
func ConsensusInit(mi logical.MinerInfo) bool {
	groupsig.Init(1)
	return Proc.Init(mi)
}

//启动矿工进程，参与铸块。
//成功返回true，失败返回false。
func StartMiner() bool {
	return Proc.Start()
}

//结束矿工进程，不再参与铸块。
func StopMiner() {
	return
}
