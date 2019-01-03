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

package types

import (
	"testing"
	"common"
	"storage/serialize"
	"fmt"
)

func TestTransaction(t *testing.T) {
	transaction := &Transaction{Value:5000,Nonce:2,GasLimit:1000000000,GasPrice:0,ExtraDataType:0}
	addr := common.HexStringToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
	transaction.Source = &addr
	fmt.Println(&addr)
	addr = common.HexStringToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4c")
	transaction.Target = &addr
	fmt.Println(&addr)
	b,_ := serialize.EncodeToBytes(transaction)
	fmt.Println(b)
	addr2 := common.HexStringToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
	transaction.Source = &addr2
	fmt.Println(&addr2)
	addr2 = common.HexStringToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4c")
	transaction.Target = &addr2
	fmt.Println(&addr2)
	c,_ := serialize.EncodeToBytes(transaction)
	fmt.Println(c)
}