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

package types

import (
	"common"
	"encoding/json"
	"time"
	"math/big"
)

const (
	TransactionTypeTransfer = 0
	TransactionTypeContractCreate = 1
	TransactionTypeContractCall = 2
	TransactionTypeBonus = 3
	TransactionTypeMinerApply = 4
	TransactionTypeMinerAbort = 5
	TransactionTypeMinerRefund = 6

	TransactionTypeToBeRemoved = -1
)

type Transaction struct {
	Data   []byte
	Value  uint64
	Nonce  uint64
	Source *common.Address
	Target *common.Address
	Type   int32

	GasLimit uint64
	GasPrice uint64
	Hash     common.Hash

	ExtraData     []byte
	ExtraDataType int32
}

func (tx *Transaction) GenHash() common.Hash {
	if nil == tx {
		return common.Hash{}
	}

	blockByte, _ := json.Marshal(tx)
	return common.BytesToHash(common.Sha256(blockByte))
}

type Transactions []*Transaction

func (c Transactions) Len() int {
	return len(c)
}
func (c Transactions) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c Transactions) Less(i, j int) bool {
	return c[i].Nonce < c[j].Nonce
}

type GasPriceTransactions []*Transaction

func (c GasPriceTransactions) Len() int {
	return len(c)
}
func (c GasPriceTransactions) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c GasPriceTransactions) Less(i, j int) bool {
	return c[i].GasPrice > c[j].GasPrice
}

// 根据gasprice决定优先级的transaction数组
// gasprice 低的，放在前
type PriorityTransactions []*Transaction

func (pt PriorityTransactions) Len() int {
	return len(pt)
}
func (pt PriorityTransactions) Swap(i, j int) {
	pt[i], pt[j] = pt[j], pt[i]
}
func (pt PriorityTransactions) Less(i, j int) bool {
	if pt[i].Type == TransactionTypeToBeRemoved && pt[j].Type != TransactionTypeToBeRemoved{
		return true
	} else if pt[i].Type != TransactionTypeToBeRemoved && pt[j].Type == TransactionTypeToBeRemoved{
		return false
	} else{
		return pt[i].GasPrice < pt[j].GasPrice
	}
}
func (pt *PriorityTransactions) Push(x interface{}) {
	item := x.(*Transaction)
	*pt = append(*pt, item)
}

func (pt *PriorityTransactions) Pop() interface{} {
	old := *pt
	n := len(old)
	item := old[n-1]

	*pt = old[0: n-1]
	return item
}

type Bonus struct {
	TxHash		common.Hash
	TargetIds	[]int32
	BlockHash	common.Hash
	GroupId		[]byte
	Sign		[]byte
	TotalValue	uint64
}

const (
	MinerTypeLight  = 0
	MinerTypeHeavy  = 1
	MinerStatusNormal = 0
	MinerStatusAbort = 1
)

type Miner struct {
	Id				[]byte
	PublicKey 		[]byte
	VrfPublicKey 	[]byte
	ApplyHeight 	uint64
	Stake			uint64
	AbortHeight		uint64
	Type			byte
	Status			byte
}


//区块头结构
type BlockHeader struct {
	Hash         common.Hash   // 本块的hash，to do : 是对哪些数据的哈希
	Height       uint64        // 本块的高度
	PreHash      common.Hash   //上一块哈希
	PreTime      time.Time     //上一块铸块时间
	ProveValue   *big.Int      //轮转序号
	TotalQN      uint64      //整条链的QN
	CurTime      time.Time     //当前铸块时间
	Castor       []byte        //出块人ID
	GroupId      []byte        //组ID，groupsig.ID的二进制表示
	Signature    []byte        // 组签名
	Nonce        uint64        //盐
	Transactions []common.Hash // 交易集哈希列表
	TxTree       common.Hash   // 交易默克尔树根hash
	ReceiptTree  common.Hash
	StateTree    common.Hash
	ExtraData    []byte
	Random       []byte
	ProveRoot	 common.Hash
}

type header struct {
	Height       uint64        // 本块的高度
	PreHash      common.Hash   //上一块哈希
	PreTime      time.Time     //上一块铸块时间
	ProveValue   *big.Int        //轮转序号
	TotalQN      uint64       //整条链的QN
	CurTime      time.Time     //当前铸块时间
	Castor       []byte        //出块人ID
	GroupId      []byte        //组ID，groupsig.ID的二进制表示
	Nonce        uint64        //盐
	Transactions []common.Hash // 交易集哈希列表
	TxTree       common.Hash   // 交易默克尔树根hash
	ReceiptTree  common.Hash
	StateTree    common.Hash
	ExtraData    []byte
	ProveRoot	 common.Hash
}

func (bh *BlockHeader) GenHash() common.Hash {
	header := &header{
		Height:       bh.Height,
		PreHash:      bh.PreHash,
		PreTime:      bh.PreTime,
		ProveValue:   bh.ProveValue,
		TotalQN:      bh.TotalQN,
		CurTime:      bh.CurTime,
		Castor:       bh.Castor,
		Nonce:        bh.Nonce,
		Transactions: bh.Transactions,
		TxTree:       bh.TxTree,
		ReceiptTree:  bh.ReceiptTree,
		StateTree:    bh.StateTree,
		ExtraData:    bh.ExtraData,
		ProveRoot:    bh.ProveRoot,
	}
	blockByte, _ := json.Marshal(header)
	result := common.BytesToHash(common.Sha256(blockByte))

	return result
}

type Block struct {
	Header       *BlockHeader
	Transactions []*Transaction
}

type Member struct {
	Id     []byte
	PubKey []byte
}
type Group struct {
	Id          []byte
	Members     []Member
	PubKey      []byte
	Parent      []byte //父亲组 的组ID
	PreGroup	[]byte //前一块的ID
	Dummy       []byte
	Signature   []byte
	BeginHeight uint64 //组开始参与铸块的高度
	DismissHeight uint64				//组解散的高度
	Authority 	uint64      //权限相关数据（父亲组赋予）
	Name      	string    //父亲组取的名字
	Extends   	string      //带外数据
	GroupHeight uint64
}

type StateNode struct {
	Key 	[]byte
	Value 	[]byte
}