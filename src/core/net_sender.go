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
)

type TransactionRequestMessage struct {
	TransactionHashes []common.Hash
	CurrentBlockHash  common.Hash
	BlockHeight       uint64
	BlockQn           uint64
}

type BlockHashesReq struct {
	Height uint64 //起始高度
	Length uint64 //从起始高度开始,向前的非空长度
}

type BlockHash struct {
	Height uint64 //所在链高度
	Hash   common.Hash
	Qn     uint64
}

type BlockRequestInfo struct {
	SourceHeight      uint64
	SourceCurrentHash common.Hash
}

type BlockInfo struct {
	Block      *types.Block
	IsTopBlock bool
	ChainPiece []*BlockHash
}

//验证节点 交易集缺失，向CASTOR索要特定交易
func RequestTransaction(m TransactionRequestMessage, castorId string) {
	if castorId == "" {
		return
	}

	body, e := marshalTransactionRequestMessage(&m)
	if e != nil {
		Logger.Errorf("[peer]Discard MarshalTransactionRequestMessage because of marshal error:%s!", e.Error())
		return
	}
	//network.Logger.Debugf("send REQ_TRANSACTION_MSG to %s,%d-%d,tx_len:%d,time at:%v", castorId, m.BlockHeight, m.BlockQn, len(m.TransactionHashes), time.Now())
	message := network.Message{Code: network.ReqTransactionMsg, Body: body}
	network.GetNetInstance().Send(castorId, message)
}

//本地查询到交易，返回请求方
func SendTransactions(txs []*types.Transaction, sourceId string, blockHeight uint64, blockQn uint64) {
	body, e := types.MarshalTransactions(txs)
	if e != nil {
		Logger.Errorf("[peer]Discard MarshalTransactions because of marshal error:%s!", e.Error())
		return
	}
	//network.Logger.Debugf("send TRANSACTION_GOT_MSG to %s,%d-%d,tx_len,time at:%v",sourceId,blockHeight,blockQn,len(txs),time.Now())
	message := network.Message{Code: network.TransactionGotMsg, Body: body}
	network.GetNetInstance().Send(sourceId, message)
}

//收到交易 全网扩散
func BroadcastTransactions(txs []*types.Transaction) {
	defer func() {
		if r := recover(); r != nil {
			Logger.Errorf("[peer]Runtime error caught: %v", r)
		}
	}()

	body, e := types.MarshalTransactions(txs)
	if e != nil {
		Logger.Errorf("[peer]Discard MarshalTransactions because of marshal error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.TransactionMsg, Body: body}
	network.GetNetInstance().Relay(message, 3)
}

//向某一节点请求Block信息
func RequestBlockInfoByHeight(id string, localHeight uint64, currentHash common.Hash) {
	Logger.Debugf("Req block info to:%s,localHeight:%d,current hash:%v",id,localHeight,currentHash)
	m := BlockRequestInfo{SourceHeight: localHeight, SourceCurrentHash: currentHash}
	body, e := MarshalBlockRequestInfo(&m)
	if e != nil {
		Logger.Errorf("[peer]RequestBlockInfoByHeight marshal EntityRequestMessage error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.ReqBlockInfo, Body: body}
	network.GetNetInstance().Send(id, message)
}

//本地查询之后将结果返回
func SendBlockInfo(targetId string, blockInfo *BlockInfo) {
	Logger.Debugf("Send local block info to:%s",targetId)
	body, e := marshalBlockInfo(blockInfo)
	if e != nil {
		Logger.Errorf("[peer]SendBlockInfo marshal BlockEntity error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.BlockInfo, Body: body}
	network.GetNetInstance().Send(targetId, message)
}

//向目标结点索要 block hash
func RequestBlockHashes(targetNode string, bhr BlockHashesReq) {
	body, e := marshalBlockHashesReq(&bhr)
	if e != nil {
		Logger.Errorf("[peer]Discard RequestBlockChainHashes because of marshal error:%s!", e.Error())
		return
	}
	message := network.Message{Code: network.BlockHashesReq, Body: body}
	network.GetNetInstance().Send(targetNode, message)
}

//向目标结点发送 block hash
func SendBlockHashes(targetNode string, bhs []*BlockHash) {
	body, e := marshalBlockHashes(bhs)
	if e != nil {
		Logger.Errorf("[peer]Discard sendChainBlockHashes because of marshal error:%s!", e.Error())
		return
	}
	message := network.Message{Code: network.BlockHashes, Body: body}
	network.GetNetInstance().Send(targetNode, message)
}

func ReqBlockBody(targetNode string, blockHash common.Hash) {
	body := blockHash.Bytes()

	message := network.Message{Code: network.BlockBodyReqMsg, Body: body}
	network.GetNetInstance().Send(targetNode, message)
}

func SendBlockBody(targetNode string, blockHash common.Hash, transactions []*types.Transaction) {
	body, e := marshalBlockBody(blockHash, transactions)
	if e != nil {
		Logger.Errorf("[peer]Discard MarshalTransactions because of marshal error:%s!", e.Error())
		return
	}

	message := network.Message{Code: network.BlockBodyMsg, Body: body}
	network.GetNetInstance().Send(targetNode, message)
}

//--------------------------------------------------Transaction---------------------------------------------------------------
func marshalTransactionRequestMessage(m *TransactionRequestMessage) ([]byte, error) {
	txHashes := make([][]byte, 0)
	for _, txHash := range m.TransactionHashes {
		txHashes = append(txHashes, txHash.Bytes())
	}

	currentBlockHash := m.CurrentBlockHash.Bytes()
	message := tas_middleware_pb.TransactionRequestMessage{TransactionHashes: txHashes, CurrentBlockHash: currentBlockHash, BlockHeight: &m.BlockHeight, BlockQn: &m.BlockQn}
	return proto.Marshal(&message)
}

//--------------------------------------------------Block---------------------------------------------------------------

func marshalBlockHashesReq(req *BlockHashesReq) ([]byte, error) {
	if req == nil {
		return nil, nil
	}
	cbhr := blockHashesReqToPb(req)
	return proto.Marshal(cbhr)
}

func blockHashesReqToPb(req *BlockHashesReq) *tas_middleware_pb.BlockHashesReq {
	if req == nil {
		return nil
	}
	cbhr := tas_middleware_pb.BlockHashesReq{Height: &req.Height, Length: &req.Length}
	return &cbhr
}
func marshalBlockHashes(cbh []*BlockHash) ([]byte, error) {
	if cbh == nil {
		return nil, nil
	}

	blockHashes := make([]*tas_middleware_pb.BlockHash, 0)
	for _, c := range cbh {
		blockHashes = append(blockHashes, blockHashToPb(c))
	}
	r := tas_middleware_pb.BlockChainPiece{BlockHashes: blockHashes}
	return proto.Marshal(&r)
}

func blockHashToPb(bh *BlockHash) *tas_middleware_pb.BlockHash {
	if bh == nil {
		return nil
	}

	r := tas_middleware_pb.BlockHash{Hash: bh.Hash.Bytes(), Height: &bh.Height}
	return &r
}

func marshalBlockBody(blockHash common.Hash, transactions []*types.Transaction) ([]byte, error) {
	hash := blockHash.Bytes()

	txs := types.TransactionsToPb(transactions)
	blockBody := tas_middleware_pb.BlockBody{BlockHash: hash, Transactions: txs}
	return proto.Marshal(&blockBody)
}
func MarshalBlockRequestInfo(e *BlockRequestInfo) ([]byte, error) {
	sourceHeight := e.SourceHeight
	currentHash := e.SourceCurrentHash.Bytes()

	m := tas_middleware_pb.BlockRequestInfo{SourceHeight: &sourceHeight, SourceCurrentHash: currentHash}
	return proto.Marshal(&m)
}

func marshalBlockInfo(e *BlockInfo) ([]byte, error) {
	if e == nil {
		return nil, nil
	}

	var block *tas_middleware_pb.Block
	if e.Block != nil {
		block = types.BlockToPb(e.Block)
		if block == nil {
			Logger.Errorf("Block is nil while marshalBlockMessage")
		}
	}

	cbh := make([]*tas_middleware_pb.BlockHash, 0)
	if e.ChainPiece != nil {
		for _, b := range e.ChainPiece {
			pb := blockHashToPb(b)
			if pb == nil {
				Logger.Errorf("ChainBlockHash is nil while marshalBlockMessage")
			}
			cbh = append(cbh, pb)
		}
	}
	cbhs := tas_middleware_pb.BlockChainPiece{BlockHashes: cbh}

	message := tas_middleware_pb.BlockInfo{Block: block, IsTopBlock: &e.IsTopBlock, ChainPiece: &cbhs}
	return proto.Marshal(&message)
}