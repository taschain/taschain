package common

import "math/big"

/*
**  Creator: pxf
**  Date: 2018/10/17 下午12:30
**  Description: 
*/

var (
	//总铸块奖励
	castTotalBonus = 160

	//提案奖励比例
	propsalBonus = new(big.Int).SetUint64(50)

	//打包分红交易奖励
	packBonus = new(big.Int).SetUint64(10)

	//验证奖励
	verifyBonus = new(big.Int).SetUint64(100)
)

func GetProposalBonus() *big.Int {
	return propsalBonus
}

func GetPackBonus() *big.Int {
	return packBonus
}

func GetVerifyBonus() *big.Int {
	return verifyBonus
}