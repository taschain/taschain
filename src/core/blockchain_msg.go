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
	"network"
	"common"
	"middleware/types"
	"middleware/pb"
	"middleware/notify"

	"github.com/gogo/protobuf/proto"
	"consensus/groupsig"
)


func (chain *FullBlockChain) initMessageHandler() {
	notify.BUS.Subscribe(notify.BlockAddSucc, chain.onBlockAddSuccess)
	notify.BUS.Subscribe(notify.NewBlock, chain.newBlockHandler)
	//notify.BUS.Subscribe(notify.TransactionBroadcast, chain.transactionBroadcastHandler)
	notify.BUS.Subscribe(notify.TransactionReq, chain.transactionReqHandler)
	notify.BUS.Subscribe(notify.TransactionGot, chain.transactionGotHandler)
	notify.BUS.Subscribe(notify.TxPoolAddTxs, chain.notifyTransactionGot)
}

func (chain *FullBlockChain) notifyTransactionGot(msg notify.Message)  {
	m := msg.(*TxPoolAddMessage)

	availTxs := m.txs
	txsrc := m.txSrc
	if txsrc != txRequest {
		wantedTxs := make([]*types.Transaction, 0)
		for _, tx := range m.txs {
			if chain.txsWanted.Has(tx.Hash) {
				wantedTxs = append(wantedTxs, tx)
			}
		}
		availTxs = wantedTxs
	}
	if len(availTxs) == 0 {
		return
	}
	nm := notify.TransactionGotAddSuccMessage{Transactions: availTxs, Peer: ""}
	notify.BUS.Publish(notify.TransactionGotAddSucc, &nm)
	Logger.Debugf("Got transactions by %v , size %v", txsrc, len(availTxs))
	txHashs := make([]interface{}, len(availTxs))
	for i, tx := range availTxs {
		txHashs[i] = tx.Hash
	}
	chain.txsWanted.Remove(txHashs...)
}

func (chain *FullBlockChain) addWantedTransactions(txHashs []common.Hash)  {
	txs := make([]interface{}, len(txHashs))
	for i, txh := range txs {
		txs[i] = txh
	}
	removeCnt := chain.txsWanted.Size()+len(txs) - wantedTxsSize
	if removeCnt > 0 {
		removeTxs := make([]interface{}, 0)
		chain.txsWanted.Each(func(item interface{}) bool {
			removeTxs = append(removeTxs, item)
			return len(removeTxs) < removeCnt
		})
		chain.txsWanted.Remove(removeTxs...)
	}
	chain.txsWanted.Add(txs...)
}
//
//func (chain *FullBlockChain) transactionBroadcastHandler(msg notify.Message) {
//	mtm, ok := msg.(*notify.TransactionBroadcastMessage)
//	if !ok {
//		Logger.Debugf("transactionBroadcastHandler:Message assert not ok!")
//		return
//	}
//	txs, e := types.UnMarshalTransactions(mtm.TransactionsByte)
//	if e != nil {
//		Logger.Errorf("Unmarshal transactions error:%s", e.Error())
//		return
//	}
//	chain.transactionPool.AddTransactions(txs, 1)
//
//	chain.txsWanted.Has()
//	wantedTxs := make([]*types.Transaction, 0)
//	for _, tx := range txs {
//		if chain.txsWanted.Has(tx.Hash) {
//			wantedTxs = append(wantedTxs, tx)
//		}
//	}
//	if len(wantedTxs) > 0 {
//		chain.notifyTransactionGot(wantedTxs, mtm.Peer, "broadcast")
//	}
//}

func (chain *FullBlockChain) transactionReqHandler(msg notify.Message) {
	trm, ok := msg.(*notify.TransactionReqMessage)
	if !ok {
		Logger.Debugf("transactionReqHandler:Message assert not ok!")
		return
	}
	m, e := unMarshalTransactionRequestMessage(trm.TransactionReqByte)
	if e != nil {
		Logger.Errorf("unmarshal transaction request message error:%s", e.Error())
		return
	}

	source := trm.Peer
	//Logger.Debugf("receive transaction req from %s,hash %v,tx_len %v", source, m.CurrentBlockHash.String(), len(m.TransactionHashes))

	transactions, _ := chain.GetBlockTransactions(m.CurrentBlockHash, m.TransactionHashes, false)

	if nil != transactions && 0 != len(transactions) {
		chain.sendTransactions(transactions, source)
	}
	return
}

func (chain *FullBlockChain) transactionGotHandler(msg notify.Message) {
	tgm, ok := msg.(*notify.TransactionGotMessage)
	if !ok {
		Logger.Debugf("transactionGotHandler:Message assert not ok!")
		return
	}

	txs, e := types.UnMarshalTransactions(tgm.TransactionGotByte)
	if e != nil {
		Logger.Errorf("Unmarshal got transactions error:%s", e.Error())
		return
	}

	chain.transactionPool.AddTransactions(txs, txRequest)

	return
}



func (chain *FullBlockChain) newBlockHandler(msg notify.Message) {
	m, ok := msg.(*notify.NewBlockMessage)
	if !ok {
		return
	}
	source := m.Peer
	block, e := types.UnMarshalBlock(m.BlockByte)
	if e != nil {
		Logger.Debugf("UnMarshal block error:%d", e.Error())
		return
	}

	Logger.Debugf("Rcv new block from %s,hash:%v,height:%d,totalQn:%d,tx len:%d", source, block.Header.Hash.Hex(), block.Header.Height, block.Header.TotalQN, len(block.Transactions))
	chain.AddBlockOnChain(source, block)
}


func (chain *FullBlockChain) requestTransaction(bh *types.BlockHeader, missing []common.Hash) {
	var castorId groupsig.ID
	error := castorId.Deserialize(bh.Castor)
	if error != nil {
		panic("Groupsig id deserialize error:" + error.Error())
	}

	m := &transactionRequestMessage{TransactionHashes: missing, CurrentBlockHash: bh.Hash}

	body, e := marshalTransactionRequestMessage(m)
	if e != nil {
		Logger.Errorf("Discard MarshalTransactionRequestMessage because of marshal error:%s!", e.Error())
		return
	}
	Logger.Debugf("send REQ_TRANSACTION_MSG to %s,reqLen:%v, bhLen:%v, hash:%s", castorId, len(missing), len(bh.Transactions),m.CurrentBlockHash.String())
	message := network.Message{Code: network.ReqTransactionMsg, Body: body}
	network.GetNetInstance().Send(castorId.String(), message)

	chain.addWantedTransactions(m.TransactionHashes)
}

func (chain *FullBlockChain) sendTransactions(txs []*types.Transaction, sourceId string) {
	body, e := types.MarshalTransactions(txs)
	if e != nil {
		Logger.Errorf("Discard MarshalTransactions because of marshal error:%s!", e.Error())
		return
	}
	//Logger.Debugf("send transactions to %v size %v", len(txs), sourceId)
	message := network.Message{Code: network.TransactionGotMsg, Body: body}
	go network.GetNetInstance().Send(sourceId, message)
}

func unMarshalTransactionRequestMessage(b []byte) (*transactionRequestMessage, error) {
	m := new(tas_middleware_pb.TransactionRequestMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("UnMarshal transaction request message error:%s", e.Error())
		return nil, e
	}

	txHashes := make([]common.Hash, 0)
	for _, txHash := range m.TransactionHashes {
		txHashes = append(txHashes, common.BytesToHash(txHash))
	}

	currentBlockHash := common.BytesToHash(m.CurrentBlockHash)

	message := transactionRequestMessage{TransactionHashes: txHashes, CurrentBlockHash: currentBlockHash}
	return &message, nil
}


func marshalTransactionRequestMessage(m *transactionRequestMessage) ([]byte, error) {
	txHashes := make([][]byte, 0)
	for _, txHash := range m.TransactionHashes {
		txHashes = append(txHashes, txHash.Bytes())
	}

	currentBlockHash := m.CurrentBlockHash.Bytes()
	message := tas_middleware_pb.TransactionRequestMessage{TransactionHashes: txHashes, CurrentBlockHash: currentBlockHash}
	return proto.Marshal(&message)
}
