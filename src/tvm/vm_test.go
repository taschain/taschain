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

package tvm

import (
	"testing"
	"encoding/json"
	"fmt"
)

func TestVmTest(t *testing.T) {
	//db, _ := tasdb.NewMemDatabase()
	//statedb, _ := core.NewAccountDB(common.Hash{}, core.NewDatabase(db))
	vm := NewTvm(nil, nil, "")
//	script := `
//import utime
//print(utime.time())
//`
//	vm.Execute(script)

	abiJson := `{"FuncName": "Test", "Args": [10, "ten", [1, 2], {"key":"value", "key2":"value2"}]}`;//fmt.Sprintf(`{"FuncName": "%s", "Args": %s}`, task.FuncName, task.Params)
	abi := ABI{}
	json.Unmarshal([]byte(abiJson), &abi)

	str := PycodeCheckAbi(abi)
	fmt.Println(str)
	ret := vm.Execute(str)
	fmt.Println(ret)
}

