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
package base

import (
	"encoding/hex"
	"strconv"
)

//把unicode字符集转化为字符串
//如输入xi := `\u5bb6\u65cf`
func MapHexToBytes(x []string) [][]byte {
	y := make([][]byte, len(x))
	for k, xi := range x {
		// TODO handle errors
		y[k], _ = hex.DecodeString(xi)
	}
	return y
}

func MapStringToBytes(x []string) [][]byte {
	y := make([][]byte, len(x))
	for k, xi := range x {
		y[k] = []byte(xi)
	}
	return y
}

//整数数组转化为字符串数组
func MapItoa(x []int) []string {
	y := make([]string, len(x))
	for k, xi := range x {
		y[k] = strconv.Itoa(xi)
	}
	return y
}

func MapShortIDToInt(x []ShortID) []int {
	y := make([]int, len(x))
	for k, xi := range x {
		y[k] = int(xi)
	}
	return y
}