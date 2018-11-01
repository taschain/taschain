package tvm

/*
 * 这个用来保存vm：当前设计只考虑单线程，因为区块链交易执行是单线程的
 */

var controller *Controller; // vm的controller

const MAX_DEPTH int = 128;

// 合约调合约场景中（从c回调go时）执行合约call前保存
func (con *Controller) StoreVmContext(newTvm *Tvm) {
	if len(con.VmStack) > MAX_DEPTH {
		// TODO 向vm抛异常
		return;
	}

	currentVm := con.Vm
	con.VmStack = append(con.VmStack, currentVm)
	con.Vm = newTvm
}

// 恢复tvm上下文
func (con *Controller) RecoverVmContext() {
	con.Vm = con.VmStack[len(con.VmStack)-1]
	con.VmStack = con.VmStack[:len(con.VmStack)-1]
}
