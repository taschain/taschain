package tvm

var controller *Controller // vm的controller

// MaxDepth max depth of running stack
const MaxDepth int = 8

// StoreVMContext Store VM Context
func (con *Controller) StoreVMContext(newTvm *TVM) bool {
	if len(con.VMStack) >= MaxDepth {
		return false
	}

	currentVM := con.VM
	con.VMStack = append(con.VMStack, currentVM)
	con.VM = newTvm
	return true
}

// RecoverVMContext Recover VM Context
func (con *Controller) RecoverVMContext() {
	con.VM = con.VMStack[len(con.VMStack)-1]
	con.VMStack = con.VMStack[:len(con.VMStack)-1]
}
