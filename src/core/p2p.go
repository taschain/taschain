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

package core

import (
	"common"
	"fmt"
	"middleware/types"
)

var (
	BlockChainConnectorImpl *BlockChainConnector
	GroupChainConnectorImpl *GroupChainConnector
	isDebug                 bool
)

type BlockChainConnector struct {
	chain BlockChain
}

type GroupChainConnector struct {
	chain *GroupChain
}

func InitCore(light bool, helper types.ConsensusHelper) error {
	light = false
	// 默认是debug模式
	isDebug = common.GlobalConf.GetBool(CONFIG_SEC, "debug", true)
	if isDebug {
		//Clear()
	}

	if nil == BlockChainImpl {
		var err error
		if light {
			err = initLightChain(helper)
		} else {
			err = initBlockChain(helper)
		}
		if nil != err {
			return err
		}
	}
	BlockChainConnectorImpl = &BlockChainConnector{
		chain: BlockChainImpl,
	}

	if nil == GroupChainImpl {
		err := initGroupChain(helper.GenerateGenesisInfo(), helper)
		if nil != err {
			return err
		}
	}
	GroupChainConnectorImpl = &GroupChainConnector{
		chain: GroupChainImpl,
	}
	return nil
}

//queryTracsactionFn 实现
func (connector *BlockChainConnector) QueryTransaction(hs []common.Hash) ([]*types.Transaction, error) {
	if nil == hs || 0 == len(hs) {
		return nil, nil
	}

	var err error
	txs := make([]*types.Transaction, len(hs))
	for i, hash := range hs {
		txs[i], err = connector.chain.GetTransactionByHash(hash)
	}

	return txs, err
}

//transactionArrivedNotifyBlockChainFn 实现
func (connector *BlockChainConnector) TransactionArrived(ts []*types.Transaction) error {
	if nil == ts || 0 == len(ts) {
		return fmt.Errorf("nil transactions")
	}

	return connector.chain.GetTransactionPool().AddTransactions(ts)
}

//addNewBlockToChainFn 实现
func (connector *BlockChainConnector) AddNewBlock(b *types.Block, sig []byte) {
	if nil == b {
		return
	}
	connector.chain.AddBlockOnChain(b)
}

//addTransactionToPoolFn 实现
func (connector *BlockChainConnector) AddTransactionToPool(ts []*types.Transaction) {
	connector.TransactionArrived(ts)
}

//getBlockChainHeightFn 实现
func (connector *BlockChainConnector) getBlockChainHeight() (uint64, error) {
	if nil == connector.chain {
		return 0, fmt.Errorf("nil blockchain")
	}
	return connector.chain.Height(), nil
}

//getGroupChainHeightFn 实现
func (connector *GroupChainConnector) getGroupChainHeight() (uint64, error) {
	if nil == connector.chain {
		return 0, fmt.Errorf("nil blockchain")
	}
	return connector.chain.count, nil
}
