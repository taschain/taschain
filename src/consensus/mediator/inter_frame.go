package mediator

import (
	"common"
)

///////////////////////////////////////////////////////////////////////////////
//共识模块提供给主框架的接口

//创建一个账户
//s：种子字符串，为空则采用系统默认强随机数作为种子。
//返回：成功返回私钥，失败返回nil。
type NewAccount func(s string) *common.PrivateKey

//共识初始化
//a: 铸块节点地址
//返回：true初始化成功，可以启动铸块。内部会和链进行交互，进行初始数据加载和预处理。失败返回false。
type ConsensusInit func(a common.Address) bool

//启动矿工进程，参与铸块。
//成功返回true，失败返回false。
type StartMiner func() bool
