package core

import (
	"vm/core/state"
	"vm/core/vm"
	"common"
	vmc "vm/common"
)

/*
**  Creator: pxf
**  Date: 2018/4/17 下午3:58
**  Description: 
*/

func NewStateDB(hash common.Hash, chain *BlockChain) vm.StateDB {
	state, err := state.New(vmc.BytesToHash(hash.Bytes()), chain.stateCache)
	if err == nil {
		return state
	}
	return nil
}
