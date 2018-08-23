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
print(account.get_nonce("0x1234"))
#tas.test()`
	vm.Execute(script, nil, nil)
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

func TestVm(t *testing.T) {
	VmTestABI()
}