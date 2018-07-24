package core

import (
	"testing"
	"common"
	"fmt"
	"vm/core/state"
	vc "vm/common"
)

func TestQueryBlock(t *testing.T) {
	common.InitConf("/Users/Kaede/TasProject/work/test2/d1")
	initBlockChain()
	fmt.Println(BlockChainImpl.latestBlock.Height)
	lastStateHash := BlockChainImpl.latestBlock.StateTree.Hex()
	preHeader := BlockChainImpl.queryBlockHeaderByHeight(BlockChainImpl.latestBlock.Height - 1, false)
	for {
		if preHeader.Height == 0 {
			break
		}
		if lastStateHash != preHeader.StateTree.Hex() {
			break
		}
		preHeader = BlockChainImpl.queryBlockHeaderByHeight(preHeader.Height - 1, false)
	}


	state, _ := state.New(vc.BytesToHash(preHeader.StateTree.Bytes()), BlockChainImpl.stateCache)
	fmt.Println(string(BlockChainImpl.LatestStateDB().Dump()))
	fmt.Println(string(state.Dump()))
}
