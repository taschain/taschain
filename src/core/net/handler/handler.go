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
	"fmt"
	"middleware/types"
	"middleware/pb"
	"middleware/notify"
	"github.com/hashicorp/golang-lru"
	"math/big"
	"time"
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

	notify.BUS.Subscribe(notify.NewBlockHeader, handler.newBlockHeaderHandler)
	notify.BUS.Subscribe(notify.BlockBody, handler.blockBodyHandler)
	notify.BUS.Subscribe(notify.BlockBodyReq, handler.blockBodyReqHandler)
	notify.BUS.Subscribe(notify.StateInfoReq, handler.stateInfoReqHandler)
	notify.BUS.Subscribe(notify.StateInfo, handler.stateInfoHandler)

	go handler.loop()
	return &handler
}

func (c *ChainHandler) Handle(sourceId string, msg network.Message) error {
	switch msg.Code {
	case network.ReqTransactionMsg:
		m, e := unMarshalTransactionRequestMessage(msg.Body)
		if e != nil {
			core.Logger.Errorf("[handler]Discard TransactionRequestMessage because of unmarshal error:%s", e.Error())
			return nil
		}
		OnTransactionRequest(m, sourceId)
	case network.TransactionGotMsg, network.TransactionMsg:
		m, e := types.UnMarshalTransactions(msg.Body)
		if e != nil {
			core.Logger.Errorf("[handler]Discard TRANSACTION_MSG because of unmarshal error:%s", e.Error())
			return nil
		}
		if msg.Code == network.TransactionGotMsg {
			network.Logger.Debugf("receive TRANSACTION_GOT_MSG from %s,tx_len:%d,time at:%v", sourceId, len(m), time.Now())
		}
		err := onMessageTransaction(m)
		return err
	case network.NewBlockMsg:
		block, e := types.UnMarshalBlock(msg.Body)
		if e != nil {
			core.Logger.Errorf("[handler]Discard NEW_BLOCK_MSG because of unmarshal error:%s", e.Error())
			return nil
		}
		onMessageNewBlock(block)

	case network.ReqGroupChainCountMsg:
		core.GroupSyncer.ReqHeightCh <- sourceId
	case network.GroupChainCountMsg:
		height := utility.ByteToUInt64(msg.Body)
		ghi := core.GroupHeightInfo{Height: height, SourceId: sourceId}
		core.GroupSyncer.HeightCh <- ghi
	case network.ReqGroupMsg:
		//baseHeight := utility.ByteToUInt64(msg.Body)
		gri := core.GroupRequestInfo{GroupId: msg.Body, SourceId: sourceId}
		core.GroupSyncer.ReqGroupCh <- gri
	case network.GroupMsg:
		m, e := unMarshalGroupInfo(msg.Body)
		if e != nil {
			core.Logger.Errorf("[handler]Discard GROUP_MSG because of unmarshal error:%s", e.Error())
			return e
		}
		m.SourceId = sourceId
		core.GroupSyncer.GroupCh <- *m

	case network.ReqBlockChainTotalQnMsg:
		core.BlockSyncer.ReqTotalQnCh <- sourceId
	case network.BlockChainTotalQnMsg:
		m, e := unmarshalTotalQnInfo(msg.Body)
		if e != nil {
			core.Logger.Errorf("[handler]Discard BlockChainTotalQnMsg because of unmarshal error:%s", e.Error())
			return e
		}
		s := core.TotalQnInfo{TotalQn: m.TotalQn, SourceId: sourceId,Height:m.Height}
		core.BlockSyncer.TotalQnCh <- s
	case network.ReqBlockInfo:
		m, e := unMarshalBlockRequestInfo(msg.Body)
		if e != nil {
			core.Logger.Errorf("[handler]Discard REQ_BLOCK_MSG_WITH_PRE because of unmarshal error:%s", e.Error())
			return e
		}
		onBlockInfoReq(*m, sourceId)
	case network.BlockInfo:
		m, e := unMarshalBlockInfo(msg.Body)
		if e != nil {
			core.Logger.Errorf("[handler]Discard BLOCK_MSG because of unmarshal error:%s", e.Error())
			return e
		}
		onBlockInfo(*m, sourceId)
	case network.BlockHashesReq:
		cbhr, e := unMarshalBlockHashesReq(msg.Body)
		if e != nil {
			core.Logger.Errorf("[handler]Discard BLOCK_CHAIN_HASHES_REQ because of unmarshal error:%s", e.Error())
			return e
		}
		onBlockHashesReq(cbhr, sourceId)
	case network.BlockHashes:
		cbh, e := unMarshalBlockHashes(msg.Body)
		if e != nil {
			core.Logger.Errorf("[handler]Discard BLOCK_CHAIN_HASHES because of unmarshal error:%s", e.Error())
			return e
		}
		onBlockHashes(cbh, sourceId)
	}
	return nil
}

func (ch ChainHandler) newBlockHeaderHandler(msg notify.Message) {
	//core.Logger.Debugf("[ChainHandler]newBlockHeaderHandler RCV MSG")
	m, ok := msg.GetData().(*notify.BlockHeaderNotifyMessage)
	if !ok {
		core.Logger.Debugf("[ChainHandler]newBlockHeaderHandler GetData assert not ok!")
		return
	}

	header, e := types.UnMarshalBlockHeader(m.HeaderByte)
	if e != nil {
		core.Logger.Errorf("[handler]Discard NewBlockHeader because of unmarshal error:%s", e.Error())
		return
	}

	notify := blockHeaderNotify{header: *header, peer: m.Peer}
	ch.headerCh <- notify
}

func (ch ChainHandler) blockBodyReqHandler(msg notify.Message) {
	//core.Logger.Debugf("[ChainHandler]blockBodyReqHandler RCV MSG")
	m, ok := msg.GetData().(*notify.BlockBodyReqMessage)
	if !ok {
		core.Logger.Debugf("[ChainHandler]blockBodyReqHandler GetData assert not ok!")
		return
	}

	hash := common.BytesToHash(m.BlockHashByte)

	body := core.BlockChainImpl.QueryBlockBody(hash)

	core.SendBlockBody(m.Peer, hash, body)
}

func (ch ChainHandler) blockBodyHandler(msg notify.Message) {
	//core.Logger.Debugf("[ChainHandler]blockBodyHandler RCV MSG")
	m, ok := msg.GetData().(*notify.BlockBodyNotifyMessage)
	if !ok {
		return
	}

	blockHash, txs, e := unMarshalBlockBody(m.BodyByte)
	if e != nil {
		core.Logger.Errorf("[handler]Discard BlockBodyMessage because of unmarshal error:%s", e.Error())
		return
	}

	notify := blockBodyNotify{blockHash: blockHash, body: txs, peer: m.Peer}
	ch.bodyCh <- notify
}

//只有重节点
func (ch ChainHandler) stateInfoReqHandler(msg notify.Message) {
	if core.BlockChainImpl.IsLightMiner() {
		return
	}

	m, ok := msg.GetData().(*notify.StateInfoReqMessage)
	if !ok {
		return
	}
	message, e := unMarshalStateInfoReq(m.StateInfoReqByte)
	if e != nil {
		core.Logger.Errorf("[handler]Discard unMarshalStateInfoReq because of unmarshal error:%s", e.Error())
		return
	}

	header := core.BlockChainImpl.QueryBlockByHeight(message.Height)
	if header == nil {
		//正在出这一块，链上没有
		core.Logger.Debugf("stateInfoReqHandler,local chain has no block! Get from block cache !height:%d,blockhash:%x", message.Height, message.BlockHash)
		b := core.BlockChainImpl.(*core.FullBlockChain).GetCastingBlock(message.BlockHash)
		if b == nil {
			core.Logger.Errorf("stateInfoReqHandler,local has no block height:%d", message.Height)
			panic("Header is nil! ")
		}
		header = b.Header
		core.Logger.Debugf("stateInfoReqHandler, Get block from block cache !height:%d,blockhash:%x", header.Height, header.Hash)
	}
	core.Logger.Errorf("stateInfoReqHandler,block height:%d", message.Height)
	preHeader := core.BlockChainImpl.QueryBlockByHash(header.PreHash)
	stateNodes := core.BlockChainImpl.GetTrieNodesByExecuteTransactions(preHeader.Header, message.Transactions, message.Addresses)
	core.SendStateInfo(m.Peer, message.Height, stateNodes, header.Hash, preHeader.Header.StateTree)
}

//只有轻节点
func (ch ChainHandler) stateInfoHandler(msg notify.Message) {
	//core.Logger.Debugf("into stateInfoHandler")
	if !core.BlockChainImpl.IsLightMiner() {
		return
	}
	m, ok := msg.GetData().(*notify.StateInfoMessage)
	if !ok {
		return
	}
	message, e := unMarshalStateInfo(m.StateInfoByte)
	if e != nil {
		core.Logger.Errorf("[handler]Discard unMarshalStateInfo because of unmarshal error:%s", e.Error())
		return
	}
	core.BlockChainImpl.InsertStateNode(message.TrieNodes)
	core.BlockChainImpl.(*core.LightChain).MarkMissNodeState(message.BlockHash)
	core.BlockChainImpl.(*core.LightChain).SetPreBlockStateRoot(message.BlockHash, message.PreBlockSateRoot)

	b := core.BlockChainImpl.(*core.LightChain).GetCachedBlock(message.Height)
	if b == nil {
		return
	}
	core.Logger.Debugf("After InsertStateNode,get cached node to add on chain! height:%d", b.Header.Height)
	result := core.BlockChainImpl.AddBlockOnChain(b)
	if result == 0 {
		core.BlockChainImpl.(*core.LightChain).RemoveFromCache(b)
		core.BlockSyncer.SetSyncedFirstBlock(true)
		core.RequestBlockInfoByHeight(m.Peer, core.BlockChainImpl.Height(), core.BlockChainImpl.QueryTopBlock().Hash, true)
	}
}

func (ch ChainHandler) loop() {
	for {
		select {
		case headerNotify := <-ch.headerCh:
			//core.Logger.Debugf("[ChainHandler]headerCh receive,hash:%v,peer:%s,tx len:%d,block:%d-%d",headerNotify.header.Hash,headerNotify.peer,len(headerNotify.header.Transactions),headerNotify.header.Height,headerNotify.header.QueueNumber)
			hash := headerNotify.header.Hash
			if _, ok := ch.headerPending[hash]; ok || ch.complete.Contains(hash) {
				//core.Logger.Debugf("[ChainHandler]header hit pending or complete")
				break
			}

			local := core.BlockChainImpl.QueryBlockByHash(hash)
			if local != nil {
				ch.complete.Add(hash, headerNotify.header)
				break
			}

			ch.headerPending[hash] = headerNotify
			core.ReqBlockBody(headerNotify.peer, hash)
		case bodyNotify := <-ch.bodyCh:
			//core.Logger.Debugf("[ChainHandler]bodyCh receive,hash:%v,peer:%s,body len:%d",bodyNotify.blockHash,bodyNotify.peer,len(bodyNotify.body))
			headerNotify, ok := ch.headerPending[bodyNotify.blockHash]
			if !ok {
				break
			}

			if headerNotify.peer != bodyNotify.peer {
				break
			}
			block := types.Block{Header: &headerNotify.header, Transactions: bodyNotify.body}
			msg := notify.BlockMessage{Block: block}
			ch.complete.Add(block.Header.Hash, block)
			delete(ch.headerPending, block.Header.Hash)
			notify.BUS.Publish(notify.NewBlock, &msg)
		}
	}
}

//-----------------------------------------------铸币-------------------------------------------------------------------

//接收索要交易请求 查询自身是否有该交易 有的话返回, 没有的话自己广播该请求
func OnTransactionRequest(m *core.TransactionRequestMessage, sourceId string) error {
	core.Logger.Debugf("receive REQ_TRANSACTION_MSG from %s,%d-%D,tx_len", sourceId, m.BlockHeight, m.CurrentBlockHash.ShortS(),len(m.TransactionHashes))
	//本地查询transaction
	if nil == core.BlockChainImpl {
		return nil
	}
	transactions, need, e := core.BlockChainImpl.GetTransactionPool().GetTransactions(m.CurrentBlockHash, m.TransactionHashes)
	if e == core.ErrNil {
		m.TransactionHashes = need
	}

	if nil != transactions && 0 != len(transactions) {
		core.SendTransactions(transactions, sourceId, m.BlockHeight, m.BlockPv)
	}

	return nil
}

//验证节点接收交易 或者接收来自客户端广播的交易
func onMessageTransaction(txs []*types.Transaction) error {
	//验证节点接收交易 加入交易池
	if nil == core.BlockChainImpl {
		return nil
	}
	e := core.BlockChainImpl.GetTransactionPool().AddTransactions(txs)
	if e != nil {
		return e
	}
	return nil
}

//全网其他节点 接收block 进行验证
func onMessageNewBlock(b *types.Block) error {
	//接收到新的块 本地上链
	if nil == core.BlockChainImpl {
		return nil
	}
	if core.BlockChainImpl.AddBlockOnChain(b) == -1 {
		core.Logger.Errorf("[handler]Add new block to chain error!")
		return fmt.Errorf("fail to add block")
	}
	return nil
}

//----------------------------------------------链调整-------------------------------------------------------------

func onBlockHashesReq(cbhr *core.BlockHashesReq, sourceId string) {
	if cbhr == nil {
		return
	}
	cbh := core.BlockChainImpl.QueryBlockHashes(cbhr.Height, cbhr.Length)
	core.SendBlockHashes(sourceId, cbh)
}

func onBlockHashes(bhs []*core.BlockHash, sourceId string) {
	//core.Logger.Debugf("Get OnChainBlockHashes from:%s", sourceId)
	core.BlockChainImpl.CompareChainPiece(bhs, sourceId)
}

func onBlockInfoReq(erm core.BlockRequestInfo, sourceId string) {
	//收到块请求
	core.Logger.Debugf("[handler]onBlockInfoReq get message from:%s", sourceId)
	if nil == core.BlockChainImpl {
		return
	}
	blockInfo := core.BlockChainImpl.QueryBlockInfo(erm.SourceHeight, erm.SourceCurrentHash, erm.VerifyHash)
	core.SendBlockInfo(sourceId, blockInfo)
}

func onBlockInfo(blockInfo core.BlockInfo, sourceId string) {
	//收到块信息
	core.Logger.Debugf("[handler] onBlockInfo get message from:%s", sourceId)
	if nil == core.BlockChainImpl {
		return
	}
	block := blockInfo.Block
	if block != nil {
		core.Logger.Debugf("[handler] onBlockInfo receive block,height:%d,qn:%d",block.Header.Height,block.Header.ProveValue)
		code := core.BlockChainImpl.AddBlockOnChain(block)
		if code < 0 {
			core.BlockChainImpl.SetAdujsting(false)
			core.Logger.Errorf("fail to add block to block chain,code:%d", code)
			return
		}
		if code == 2 {
			return
		}
		if code == 3 {
			//轻节点缺少账户状态信息
			core.BlockChainImpl.(*core.LightChain).Cache(block)
			return
		}

		if !blockInfo.IsTopBlock {
			core.RequestBlockInfoByHeight(sourceId, block.Header.Height, block.Header.Hash, true)
		} else {
			core.BlockChainImpl.SetAdujsting(false)
			if !core.BlockSyncer.IsInit() {
				core.Logger.Errorf("Block sync finished,local block height:%d\n", core.BlockChainImpl.Height())
				core.BlockSyncer.SetInit(true)
			}
		}
	} else if len(blockInfo.ChainPiece) != 0 {
		core.Logger.Debugf("[handler] onBlockInfo receive chainPiece,length:%d", len(blockInfo.ChainPiece))
		chainPiece := blockInfo.ChainPiece
		core.BlockChainImpl.CompareChainPiece(chainPiece, sourceId)
	} else {
		core.BlockChainImpl.SetAdujsting(false)
		if !core.BlockSyncer.IsInit() {
			core.Logger.Errorf("Block sync finished,local block height:%d\n", core.BlockChainImpl.Height())
			core.BlockSyncer.SetInit(true)
		}
	}
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

//--------------------------------------------------Block---------------------------------------------------------------

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

func unMarshalBlockHashesReq(byte []byte) (*core.BlockHashesReq, error) {
	b := new(tas_middleware_pb.BlockHashesReq)

	error := proto.Unmarshal(byte, b)
	if error != nil {
		network.Logger.Errorf("[handler]unMarshalChainBlockHashesReq error:%s", error.Error())
		return nil, error
	}
	r := pbToBlockHashesReq(b)
	return r, nil
}

func pbToBlockHashesReq(cbhr *tas_middleware_pb.BlockHashesReq) *core.BlockHashesReq {
	if cbhr == nil {
		return nil
	}
	r := core.BlockHashesReq{Height: *cbhr.Height, Length: *cbhr.Length}
	return &r
}

func unMarshalBlockHashes(b []byte) ([]*core.BlockHash, error) {
	blockHashSlice := new(tas_middleware_pb.BlockChainPiece)
	error := proto.Unmarshal(b, blockHashSlice)
	if error != nil {
		network.Logger.Errorf("[handler]unMarshalChainBlockHashes error:%s\n", error.Error())
		return nil, error
	}
	chainBlockHashes := blockHashSlice.BlockHashes
	result := make([]*core.BlockHash, 0)

	for _, b := range chainBlockHashes {
		h := pbToBlockHash(b)
		result = append(result, h)
	}
	return result, nil
}

func pbToBlockHash(cbh *tas_middleware_pb.BlockHash) *core.BlockHash {
	if cbh == nil {
		return nil
	}
	r := core.BlockHash{Height: *cbh.Height, Hash: common.BytesToHash(cbh.Hash)}
	return &r
}

//----------------------------------------------块同步------------------------------------------------------------------

func unMarshalBlockRequestInfo(b []byte) (*core.BlockRequestInfo, error) {
	m := new(tas_middleware_pb.BlockRequestInfo)

	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("[handler]Unmarshal EntityRequestMessage error:%s", e.Error())
		return nil, e
	}

	sourceHeight := m.SourceHeight
	sourceCurrentHash := common.BytesToHash(m.SourceCurrentHash)
	message := core.BlockRequestInfo{SourceHeight: *sourceHeight, SourceCurrentHash: sourceCurrentHash, VerifyHash: *m.VerifyHash}
	return &message, nil
}

func unMarshalBlockInfo(b []byte) (*core.BlockInfo, error) {
	message := new(tas_middleware_pb.BlockInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		network.Logger.Errorf("[handler]Unmarshal BlockMessage error:%s", e.Error())
		return nil, e
	}

	var block *types.Block
	if message.Block != nil {
		block = types.PbToBlock(message.Block)
	}

	cbh := make([]*core.BlockHash, 0)
	if message.ChainPiece != nil && message.ChainPiece.BlockHashes != nil {
		for _, b := range message.ChainPiece.BlockHashes {
			cbh = append(cbh, pbToBlockHash(b))
		}
	}

	var topBlock bool
	if message.IsTopBlock != nil {
		topBlock = *(message.IsTopBlock)
	}
	m := core.BlockInfo{Block: block, IsTopBlock: topBlock, ChainPiece: cbh}
	return &m, nil
}

func unMarshalGroups(b []byte) ([]*types.Group, error) {
	message := new(tas_middleware_pb.GroupSlice)
	e := proto.Unmarshal(b, message)
	if e != nil {
		core.Logger.Errorf("[handler]Unmarshal Groups error:%s", e.Error())
		return nil, e
	}

	groups := make([]*types.Group, 0)
	if message.Groups != nil {
		for _, g := range message.Groups {
			groups = append(groups, types.PbToGroup(g))
		}
	}
	return groups, nil
}

func unMarshalGroupInfo(b []byte) (*core.GroupInfo, error) {
	message := new(tas_middleware_pb.GroupInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		core.Logger.Errorf("[handler]unMarshalGroupInfo error:%s", e.Error())
		return nil, e
	}
	groups := make([]*types.Group, len(message.Groups))
	for i, g := range message.Groups {
		groups[i] = types.PbToGroup(g)
	}
	groupInfo := core.GroupInfo{Groups: groups, IsTopGroup: *message.IsTopGroup}
	return &groupInfo, nil
}

func unMarshalBlockBody(b []byte) (hash common.Hash, transactions []*types.Transaction, err error) {
	message := new(tas_middleware_pb.BlockBody)
	e := proto.Unmarshal(b, message)
	if e != nil {
		core.Logger.Errorf("[handler]unMarshalBlockBody error:%s", e.Error())
		return hash, nil, e
	}
	hash = common.BytesToHash(message.BlockHash)
	transactions = types.PbToTransactions(message.Transactions)
	return hash, transactions, nil
}

func unMarshalStateInfoReq(b []byte) (core.StateInfoReq, error) {
	message := new(tas_middleware_pb.StateInfoReq)
	e := proto.Unmarshal(b, message)
	if e != nil {
		core.Logger.Errorf("[handler]unMarshalStateInfoReq error:%s", e.Error())
		return core.StateInfoReq{}, e
	}

	var transactions []*types.Transaction
	if message.Transactions != nil {
		transactions = types.PbToTransactions(message.Transactions.Transactions)
	}

	var addresses []common.Address
	for _,addr := range message.Addresses{
		addresses = append(addresses,common.BytesToAddress(addr))
	}
	stateInfoReq := core.StateInfoReq{Height: *message.Height, Transactions: transactions, Addresses: addresses, BlockHash: common.BytesToHash(message.BlockHash)}
	return stateInfoReq, nil
}

func unMarshalStateInfo(b []byte) (core.StateInfo, error) {
	message := new(tas_middleware_pb.StateInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		core.Logger.Errorf("[handler]unMarshalStateInfo error:%s", e.Error())
		return core.StateInfo{}, e
	}

	trieNodes := make([]types.StateNode, 0)
	for _, node := range message.TrieNodes {
		n := types.StateNode{Key: node.Key, Value: node.Data}
		trieNodes = append(trieNodes, n)
	}

	stateInfo := core.StateInfo{Height: *message.Height, TrieNodes: &trieNodes,
		BlockHash: common.BytesToHash(message.BlockHash), PreBlockSateRoot: common.BytesToHash(message.ProBlockStateRoot)}
	return stateInfo, nil
}

func unmarshalTotalQnInfo(b []byte) (core.TotalQnInfo, error) {
	message := new(tas_middleware_pb.TotalQnInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		core.Logger.Errorf("[handler]unmarshalTotalQnInfo error:%s", e.Error())
		return core.TotalQnInfo{}, e
	}
	totalQnInfo := core.TotalQnInfo{TotalQn: *message.TotalQn, Height: *message.Height,}
	return totalQnInfo, nil
}
