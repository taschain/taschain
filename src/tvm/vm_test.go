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
)

func TestVmTest(t *testing.T) {
	//db, _ := tasdb.NewMemDatabase()
	//statedb, _ := core.NewAccountDB(common.Hash{}, core.NewDatabase(db))
	vm := NewTvm()
	script := `
import account
account.create_account("0x2234")
value = account.get_balance("0x1234")
value = account.add_balance("0x1234",10)
account.set_nonce("0x1234", -1)
print(account.get_nonce("0x1234"))
#tas.test()`
	vm.Execute(script)
}

func TestVmTestContract(t *testing.T) {
	VmTestContract()
}

func TestVmTestClass(t *testing.T) {
	VmTestClass()
}

func MockBlockChain() {

}

func TestVmTestABI(t *testing.T) {
	VmTestABI()
}

func TestVmTestException(t *testing.T) {
	VmTestException()
}

func TestVmTestToken(t *testing.T) {
	VmTestToken()
}

func TestVmTest2(t *testing.T) {
	VmTest()
}

// 设置python lib目录
func TestVmTest3(t *testing.T) {
	vm := NewTvmTest(nil, nil)
	script := `
from test import test_lib_helloworld

test_lib_helloworld.helloworld()

`
	vm.Execute(script)
}

// msg变量
func TestVmTest4(t *testing.T) {
	vm := NewTvmTest(nil, nil)
	script := `
from clib.tas_runtime import glovar
from clib.tas_runtime.msgxx import Msg
from clib.tas_runtime.address_tas import Address

glovar.msg = Msg(data="", sender=Address(""), value=100)

print(glovar.msg)
`
	vm.Execute(script)
}

// Address.call
func TestVmTest5(t *testing.T) {
	vm := NewTvmTest(nil, nil)
	script := `
from clib.tas_runtime import glovar
from clib.tas_runtime.msgxx import Msg
from clib.tas_runtime.address_tas import Address

glovar.msg = Msg(data="", sender=Address(""), value=100)
print(glovar.msg)

from token.contract_token_tas import MyAdvancedToken

`
	vm.Execute(script)
}

// TVM释放
func TestVmTest6(t *testing.T) {
	vm := NewTvm()
	vm.init(nil, nil, nil, nil)
	vm.AddLibPath("/Users/guangyujing/workspace/tas/src/tvm/py")

	script := `
from test import test_lib_helloworld

test_lib_helloworld.helloworld()

`
	vm.Execute(script)

	vm.DelTvm()

	vm = NewTvm()
	vm.init(nil, nil, nil, nil)
	vm.AddLibPath("/Users/guangyujing/workspace/tas/src/tvm/py")

	script = `
test_lib_helloworld.helloworld()
`
	vm.Execute(script)

	vm.DelTvm()
}

func TestVmTest7(t *testing.T) {
	vm := NewTvm()
	vm.AddLibPath("/Users/guangyujing/workspace/tas/src/tvm/py")
	vm.ContractName = "MyAdvancedToken"
	vm.ContractAddress = "0x2"
	vm.init(nil, nil, nil, nil)
	vm.SetGas(10000000)

	code := Read0("/Users/guangyujing/workspace/tas/src/tvm/py/token/contract_token_tas.py")
	vm.Execute(code)

	msg := Msg{}
	msg.Value = 0
	msg.Sender = "0x1"
	vm.Deploy(msg)

	j := `{"FuncName": "test", "Args": []}`
	vm.ExecuteABIJson(msg, j)

	j = `{"FuncName": "transfer", "Args": ["0x3", 1000]}`
	vm.ExecuteABIJson(msg, j)
}