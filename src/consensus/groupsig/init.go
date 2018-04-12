package groupsig

import (
	//"github.com/dfinity/go-dfinity-crypto/bls"
	"consensus/bls"
)

const PREFIX = "0x"

// Init --
func Init(curve int) {
	err := bls.Init(curve) //以特定的椭圆曲线初始化BLS C库
	if err != nil {
		panic("groupsig.Init")
	}
	curveOrder.SetString(bls.GetCurveOrder(), 10)
	fieldOrder.SetString(bls.GetFieldOrder(), 10)
	bitLength = curveOrder.BitLen()
}
