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

/*
 * 这个用来保存vm：当前设计只考虑单线程，因为区块链交易执行是单线程的
 */

var controller *Controller // vm的controller

const MAX_DEPTH int = 8 //vm执行的最大深度为8

// 合约调合约场景中（从c回调go时）执行合约call前保存
func (con *Controller) StoreVmContext(newTvm *Tvm) bool {
	if len(con.VmStack) >= MAX_DEPTH {
		print("===== too many call  levels ====")
		return false
	}

	currentVm := con.Vm
	con.VmStack = append(con.VmStack, currentVm)
	con.Vm = newTvm
	return true
}

// 恢复tvm上下文
func (con *Controller) RecoverVmContext() {
	con.Vm = con.VmStack[len(con.VmStack)-1]
	con.VmStack = con.VmStack[:len(con.VmStack)-1]
}
