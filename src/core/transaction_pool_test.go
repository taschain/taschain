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
	"fmt"
	"common"
	"middleware/types"
)

func TestCreatePool(t *testing.T) {

	pool := NewTransactionPool()

	fmt.Printf("received: %d transactions\n", pool.received.Len())

	transaction := &types.Transaction{
		GasPrice: 1234,
	}

	pool.Add(transaction)
	fmt.Printf("received: %d transactions\n", pool.received.Len())

	h := common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

	transaction = &types.Transaction{
		GasPrice: 12345,
		Hash:     h,
	}

	pool.Add(transaction)
	fmt.Printf("received: %d transactions\n", pool.received.Len())

	tGet, error := pool.GetTransaction(h)
	if nil == error {
		fmt.Printf("GasPrice: %d\n", tGet.GasPrice)
	}

	casting := pool.GetTransactionsForCasting()
	fmt.Printf("length for casting: %d\n", len(casting))

	fmt.Printf("%d\n", casting[0])
	//fmt.Printf("%d\n", casting[1])
	//fmt.Printf("%d\n", casting[2].gasprice)
	//fmt.Printf("%d\n", casting[3].gasprice)

}

func TestContainer(t *testing.T) {
	pool := NewTransactionPool()
	pool.received.limit = 1
	h := common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	e := common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b42")

	transaction := &types.Transaction{
		GasPrice: 1234,
		Hash:     e,
	}

	pool.Add(transaction)
	fmt.Printf("received: %d transactions\n", pool.received.Len())

	transaction = &types.Transaction{
		GasPrice: 12345,
		Hash:     h,
	}

	pool.Add(transaction)
	fmt.Printf("received: %d transactions\n", pool.received.Len())

	tGet, error := pool.GetTransaction(h)
	if nil == error {
		fmt.Printf("%d\n", tGet.GasPrice)
	}

	tGet, _ = pool.GetTransaction(e)
	if nil != tGet {
		fmt.Printf("%d\n", tGet.GasPrice)
	} else {
		fmt.Printf("success %x\n", e)
	}
}
