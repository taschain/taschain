package core

import (
	"common"
)

type Transaction struct {
	id       common.Address
	status   int8
	data     []int8
	source   common.Address
	target   common.Address
	gas      uint32
	gaslimit uint32
	gasprice uint32
	hash     common.Hash
}
