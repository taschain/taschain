package groupsig

import (
	"consensus/bls"
	"fmt"
)

const PREFIX = "0x"

const (
	Curve254     = 0 //256位曲线
	Curve382_1   = 1 //384位曲线1
	Curve382_2   = 2 //384位曲线2
	DefaultCurve = 1 //默认使用的曲线
	//默认曲线相关参数开始：如默认曲线的位数调整，则这些参数也需要修改
	IDLENGTH     = 48 //ID字节长度(384位，同私钥长度)
	PUBKEYLENGTH = 96 //公钥字节长度（768位）
	SECKEYLENGTH = 48 //私钥字节长度（384位）
	SIGNLENGTH   = 48 //签名字节长度（384位）
	//默认曲线相关参数结束。
	HASHLENGTH = 32 //哈希字节长度(golang.SHA3, 256位。和common包相同)
)

// Init --
func Init(curve int) {
	fmt.Printf("\nbegin groupsig init, curve=%v.\n", curve)
	err := bls.Init(curve) //以特定的椭圆曲线初始化BLS C库
	if err != nil {
		panic("groupsig.Init")
	}
	curveOrder.SetString(bls.GetCurveOrder(), 10)
	fieldOrder.SetString(bls.GetFieldOrder(), 10)
	bitLength = curveOrder.BitLen()
	fmt.Printf("end groupsig init.")
}
