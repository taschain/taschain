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
	fmt.Printf("groupsig init success, curve_order=%v, field_order=%v, bitlen=%v.\n",
		curveOrder.String(), fieldOrder.String(), bitLength)
	return
}
