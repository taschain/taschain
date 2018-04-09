package core

import (
	"common"
)

type Transaction struct {
	Id       common.Address
	Status   int8
	Data     []int8
	Source   common.Address
	Target   common.Address
	Gas      uint32
	Gaslimit uint32
	Gasprice uint32
	Hash     common.Hash
	ExtraData []int8
}
