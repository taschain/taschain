package core

import (
	"middleware/types"
	"common"
	"storage/core"
	"tvm"
)

type TVMExecutor struct {
	bc     *BlockChain
}

func NewTVMExecutor(bc *BlockChain) *TVMExecutor {
	return &TVMExecutor{
		bc:     bc,
	}
}

func (executor *TVMExecutor) Execute(statedb *core.AccountDB, block *types.Block, processor VoteProcessor) ([]*types.Transaction, []common.Hash, *common.Hash, uint64) {
	vm := tvm.NewTvm(statedb)
	for _,transaction := range block.Transactions{
		if(len(transaction.Data) > 0) {
			script := string(transaction.Data)
			vm.Execute(script)
		}
	}

	return nil,nil,nil,0
}
