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
	"github.com/gogo/protobuf/proto"
	"common"
	"utility"
	"middleware/types"
	"middleware/pb"
	"middleware/notify"
	"github.com/hashicorp/golang-lru"
	"math/big"
)

const (
	blockResponseSzie = 1
)

type ChainHandler struct {
	headerPending map[common.Hash]blockHeaderNotify

	complete *lru.Cache

	headerCh chan blockHeaderNotify

	bodyCh chan blockBodyNotify
}

type blockHeaderNotify struct {
	header types.BlockHeader

	peer string
}

type blockBodyNotify struct {
	body []*types.Transaction

	blockHash common.Hash

	peer string
}

func NewChainHandler() network.MsgHandler {
	headerPending := make(map[common.Hash]blockHeaderNotify)
	complete, _ := lru.New(256)
	headerCh := make(chan blockHeaderNotify, 100)
	bodyCh := make(chan blockBodyNotify, 100)
	handler := ChainHandler{headerPending: headerPending, complete: complete, headerCh: headerCh, bodyCh: bodyCh,}

	notify.BUS.Subscribe(notify.BlockReq, handler.blockReqHandler)
	notify.BUS.Subscribe(notify.NewBlock, handler.newBlockHandler)

	notify.BUS.Subscribe(notify.TransactionBroadcast, handler.TransactionBroadcastHandler)
	notify.BUS.Subscribe(notify.TransactionReq, handler.transactionReqHandler)
	notify.BUS.Subscribe(notify.TransactionGot, handler.transactionGotHandler)

	return &handler
}

func (c *ChainHandler) Handle(sourceId string, msg network.Message) error {
	return nil
}

func (ch ChainHandler) TransactionBroadcastHandler(msg notify.Message) {
	mtm, ok := msg.(*notify.TransactionBroadcastMessage)
	if !ok {
		Logger.Debugf("TransactionBroadcastHandler Message assert not ok!")
		return
	}
	txs, e := types.UnMarshalTransactions(mtm.TransactionsByte)
	if e != nil {
		Logger.Errorf("Discard MINER_TRANSACTION_MSG because of unmarshal error:%s", e.Error())
		return
	}
	BlockChainImpl.GetTransactionPool().AddBroadcastTransactions(txs)

}

func (ch ChainHandler) transactionReqHandler(msg notify.Message) {
	trm, ok := msg.(*notify.TransactionReqMessage)
	if !ok {
		Logger.Debugf("transactionReqHandler Message assert not ok!")
		return
	}
	m, e := unMarshalTransactionRequestMessage(trm.TransactionReqByte)
	if e != nil {
		Logger.Errorf("[handler]Discard TransactionRequestMessage because of unmarshal error:%s", e.Error())
		return
	}

	source := trm.Peer
	Logger.Debugf("receive REQ_TRANSACTION_MSG from %s,%d-%D,tx_len", source, m.BlockHeight, m.CurrentBlockHash.ShortS(), len(m.TransactionHashes))
	if nil == BlockChainImpl {
		return
	}
	transactions, need, e := BlockChainImpl.GetTransactionPool().GetTransactions(m.CurrentBlockHash, m.TransactionHashes)
	if e == ErrNil {
		m.TransactionHashes = need
	}

	if nil != transactions && 0 != len(transactions) {
		SendTransactions(transactions, source, m.BlockHeight, m.BlockPv)
	}
	return
}

func (ch ChainHandler) transactionGotHandler(msg notify.Message) {
	tgm, ok := msg.(*notify.TransactionGotMessage)
	if !ok {
		Logger.Debugf("transactionGotHandler Message assert not ok!")
		return
	}

	txs, e := types.UnMarshalTransactions(tgm.TransactionGotByte)
	if e != nil {
		Logger.Errorf("[handler]Discard TRANSACTION_MSG because of unmarshal error:%s", e.Error())
		return
	}
	BlockChainImpl.GetTransactionPool().AddMissTransactions(txs)

	m := notify.TransactionGotAddSuccMessage{Transactions: txs, Peer: tgm.Peer}
	notify.BUS.Publish(notify.TransactionGotAddSucc, &m)
	return
}

func (ch ChainHandler) blockReqHandler(msg notify.Message) {
	if BlockChainImpl.IsLightMiner() {
		Logger.Debugf("Is Light Miner!")
		return
	}

	m, ok := msg.(*notify.BlockReqMessage)
	if !ok {
		Logger.Debugf("blockReqHandler Message assert not ok!")
		return
	}
	reqHeight := utility.ByteToUInt64(m.HeightByte)
	localHeight := BlockChainImpl.Height()

	Logger.Debugf("blockReqHandler:reqHeight:%d,localHeight:%d", reqHeight, localHeight)
	var count = 0
	for i := reqHeight; i <= localHeight; i++ {
		block := BlockChainImpl.QueryBlock(i)
		if block == nil {
			continue
		}
		count++
		if count == blockResponseSzie || i == localHeight {
			SendBlock(m.Peer, block, true)
		} else {
			SendBlock(m.Peer, block, false)
		}
		if count >= blockResponseSzie {
			break
		}
	}
	if count == 0 {
		SendBlock(m.Peer, nil, true)
	}
}

func (ch ChainHandler) newBlockHandler(msg notify.Message) {
	m, ok := msg.(*notify.NewBlockMessage)
	if !ok {
		return
	}
	source := m.Peer
	block, e := types.UnMarshalBlock(m.BlockByte)
	if e != nil {
		Logger.Debugf("Discard BlockMsg because UnMarshalBlock error:%d", e.Error())
		return
	}

	Logger.Debugf("Rcv new block from %s,hash:%v,height:%d,totalQn:%d,tx len:%d", source, block.Header.Hash.Hex(), block.Header.Height, block.Header.TotalQN, len(block.Transactions))
	BlockChainImpl.AddBlockOnChain(source, block, types.NewBlock)
}

//----------------------------------------------------------------------------------------------------------------------
func unMarshalTransactionRequestMessage(b []byte) (*TransactionRequestMessage, error) {
	m := new(tas_middleware_pb.TransactionRequestMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("[handler]UnMarshal TransactionRequestMessage error:%s", e.Error())
		return nil, e
	}

	txHashes := make([]common.Hash, 0)
	for _, txHash := range m.TransactionHashes {
		txHashes = append(txHashes, common.BytesToHash(txHash))
	}

	currentBlockHash := common.BytesToHash(m.CurrentBlockHash)
	blockPv := &big.Int{}
	blockPv.SetBytes(m.BlockPv)
	message := TransactionRequestMessage{TransactionHashes: txHashes, CurrentBlockHash: currentBlockHash, BlockHeight: *m.BlockHeight, BlockPv: blockPv}
	return &message, nil
}
