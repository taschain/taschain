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

package mediator

import (
	"consensus/groupsig"
	"consensus/logical"
	"consensus/model"
	"consensus/net"
	"consensus/base"
)

///////////////////////////////////////////////////////////////////////////////
//共识模块提供给主框架的接口

//所有私钥，公钥，地址，ID的对外格式均为“0xa19d...854e”的加前缀十六进制格式

var Proc logical.Processor

//创建一个矿工
//id:矿工id，需要全网唯一性。
//secret：种子字符串，为空则采用系统默认强随机数作为种子。种子字符串越复杂，则矿工私钥的安全系数越高。
//返回：成功返回矿工结构，该结构包含挖矿私钥信息，请妥善保管。
func NewMiner(id string, secret string) (mi model.MinerInfo, ok bool) {
	mi = model.NewMinerInfo(id, secret)
	ok = true
	return
}

func NewMinerEx(id groupsig.ID, secret string) (mi model.MinerInfo, ok bool) {
	mi.Init(id, base.RandFromString(secret))
	ok = true
	return
}

//共识初始化
//mid: 矿工ID
//返回：true初始化成功，可以启动铸块。内部会和链进行交互，进行初始数据加载和预处理。失败返回false。
func ConsensusInit(mi model.MinerInfo) bool {
	model.InitParam()
	logical.InitConsensus()
	//groupsig.Init(1)
	ret := Proc.Init(mi)
	net.MessageHandler.Init(&Proc)
	return ret
}

//启动矿工进程，参与铸块。
//成功返回true，失败返回false。
func StartMiner() bool {
	return Proc.Start()
}

//结束矿工进程，不再参与铸块。
func StopMiner() {
	Proc.Stop()
	Proc.Finalize()
	return
}

//创建一个待初始化的新组
//返回0成功，返回<0异常。
//func CreateGroup(miners []logical.PubKeyInfo, gn string) int {
//	n := Proc.CreateDummyGroup(miners, nil, gn)
//	return n
//}
