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
	"math/rand"
	"testing"
	"time"
)

func TestSortAddresses(test *testing.T) {
	rand.Seed(time.Now().UnixNano()) //以当前时间值作为随机数种子
	n := rand.Intn(10)               //生成10个随机数？
	addresses, err := RandomAddresses(n)
	if err != nil {
		test.Fatal(err)
	}
	SortAddresses(addresses)
	for i := 0; i < n-1; i++ {
		if addresses[i].GetHexString() > addresses[i+1].GetHexString() {
			test.Fatal(addresses)
		}
	}
}
