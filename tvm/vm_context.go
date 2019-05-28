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