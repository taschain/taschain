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
	"github.com/gogo/protobuf/proto"
	"middleware/pb"
	"middleware/types"
	"network"
	"math/big"
	"time"
	"utility"
)

type TransactionRequestMessage struct {
	TransactionHashes []common.Hash
	CurrentBlockHash  common.Hash
	BlockHeight       uint64
	BlockPv           *big.Int
}

type StateInfoReq struct {
	Height       uint64
	BlockHash    common.Hash
	Transactions types.Transactions
	Addresses    []common.Address
}

type StateInfo struct {
	Height           uint64
	BlockHash        common.Hash
	TrieNodes        *[]types.StateNode
	PreBlockSateRoot common.Hash
}

type ChainPieceInfo struct {
	ChainPiece []*types.BlockHeader
	TopHeader  *types.BlockHeader
}

//验证节点 交易集缺失，向CASTOR索要特定交易
func RequestTransaction(m TransactionRequestMessage, castorId string) {
	if castorId == "" {
		return
	}

	body, e := marshalTransactionRequestMessage(&m)
	if e != nil {
		Logger.Errorf("Discard MarshalTransactionRequestMessage because of marshal error:%s!", e.Error())
		return
	}
	Logger.Debugf("send REQ_TRANSACTION_MSG to %s,%d-%d,tx_len:%d,time at:%v", castorId, m.BlockHeight, m.CurrentBlockHash.ShortS(), len(m.TransactionHashes), time.Now())
	message := network.Message{Code: network.ReqTransactionMsg, Body: body}
	network.GetNetInstance().Send(castorId, message)
}

//本地查询到交易，返回请求方
func SendTransactions(txs []*types.Transaction, sourceId string, blockHeight uint64, blockPv *big.Int) {
	body, e := types.MarshalTransactions(txs)
	if e != nil {
		Logger.Errorf("Discard MarshalTransactions because of marshal error:%s!", e.Error())
		return
	}
	//network.Logger.Debugf("send TRANSACTION_GOT_MSG to %s,%d-%d,tx_len,time at:%v",sourceId,blockHeight,blockQn,len(txs),time.Now())
	message := network.Message{Code: network.TransactionGotMsg, Body: body}
	go network.GetNetInstance().Send(sourceId, message)
}

func BroadcastMinerTransactions(txs []*types.Transaction) {
	defer func() {
		if r := recover(); r != nil {
			Logger.Errorf("Runtime error caught: %v", r)
		}
	}()
	if len(txs) > 0 {
		body, e := types.MarshalTransactions(txs)
		if e != nil {
			Logger.Errorf("Discard MarshalTransactions because of marshal error:%s", e.Error())
			return
		}
		Logger.Debugf("Broadcast Miner Transactions len:%d", len(txs))
		message := network.Message{Code: network.MinerTransactionMsg, Body: body}
		heavyMiners := MinerManagerImpl.GetHeavyMiners()
		go network.GetNetInstance().SpreadToRandomGroupMember(network.FULL_NODE_VIRTUAL_GROUP_ID, heavyMiners, message)
	}
}

func BroadcastTransactions(txs []*types.Transaction, heavyOnly bool) {
	//defer func() {
	//	if r := recover(); r != nil {
	//		Logger.Errorf("Runtime error caught: %v", r)
	//	}
	//}()
	//if len(txs) > 0 {
	//	body, e := types.MarshalTransactions(txs)
	//	if e != nil {
	//		Logger.Errorf("Discard MarshalTransactions because of marshal error:%s", e.Error())
	//		return
	//	}
	//	Logger.Debugf("BroadcastTransactions len:%d", len(txs))
	//	message := network.Message{Code: network.TransactionMsg, Body: body}
	//	if heavyOnly {
	//		heavyMiners := MinerManagerImpl.GetHeavyMiners()
	//		go network.GetNetInstance().SpreadToRandomGroupMember(network.FULL_NODE_VIRTUAL_GROUP_ID, heavyMiners, message)
	//	} else {
	//		go network.GetNetInstance().Broadcast(message)
	//	}
	//}
}

//向某一节点请求Block信息
func RequestBlock(id string, height uint64) {
	Logger.Debugf("Req block to:%s,height:%d", id, height)
	body := utility.UInt64ToByte(height)
	message := network.Message{Code: network.ReqBlock, Body: body}
	go network.GetNetInstance().Send(id, message)
}

//本地查询之后将结果返回
func SendBlock(targetId string, block *types.Block) {
	if block == nil {
		return
	}
	Logger.Debugf("Send local block:%d  to:%s", block.Header.Height, targetId)
	body, e := types.MarshalBlock(block)
	if e != nil {
		Logger.Errorf("SendBlock marshal MarshalBlock error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.BlockMsg, Body: body}
	go network.GetNetInstance().Send(targetId, message)
}

func ReqBlockBody(targetNode string, blockHash common.Hash) {
	body := blockHash.Bytes()

	message := network.Message{Code: network.BlockBodyReqMsg, Body: body}
	go network.GetNetInstance().Send(targetNode, message)
}

func SendBlockBody(targetNode string, blockHash common.Hash, transactions []*types.Transaction) {
	if len(transactions) == 0 {
		return
	}
	body, e := marshalBlockBody(blockHash, transactions)
	if e != nil {
		Logger.Errorf("Discard MarshalTransactions because of marshal error:%s!", e.Error())
		return
	}

	message := network.Message{Code: network.BlockBodyMsg, Body: body}
	go network.GetNetInstance().Send(targetNode, message)
}

func ReqStateInfo(targetNode string, blockHeight uint64, qn *big.Int, txs types.Transactions, addresses []common.Address, blockHash common.Hash) {
	Logger.Debugf("Req state info to:%s,blockHeight:%d,qn:%d,len txs:%d,len addresses:%d", targetNode, blockHeight, qn, len(txs), len(addresses))
	m := StateInfoReq{Height: blockHeight, Transactions: txs, Addresses: addresses, BlockHash: blockHash}
	body, e := marshalStateInfoReq(m)
	if e != nil {
		Logger.Errorf("Discard MarshalStateInfoReq because of marshal error:%s!", e.Error())
		return
	}
	message := network.Message{Code: network.ReqStateInfoMsg, Body: body}
	network.GetNetInstance().Send(targetNode, message)
}

func SendStateInfo(targetNode string, blockHeight uint64, stateInfo *[]types.StateNode, blockHash common.Hash, preBlockStateroot common.Hash) {
	m := StateInfo{Height: blockHeight, TrieNodes: stateInfo, BlockHash: blockHash, PreBlockSateRoot: preBlockStateroot}
	body, e := marshalStateInfo(m)
	if e != nil {
		Logger.Errorf("Discard MarshalTrieNodes because of marshal error:%s!", e.Error())
		return
	}
	message := network.Message{Code: network.StateInfoMsg, Body: body}
	network.GetNetInstance().Send(targetNode, message)
}

func RequestChainPiece(targetNode string, height uint64) {
	Logger.Debugf("Req chain piece to:%s,local height:%d", targetNode, height)
	body := utility.UInt64ToByte(height)
	message := network.Message{Code: network.ChainPieceReq, Body: body}
	network.GetNetInstance().Send(targetNode, message)
}

func SendChainPiece(targetNode string, chainPieceInfo ChainPieceInfo) {
	chainPiece := chainPieceInfo.ChainPiece
	if len(chainPiece) == 0 {
		return
	}
	Logger.Debugf("Send chain piece %d-%d to:%s", chainPiece[len(chainPiece)-1].Height, chainPiece[0].Height, targetNode)
	body, e := marshalChainPieceInfo(chainPieceInfo)
	if e != nil {
		Logger.Errorf("Discard marshalChainPiece because of marshal error:%s!", e.Error())
		return
	}
	message := network.Message{Code: network.ChainPiece, Body: body}
	network.GetNetInstance().Send(targetNode, message)
}

//--------------------------------------------------Transaction---------------------------------------------------------------
func marshalTransactionRequestMessage(m *TransactionRequestMessage) ([]byte, error) {
	txHashes := make([][]byte, 0)
	for _, txHash := range m.TransactionHashes {
		txHashes = append(txHashes, txHash.Bytes())
	}

	currentBlockHash := m.CurrentBlockHash.Bytes()
	message := tas_middleware_pb.TransactionRequestMessage{TransactionHashes: txHashes, CurrentBlockHash: currentBlockHash, BlockHeight: &m.BlockHeight, BlockPv: m.BlockPv.Bytes()}
	return proto.Marshal(&message)
}

//--------------------------------------------------Block---------------------------------------------------------------

func marshalBlockBody(blockHash common.Hash, transactions []*types.Transaction) ([]byte, error) {
	hash := blockHash.Bytes()

	txs := types.TransactionsToPb(transactions)
	blockBody := tas_middleware_pb.BlockBody{BlockHash: hash, Transactions: txs}
	return proto.Marshal(&blockBody)
}

func marshalStateInfoReq(stateInfoReq StateInfoReq) ([]byte, error) {

	var txSlice tas_middleware_pb.TransactionSlice
	if stateInfoReq.Transactions != nil {
		txs := types.TransactionsToPb(stateInfoReq.Transactions)
		txSlice = tas_middleware_pb.TransactionSlice{Transactions: txs}
	}

	var addresses [][]byte
	for _, addr := range stateInfoReq.Addresses {
		addresses = append(addresses, addr.Bytes())
	}
	message := tas_middleware_pb.StateInfoReq{Height: &stateInfoReq.Height, Transactions: &txSlice, Addresses: addresses, BlockHash: stateInfoReq.BlockHash.Bytes()}
	return proto.Marshal(&message)
}

func marshalStateInfo(stateInfo StateInfo) ([]byte, error) {
	var trieNodes = make([]*tas_middleware_pb.TrieNode, 0)
	for _, node := range *stateInfo.TrieNodes {
		tNode := tas_middleware_pb.TrieNode{Key: node.Key, Data: node.Value}
		trieNodes = append(trieNodes, &tNode)
	}

	message := tas_middleware_pb.StateInfo{Height: &stateInfo.Height, TrieNodes: trieNodes,
		BlockHash: stateInfo.BlockHash.Bytes(), ProBlockStateRoot: stateInfo.PreBlockSateRoot.Bytes()}
	return proto.Marshal(&message)
}

func marshalChainPieceInfo(chainPieceInfo ChainPieceInfo) ([]byte, error) {
	headers := make([]*tas_middleware_pb.BlockHeader, 0)
	for _, header := range chainPieceInfo.ChainPiece {
		h := types.BlockHeaderToPb(header)
		headers = append(headers, h)
	}
	topHeader := types.BlockHeaderToPb(chainPieceInfo.TopHeader)
	message := tas_middleware_pb.ChainPieceInfo{TopHeader: topHeader, BlockHeaders: headers}
	return proto.Marshal(&message)
}
