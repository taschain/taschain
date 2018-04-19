package util

import (
	"common"
	eth "vm/common"
)

/*
**  Creator: pxf
**  Date: 2018/4/16 下午12:05
**  Description: 
*/

func Int8ToBytes(data []int8) []byte {
	if len(data) == 0 {
		return []byte{}
	}

	var ret []byte
	for _, i := range data {
		ret = append(ret, byte(i))
	}
	return ret
}

func Byte2Int8(data []byte) []int8 {
	if len(data) == 0 {
		return []int8{}
	}

	var ret []int8
	for _, i := range data {
		ret = append(ret, int8(i))
	}
	return ret
}

func ToETHAddress(addr common.Address) eth.Address {
	return eth.BytesToAddress(addr.Bytes())
}

func ToTASAddress(addr eth.Address) common.Address {
	return common.BytesToAddress(addr.Bytes())
}