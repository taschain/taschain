package logical

import (
	"common"
	"math/big"
)

//当前节点
type node struct {
	sec_key common.PrivateKey //个人私钥（非组签名私钥）
	address common.Address    //个人地址
}

func (n node) getPubKey() *big.Int {
	return nil
}

func (n node) getGroupRandom(group_address big.Int) big.Int {
	return *big.NewInt(0)
}
