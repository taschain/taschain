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
	"time"
	"math/big"

	"common"
	"network"
	"middleware/pb"
	"middleware/types"

	"github.com/gogo/protobuf/proto"
)

type transactionRequestMessage struct {
	TransactionHashes []common.Hash
	CurrentBlockHash  common.Hash
	BlockHeight       uint64
	BlockPv           *big.Int
}


func requestTransaction(m transactionRequestMessage, castorId string) {
	if castorId == "" {
		return
	}

	body, e := marshalTransactionRequestMessage(&m)
	if e != nil {
		Logger.Errorf("Discard MarshalTransactionRequestMessage because of marshal error:%s!", e.Error())
		return
	}
	Logger.Debugf("send REQ_TRANSACTION_MSG to %s,height:%d,tx_len:%d,hash:%s,time at:%v", castorId, m.BlockHeight, m.CurrentBlockHash, len(m.TransactionHashes), time.Now())
	message := network.Message{Code: network.ReqTransactionMsg, Body: body}
	network.GetNetInstance().Send(castorId, message)
}

func sendTransactions(txs []*types.Transaction, sourceId string) {
	body, e := types.MarshalTransactions(txs)
	if e != nil {
		Logger.Errorf("Discard MarshalTransactions because of marshal error:%s!", e.Error())
		return
	}
	message := network.Message{Code: network.TransactionGotMsg, Body: body}
	go network.GetNetInstance().Send(sourceId, message)
}

func broadcastTransactions(txs []*types.Transaction) {
	defer func() {
		if r := recover(); r != nil {
			Logger.Errorf("Runtime error caught: %v", r)
		}
	}()
	if len(txs) > 0 {
		body, e := types.MarshalTransactions(txs)
		if e != nil {
			Logger.Errorf("Marshal txs error:%s", e.Error())
			return
		}
		Logger.Debugf("Broadcast transactions len:%d", len(txs))
		message := network.Message{Code: network.TransactionBroadcastMsg, Body: body}
		heavyMiners := MinerManagerImpl.GetHeavyMiners()

		netInstance := network.GetNetInstance()
		if netInstance != nil {
			go network.GetNetInstance().SpreadToRandomGroupMember(network.FULL_NODE_VIRTUAL_GROUP_ID, heavyMiners, message)
		}
	}
}



func marshalTransactionRequestMessage(m *transactionRequestMessage) ([]byte, error) {
	txHashes := make([][]byte, 0)
	for _, txHash := range m.TransactionHashes {
		txHashes = append(txHashes, txHash.Bytes())
	}

	currentBlockHash := m.CurrentBlockHash.Bytes()
	message := tas_middleware_pb.TransactionRequestMessage{TransactionHashes: txHashes, CurrentBlockHash: currentBlockHash, BlockHeight: &m.BlockHeight, BlockPv: m.BlockPv.Bytes()}
	return proto.Marshal(&message)
}



