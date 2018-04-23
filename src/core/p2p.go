package core

type ChainConnector struct {
	chain *BlockChainI
}

func NewChainConnector(chain *BlockChainI) *ChainConnector {
	return &ChainConnector{
		chain: chain,
	}
}

////queryTracsactionFn 实现
//func (connector *ChainConnector) QueryTransaction(hs []common.Hash, sd logical.SignData) ([]*Transaction, error) {
//	if nil == hs || 0 == len(hs) {
//		return nil, nil
//	}
//
//	var err error
//	txs := make([]*Transaction, len(hs))
//	for i, hash := range hs {
//		txs[i], err = connector.chain.GetTransactionByHash(hash)
//	}
//
//	return txs, err
//}
//
////transactionArrivedNotifyBlockChainFn 实现
//func (connector *ChainConnector) TransactionArrived(ts []Transaction, sd logical.SignData) {
//	if nil == ts || 0 == len(ts) {
//		return
//	}
//
//	for _, tx := range ts {
//		connector.chain.GetTransactionPool().Add(&tx)
//	}
//
//}
//
////addNewBlockToChainFn 实现
//func (connector *ChainConnector) AddNewBlock(b Block, sd logical.SignData){
//	connector.chain.AddBlockOnChain(&b)
//}
//
////addTransactionToPoolFn 实现
//func (connector *ChainConnector) AddTransactionToPool(t Transaction){
//	connector.chain.GetTransactionPool().Add(&t)
//}