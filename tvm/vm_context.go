package tvm

/*
 * 这个用来保存vm：当前设计只考虑单线程，因为区块链交易执行是单线程的
 */

var controller *Controller // vm的controller

const MaxDepth int = 8 //vm执行的最大深度为8

// 合约调合约场景中（从c回调go时）执行合约call前保存
func (con *Controller) StoreVMContext(newTvm *TVM) bool {
	if len(con.VMStack) >= MaxDepth {
		print("===== too many call  levels ====")
		return false
	}

	currentVM := con.VM
	con.VMStack = append(con.VMStack, currentVM)
	con.VM = newTvm
	return true
}

// 恢复tvm上下文
func (con *Controller) RecoverVMContext() {
	con.VM = con.VMStack[len(con.VMStack)-1]
	con.VMStack = con.VMStack[:len(con.VMStack)-1]
}
