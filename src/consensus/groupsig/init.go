package groupsig

const (
	Curve254     = 0 //256位曲线
	Curve382_1   = 1 //384位曲线1
	Curve382_2   = 2 //384位曲线2
	DefaultCurve = 1 //默认使用的曲线
	//默认曲线相关参数开始：如默认曲线的位数调整，则这些参数也需要修改
	IDLENGTH     = 32 //ID字节长度(256位，同私钥长度)
	PUBKEYLENGTH = 128 //公钥字节长度（1024位）
	SECKEYLENGTH = 32 //私钥字节长度（256位）
	SIGNLENGTH   = 64 //签名字节长度（512位）
	//默认曲线相关参数结束。
	HASHLENGTH   = 32 //哈希字节长度(golang.SHA3, 256位。和common包相同)
)

//// Init --
//func Init(curve int) {
//	//fmt.Printf("\nbegin groupsig init, curve=%v.\n", curve)
//	//err := bn_curve.Init(curve) //以特定的椭圆曲线初始化BLS C库
//	//if err != nil {
//	//	panic("groupsig.Init")
//	//}
//	//curveOrder.SetString(bn_curve.GetCurveOrder(), 10)
//	//fieldOrder.SetString(bn_curve.GetFieldOrder(), 10)
//
//	bitLength = curveOrder.BitLen()
//	fmt.Printf("groupsig init success, curve_order=%v, field_order=%v, bitlen=%v.\n",
//		curveOrder.String(), fieldOrder.String(), bitLength)
//	return
//}
