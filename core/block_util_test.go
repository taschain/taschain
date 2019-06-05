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

package core

import (
	"testing"

	"github.com/taschain/taschain/middleware/types"
)

func TestCalTree(t *testing.T) {
	tx1 := getRandomTxs()
	tree1 := calcTxTree(tx1)

	if tree1.Hex() != "0x5a312281df4bd8dfbb4d4a94ad0bf44d01bb8cfced1206b90e21b4ca0568cdb1" {
		t.Errorf("mismatch, expect 0x5a312281df4bd8dfbb4d4a94ad0bf44d01bb8cfced1206b90e21b4ca0568cdb1 but got get %s ", tree1.Hex())
	}
}

func getRandomTxs() []*types.Transaction {
	result := make([]*types.Transaction, 0)
	var i uint64
	for i = 0; i < 100; i++ {
		tx := types.Transaction{Nonce: i, Value: 100 - i}
		result = append(result, &tx)
	}
	return result
}
