package core

import (
	"common"
)

type Transaction struct {
	Id     common.Address
	Status int8
	Data   []byte
	Value  uint64
	Nonce  uint64
	Source *common.Address
	Target *common.Address

	GasLimit  uint64
	GasPrice  uint64
	Hash      common.Hash
	ExtraData []int8
}
