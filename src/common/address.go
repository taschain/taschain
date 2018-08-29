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

package common

import (
	"crypto/rand"
)

// RandomAddress - Generate a random address.
//生成一个随机地址。意义何在，如果地址不是从公钥萃取的话？
func RandomAddress() (*Address, error) {
	bytes := make([]byte, AddressLength) //分配地址？
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, err
	}
	address := BytesToAddress(bytes)
	return &address, nil
}

//生成一个随机地址列表
func RandomAddresses(n int) ([]Address, error) {
	addresses := make([]Address, n) //地址数量
	for i := range addresses {
		ptr, err := RandomAddress()
		addresses[i] = *ptr
		if err != nil {
			return nil, err
		}
	}
	return addresses, nil
}

//对addresses从l->r区间内的地址按从小到大进行16进制排序
func sortByHex(addresses []Address, l int, r int) {
	if l < r {
		pivot := addresses[(l+r)/2].GetHexString()
		i := l
		j := r
		var tmp Address
		for i <= j {
			for addresses[i].GetHexString() < pivot {
				i++
			}
			for addresses[j].GetHexString() > pivot {
				j--
			}
			if i <= j {
				tmp = addresses[i]
				addresses[i] = addresses[j]
				addresses[j] = tmp
				i++
				j--
			}
		}
		if l < j {
			sortByHex(addresses, l, j)
		}
		if i < r {
			sortByHex(addresses, i, r)
		}
	}
}

//对地址列表按十六进制排序
func SortAddresses(addresses []Address) {
	n := len(addresses)
	sortByHex(addresses, 0, n-1)
}
