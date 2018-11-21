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
	"fmt"
	"testing"
	"time"
)

func TestVmTest(t *testing.T) {
	//db, _ := tasdb.NewMemDatabase()
	//statedb, _ := core.NewAccountDB(common.Hash{}, core.NewDatabase(db))
	tt := time.Now()
	vm := NewTvm(nil, nil, "")
	vm.SetGas(9999999999999999)
	script := `

`
	vm.Execute(script)
	fmt.Println(time.Now().Sub(tt))
}

func BenchmarkAdd(b *testing.B) {
	vm := NewTvm(nil, nil, "")
	vm.SetGas(9999999999999999)
	script := `
a = 1
`
	vm.Execute(script)
	script = `
a += 1
`
	for i := 0; i < b.N; i++ { //use b.N for looping
		vm.Execute(script)
	}
}

