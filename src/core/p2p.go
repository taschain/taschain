package core

import (
	"common"
	"fmt"
)

var (
	BlockChainConnectorImpl *BlockChainConnector
	GroupChainConnectorImpl *GroupChainConnector
	isDebug                 bool
)

type BlockChainConnector struct {
	chain *BlockChain
}

type GroupChainConnector struct {
	chain *GroupChain
}

func InitCore() error {
	// 默认是debug模式
	isDebug = common.GlobalConf.GetBool(CONFIG_SEC, "debug", true)
	if isDebug {
		Clear()
	}

	if nil == BlockChainImpl {
		err := initBlockChain()
		if nil != err {
			return err
		}
	}
	BlockChainConnectorImpl = &BlockChainConnector{
		chain: BlockChainImpl,
	}

	if nil == GroupChainImpl {
		err := initGroupChain()
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
func (connector *BlockChainConnector) QueryTransaction(hs []common.Hash) ([]*Transaction, error) {
	if nil == hs || 0 == len(hs) {
		return nil, nil
	}

	var err error
	txs := make([]*Transaction, len(hs))
	for i, hash := range hs {
		txs[i], err = connector.chain.GetTransactionByHash(hash)
	}

	return txs, err
}

//transactionArrivedNotifyBlockChainFn 实现
func (connector *BlockChainConnector) TransactionArrived(ts []*Transaction) error {
	if nil == ts || 0 == len(ts) {
		return fmt.Errorf("nil transactions")
	}

	var returnErr error

	for _, tx := range ts {
		_, err := connector.chain.GetTransactionPool().Add(tx)
		if err != nil {
			returnErr = err
		}
	}

	return returnErr
}

//addNewBlockToChainFn 实现
func (connector *BlockChainConnector) AddNewBlock(b *Block, sig []byte) {
	if nil == b {
		return
	}
	connector.chain.AddBlockOnChain(b)
}

//addTransactionToPoolFn 实现
func (connector *BlockChainConnector) AddTransactionToPool(ts []*Transaction) {
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
