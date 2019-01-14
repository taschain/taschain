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

package handler

import (
	"network"
	"github.com/gogo/protobuf/proto"
	"common"
	"core"
	"utility"
	"middleware/types"
	"middleware/pb"
	"middleware/notify"
	"github.com/hashicorp/golang-lru"
	"math/big"
)

const (
	ChainPieceLength  = 10
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

	//notify.BUS.Subscribe(notify.NewBlockHeader, handler.newBlockHeaderHandler)
	//notify.BUS.Subscribe(notify.BlockBody, handler.blockBodyHandler)
	//notify.BUS.Subscribe(notify.BlockBodyReq, handler.blockBodyReqHandler)
	//notify.BUS.Subscribe(notify.StateInfoReq, handler.stateInfoReqHandler)
	//notify.BUS.Subscribe(notify.StateInfo, handler.stateInfoHandler)
	notify.BUS.Subscribe(notify.BlockReq, handler.blockReqHandler)
	notify.BUS.Subscribe(notify.NewBlock, handler.newBlockHandler)

	notify.BUS.Subscribe(notify.TransactionBroadcast, handler.TransactionBroadcastHandler)
	notify.BUS.Subscribe(notify.TransactionReq, handler.transactionReqHandler)
	notify.BUS.Subscribe(notify.TransactionGot, handler.transactionGotHandler)

	//go handler.loop()
	return &handler
}

func (c *ChainHandler) Handle(sourceId string, msg network.Message) error {
	return nil
}

func (ch ChainHandler) TransactionBroadcastHandler(msg notify.Message) {
	mtm, ok := msg.(*notify.TransactionBroadcastMessage)
	if !ok {
		core.Logger.Debugf("TransactionBroadcastHandler Message assert not ok!")
		return
	}
	txs, e := types.UnMarshalTransactions(mtm.TransactionsByte)
	if e != nil {
		core.Logger.Errorf("Discard MINER_TRANSACTION_MSG because of unmarshal error:%s", e.Error())
		return
	}
	core.BlockChainImpl.GetTransactionPool().AddBroadcastTransactions(txs)

}

func (ch ChainHandler) transactionReqHandler(msg notify.Message) {
	trm, ok := msg.(*notify.TransactionReqMessage)
	if !ok {
		core.Logger.Debugf("transactionReqHandler Message assert not ok!")
		return
	}
	m, e := unMarshalTransactionRequestMessage(trm.TransactionReqByte)
	if e != nil {
		core.Logger.Errorf("[handler]Discard TransactionRequestMessage because of unmarshal error:%s", e.Error())
		return
	}

	source := trm.Peer
	core.Logger.Debugf("receive REQ_TRANSACTION_MSG from %s,%d-%D,tx_len", source, m.BlockHeight, m.CurrentBlockHash.ShortS(), len(m.TransactionHashes))
	if nil == core.BlockChainImpl {
		return
	}
	transactions, need, e := core.BlockChainImpl.GetTransactionPool().GetTransactions(m.CurrentBlockHash, m.TransactionHashes)
	if e == core.ErrNil {
		m.TransactionHashes = need
	}

	if nil != transactions && 0 != len(transactions) {
		core.SendTransactions(transactions, source, m.BlockHeight, m.BlockPv)
	}
	return
}

func (ch ChainHandler) transactionGotHandler(msg notify.Message) {
	tgm, ok := msg.(*notify.TransactionGotMessage)
	if !ok {
		core.Logger.Debugf("transactionGotHandler Message assert not ok!")
		return
	}

	txs, e := types.UnMarshalTransactions(tgm.TransactionGotByte)
	if e != nil {
		core.Logger.Errorf("[handler]Discard TRANSACTION_MSG because of unmarshal error:%s", e.Error())
		return
	}
	e = core.BlockChainImpl.GetTransactionPool().AddMissTransactions(txs)
	if e != nil {
		return
	}

	m := notify.TransactionGotAddSuccMessage{Transactions: txs, Peer: tgm.Peer}
	notify.BUS.Publish(notify.TransactionGotAddSucc, &m)
	return
}

func (ch ChainHandler) blockReqHandler(msg notify.Message) {
	if core.BlockChainImpl.IsLightMiner() {
		core.Logger.Debugf("Is Light Miner!")
		return
	}

	m, ok := msg.(*notify.BlockReqMessage)
	if !ok {
		core.Logger.Debugf("blockReqHandler Message assert not ok!")
		return
	}
	reqHeight := utility.ByteToUInt64(m.HeightByte)
	localHeight := core.BlockChainImpl.Height()

	core.Logger.Debugf("blockReqHandler:reqHeight:%d,localHeight:%d", reqHeight, localHeight)
	var count = 0
	for i := reqHeight; i <= localHeight; i++ {
		block := core.BlockChainImpl.QueryBlock(i)
		if block == nil {
			continue
		}
		count++
		if count == blockResponseSzie || i == localHeight {
			core.SendBlock(m.Peer, block, true)
		} else {
			core.SendBlock(m.Peer, block, false)
		}
		if count >= blockResponseSzie {
			break
		}
	}
	if count == 0 {
		core.SendBlock(m.Peer, nil, true)
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
		core.Logger.Debugf("Discard BlockMsg because UnMarshalBlock error:%d", e.Error())
		return
	}

	core.Logger.Debugf("Rcv new block from %s,hash:%v,height:%d,totalQn:%d,tx len:%d", source, block.Header.Hash.Hex(), block.Header.Height, block.Header.TotalQN, len(block.Transactions))
	//此处source传空，此处上链引发如果触发分叉不处理
	core.BlockChainImpl.AddBlockOnChain("", block)
}

//----------------------------------------------------------------------------------------------------------------------

func onMessageTransaction(txs []*types.Transaction) error {
	e := core.BlockChainImpl.GetTransactionPool().AddMissTransactions(txs)
	if e != nil {
		return e
	}
	return nil
}

func unMarshalTransactionRequestMessage(b []byte) (*core.TransactionRequestMessage, error) {
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
	message := core.TransactionRequestMessage{TransactionHashes: txHashes, CurrentBlockHash: currentBlockHash, BlockHeight: *m.BlockHeight, BlockPv: blockPv}
	return &message, nil
}

//func (ch ChainHandler) onRequestTraceMsg(targetId string, data []byte) {
//	message := network.Message{Code: network.ResponseTraceMsg, Body: data}
//	network.GetNetInstance().Send(targetId, message)
//}

//func (ch ChainHandler) newBlockHeaderHandler(msg notify.Message) {
//	//core.Logger.Debugf("[ChainHandler]newBlockHeaderHandler RCV MSG")
//	m, ok := msg.GetData().(*notify.BlockHeaderNotifyMessage)
//	if !ok {
//		core.Logger.Debugf("[ChainHandler]newBlockHeaderHandler GetData assert not ok!")
//		return
//	}
//
//	header, e := types.UnMarshalBlockHeader(m.HeaderByte)
//	if e != nil {
//		core.Logger.Errorf("[handler]Discard NewBlockHeader because of unmarshal error:%s", e.Error())
//		return
//	}
//	b := core.BlockChainImpl.QueryBlockByHash(header.Hash)
//	if b != nil {
//		return
//	}
//
//	notify := blockHeaderNotify{header: *header, peer: m.Peer}
//	ch.headerCh <- notify
//}
//
//func (ch ChainHandler) blockBodyReqHandler(msg notify.Message) {
//	//core.Logger.Debugf("[ChainHandler]blockBodyReqHandler RCV MSG")
//	m, ok := msg.GetData().(*notify.BlockBodyReqMessage)
//	if !ok {
//		core.Logger.Debugf("[ChainHandler]blockBodyReqHandler GetData assert not ok!")
//		return
//	}
//
//	hash := common.BytesToHash(m.BlockHashByte)
//
//	body := core.BlockChainImpl.QueryBlockBody(hash)
//
//	core.SendBlockBody(m.Peer, hash, body)
//}
//
//func (ch ChainHandler) blockBodyHandler(msg notify.Message) {
//	//core.Logger.Debugf("[ChainHandler]blockBodyHandler RCV MSG")
//	m, ok := msg.GetData().(*notify.BlockBodyNotifyMessage)
//	if !ok {
//		return
//	}
//
//	blockHash, txs, e := unMarshalBlockBody(m.BodyByte)
//	if e != nil {
//		core.Logger.Errorf("[handler]Discard BlockBodyMessage because of unmarshal error:%s", e.Error())
//		return
//	}
//
//	notify := blockBodyNotify{blockHash: blockHash, body: txs, peer: m.Peer}
//	ch.bodyCh <- notify
//}
//
////只有重节点
//func (ch ChainHandler) stateInfoReqHandler(msg notify.Message) {
//	if core.BlockChainImpl.IsLightMiner() {
//		return
//	}
//
//	m, ok := msg.GetData().(*notify.StateInfoReqMessage)
//	if !ok {
//		return
//	}
//	message, e := unMarshalStateInfoReq(m.StateInfoReqByte)
//	if e != nil {
//		core.Logger.Errorf("[handler]Discard unMarshalStateInfoReq because of unmarshal error:%s", e.Error())
//		return
//	}
//
//	header := core.BlockChainImpl.QueryBlockByHeight(message.Height)
//	if header == nil {
//		//正在出这一块，链上没有
//		core.Logger.Debugf("stateInfoReqHandler,local chain has no block! Get from block cache !height:%d,blockhash:%x", message.Height, message.BlockHash)
//		b := core.BlockChainImpl.(*core.FullBlockChain).GetCastingBlock(message.BlockHash)
//		if b == nil {
//			core.Logger.Errorf("stateInfoReqHandler,local has no block height:%d", message.Height)
//			panic("Header is nil! ")
//		}
//		header = b.Header
//		core.Logger.Debugf("stateInfoReqHandler, Get block from block cache !height:%d,blockhash:%x", header.Height, header.Hash)
//	}
//	core.Logger.Errorf("stateInfoReqHandler,block height:%d", message.Height)
//	preHeader := core.BlockChainImpl.QueryBlockByHash(header.PreHash)
//	stateNodes := core.BlockChainImpl.GetTrieNodesByExecuteTransactions(preHeader.Header, message.Transactions, message.Addresses)
//	core.SendStateInfo(m.Peer, message.Height, stateNodes, header.Hash, preHeader.Header.StateTree)
//}
//
////只有轻节点
//func (ch ChainHandler) stateInfoHandler(msg notify.Message) {
//	m, ok := msg.GetData().(*notify.StateInfoMessage)
//	if !ok {
//		return
//	}
//	message, e := unMarshalStateInfo(m.StateInfoByte)
//	if e != nil {
//		core.Logger.Errorf("[handler]Discard unMarshalStateInfo because of unmarshal error:%s", e.Error())
//		return
//	}
//	core.BlockChainImpl.InsertStateNode(message.TrieNodes)
//
//	core.BlockChainImpl.(*core.LightChain).SetPreBlockStateRoot(message.BlockHash, message.PreBlockSateRoot)
//	//todo 此处插入后默认不再缺少账户 但是缺少验证 有安全风险
//	b := core.BlockChainImpl.(*core.LightChain).MarkFullAccountBlock(message.BlockHash)
//	if b == nil {
//		return
//	}
//	core.Logger.Debugf("After InsertStateNode,get cached node to add on chain! height:%d", b.Header.Height)
//	result := core.BlockChainImpl.AddBlockOnChain(b)
//	if result == 0 {
//		core.RequestBlock(m.Peer, core.BlockChainImpl.Height()+1)
//	}
//}

//func (ch ChainHandler) loop() {
//	for {
//		select {
//		case headerNotify := <-ch.headerCh:
//			//core.Logger.Debugf("[ChainHandler]headerCh receive,hash:%v,peer:%s,tx len:%d,block:%d-%d", headerNotify.header.Hash.Hex(), headerNotify.peer, len(headerNotify.header.Transactions), headerNotify.header.Height, headerNotify.header.TotalQN)
//			hash := headerNotify.header.Hash
//			if _, ok := ch.headerPending[hash]; ok || ch.complete.Contains(hash) {
//				//core.Logger.Debugf("[ChainHandler]header hit pending or complete")
//				break
//			}
//
//			local := core.BlockChainImpl.QueryBlockByHash(hash)
//			if local != nil {
//				ch.complete.Add(hash, headerNotify.header)
//				break
//			}
//
//			if len(headerNotify.header.Transactions) == 0 {
//				block := types.Block{Header: &headerNotify.header, Transactions: nil}
//				msg := notify.BlockMessage{Block: block}
//				ch.complete.Add(block.Header.Hash, block)
//				delete(ch.headerPending, block.Header.Hash)
//				notify.BUS.Publish(notify.NewBlock, &msg)
//				break
//			}
//			ch.headerPending[hash] = headerNotify
//			core.ReqBlockBody(headerNotify.peer, hash)
//		case bodyNotify := <-ch.bodyCh:
//			core.Logger.Debugf("[ChainHandler]bodyCh receive,hash:%v,peer:%s,body len:%d", bodyNotify.blockHash.Hex(), bodyNotify.peer, len(bodyNotify.body))
//			headerNotify, ok := ch.headerPending[bodyNotify.blockHash]
//			if !ok {
//				break
//			}
//
//			if headerNotify.peer != bodyNotify.peer {
//				break
//			}
//			block := types.Block{Header: &headerNotify.header, Transactions: bodyNotify.body}
//			msg := notify.BlockMessage{Block: block}
//			ch.complete.Add(block.Header.Hash, block)
//			delete(ch.headerPending, block.Header.Hash)
//			notify.BUS.Publish(notify.NewBlock, &msg)
//		}
//	}
//}

//func unMarshalBlockBody(b []byte) (hash common.Hash, transactions []*types.Transaction, err error) {
//	message := new(tas_middleware_pb.BlockBody)
//	e := proto.Unmarshal(b, message)
//	if e != nil {
//		core.Logger.Errorf("[handler]unMarshalBlockBody error:%s", e.Error())
//		return hash, nil, e
//	}
//	hash = common.BytesToHash(message.BlockHash)
//	transactions = types.PbToTransactions(message.Transactions)
//	return hash, transactions, nil
//}
//
//func unMarshalStateInfoReq(b []byte) (core.StateInfoReq, error) {
//	message := new(tas_middleware_pb.StateInfoReq)
//	e := proto.Unmarshal(b, message)
//	if e != nil {
//		core.Logger.Errorf("[handler]unMarshalStateInfoReq error:%s", e.Error())
//		return core.StateInfoReq{}, e
//	}
//
//	var transactions []*types.Transaction
//	if message.Transactions != nil {
//		transactions = types.PbToTransactions(message.Transactions.Transactions)
//	}
//
//	var addresses []common.Address
//	for _, addr := range message.Addresses {
//		addresses = append(addresses, common.BytesToAddress(addr))
//	}
//	stateInfoReq := core.StateInfoReq{Height: *message.Height, Transactions: transactions, Addresses: addresses, BlockHash: common.BytesToHash(message.BlockHash)}
//	return stateInfoReq, nil
//}
//
//func unMarshalStateInfo(b []byte) (core.StateInfo, error) {
//	message := new(tas_middleware_pb.StateInfo)
//	e := proto.Unmarshal(b, message)
//	if e != nil {
//		core.Logger.Errorf("[handler]unMarshalStateInfo error:%s", e.Error())
//		return core.StateInfo{}, e
//	}
//
//	trieNodes := make([]types.StateNode, 0)
//	for _, node := range message.TrieNodes {
//		n := types.StateNode{Key: node.Key, Value: node.Data}
//		trieNodes = append(trieNodes, n)
//	}
//
//	stateInfo := core.StateInfo{Height: *message.Height, TrieNodes: &trieNodes,
//		BlockHash: common.BytesToHash(message.BlockHash), PreBlockSateRoot: common.BytesToHash(message.ProBlockStateRoot)}
//	return stateInfo, nil
//}

//func unMarshalBlocks(b []byte) ([]*core.Block, error) {
//	blockSlice := new(tas_pb.BlockSlice)
//	error := proto.Unmarshal(b, blockSlice)
//	if error != nil {
//		logger.Errorf("Unmarshal Blocks error:%s\n", error.Error())
//		return nil, error
//	}
//	blocks := blockSlice.Blocks
//	result := make([]*core.Block, 0)
//
//	for _, b := range blocks {
//		block := pbToBlock(b)
//		result = append(result, block)
//	}
//	return result, nil
//}

//func unMarshalGroups(b []byte) ([]*types.Group, error) {
//	message := new(tas_middlewstateInfoReqHandlerare_pb.GroupSlice)
//	e := proto.Unmarshal(b, message)
//	if e != nil {
//		core.Logger.Errorf("[handler]Unmarshal Groups error:%s", e.Error())
//		return nil, e
//	}
//
//	groups := make([]*types.Group, 0)
//	if message.Groups != nil {
//		for _, g := range message.Groups {
//			groups = append(groups, types.PbToGroup(g))
//		}
//	}
//	return groups, nil
//}
