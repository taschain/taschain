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
	"common"
	"storage/core"
	"storage/tasdb"
)

func TestVmTest(t *testing.T) {
	db, _ := tasdb.NewMemDatabase()
	statedb, _ := core.NewAccountDB(common.Hash{}, core.NewDatabase(db))
	vm := NewTvm(statedb, nil)
	script := `

import account
account.create_account("0x2234")
value = account.get_balance("0x1234")
value = account.add_balance("0x1234",10)
account.set_nonce("0x1234", -1)
print("")
print(account.get_nonce("0x1234"))
#tas.test()`
	vm.Execute(script,nil, nil)
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
	vm.Execute(script,nil, nil)
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
	vm.Execute(script,nil, nil)
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
	vm.Execute(script,nil, nil)
}


// 创建账户

// 部署合约

// 执行合约