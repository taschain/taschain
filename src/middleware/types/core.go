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
	"bytes"
)

const (
	TransactionTypeTransfer       = 0
	TransactionTypeContractCreate = 1
	TransactionTypeContractCall   = 2
	TransactionTypeBonus          = 3
	TransactionTypeMinerApply     = 4
	TransactionTypeMinerAbort     = 5
	TransactionTypeMinerRefund    = 6

	TransactionTypeToBeRemoved = -1
)

var testTxAccount = []string{"0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103", "0xcad6d60fa8f6330f293f4f57893db78cf660e80d6a41718c7ad75e76795000d4",
	"0xca789a28069db6f1639b60a8bf1084333358672f65c6d6c2e6d58b69187fe402", "0x94bdb92d329dac69d7f107995a7b666d1092c63eadeae2dd495ab2e554bb155d",
	"0xb50eea221a1eb061dea7ca20f7b7508c2d9639e3558e69f758380e32624337b5", "0xce59fd5e1c6c99d9990b08ccf685260a2b3a03889de56e91b25878a4bf2f89e9",
	"0x5d9b2132ec1d2011f488648a8dc24f9b29ca40933ca89d8d19367280dff59a03", "0x5afb7e2617f1dd729ea3557096021e2f4eaa1a9c8fe48d8132b1f6cf13338a8f",
	"0x30c049d276610da3355f6c11de8623ec6b40fd2a73bb5d647df2ae83c30244bc", "0xa2b7bc555ca535745a7a9c55f9face88fc286a8b316352afc457ffafb40a7478"}

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
	//PubKey *common.PublicKey
	Sign *common.Sign
}

//source,sign在hash计算范围内
func (tx *Transaction) GenHash() common.Hash {
	if nil == tx {
		return common.Hash{}
	}
	buffer := bytes.Buffer{}
	if tx.Data != nil {
		buffer.Write(tx.Data)
	}
	buffer.Write(common.Uint64ToByte(tx.Value))
	buffer.Write(common.Uint64ToByte(tx.Nonce))
	if tx.Target != nil {
		buffer.Write(tx.Target.Bytes())
	}
	buffer.Write(common.UInt32ToByte(tx.Type))
	buffer.Write(common.Uint64ToByte(tx.GasLimit))
	buffer.Write(common.Uint64ToByte(tx.GasPrice))
	if tx.ExtraData != nil {
		buffer.Write(tx.ExtraData)
	}
	buffer.Write(common.UInt32ToByte(tx.ExtraDataType))

	return common.BytesToHash(common.Sha256(buffer.Bytes()))
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
	if pt[i].Type == TransactionTypeToBeRemoved && pt[j].Type != TransactionTypeToBeRemoved {
		return true
	} else if pt[i].Type != TransactionTypeToBeRemoved && pt[j].Type == TransactionTypeToBeRemoved {
		return false
	} else {
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

	*pt = old[0 : n-1]
	return item
}

type Bonus struct {
	TxHash     common.Hash
	TargetIds  []int32
	BlockHash  common.Hash
	GroupId    []byte
	Sign       []byte
	TotalValue uint64
}

const (
	MinerTypeLight    = 0
	MinerTypeHeavy    = 1
	MinerStatusNormal = 0
	MinerStatusAbort  = 1
)

type Miner struct {
	Id           []byte
	PublicKey    []byte
	VrfPublicKey []byte
	ApplyHeight  uint64
	Stake        uint64
	AbortHeight  uint64
	Type         byte
	Status       byte
}

//区块头结构
type BlockHeader struct {
	Hash         common.Hash   // 本块的hash，to do : 是对哪些数据的哈希
	Height       uint64        // 本块的高度
	PreHash      common.Hash   //上一块哈希
	PreTime      time.Time     //上一块铸块时间
	ProveValue   *big.Int      //轮转序号
	TotalQN      uint64        //整条链的QN
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
	ProveRoot    common.Hash
	EvictedTxs   []common.Hash
}

type header struct {
	Height       uint64        // 本块的高度
	PreHash      common.Hash   //上一块哈希
	PreTime      time.Time     //上一块铸块时间
	ProveValue   *big.Int      //轮转序号
	TotalQN      uint64        //整条链的QN
	CurTime      time.Time     //当前铸块时间
	Castor       []byte        //出块人ID
	GroupId      []byte        //组ID，groupsig.ID的二进制表示
	Nonce        uint64        //盐
	Transactions []common.Hash // 交易集哈希列表
	TxTree       common.Hash   // 交易默克尔树根hash
	ReceiptTree  common.Hash
	StateTree    common.Hash
	ExtraData    []byte
	ProveRoot    common.Hash
	EvictedTxs   []common.Hash
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
		EvictedTxs:   bh.EvictedTxs,
	}
	blockByte, _ := json.Marshal(header)
	result := common.BytesToHash(common.Sha256(blockByte))

	return result
}

func (bh *BlockHeader) ToString() string {
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
		EvictedTxs:   bh.EvictedTxs,
	}
	blockByte, _ := json.Marshal(header)
	return string(blockByte)
}

type Block struct {
	Header       *BlockHeader
	Transactions []*Transaction
}

type Member struct {
	Id     []byte
	PubKey []byte
}

type GroupHeader struct {
	Hash          common.Hash //组头hash
	Parent        []byte      //父亲组 的组ID
	PreGroup      []byte      //前一块的ID
	Authority     uint64      //权限相关数据（父亲组赋予）
	Name          string      //父亲组取的名字
	BeginTime     time.Time
	MemberRoot    common.Hash //成员列表hash
	CreateHeight  uint64      //建组高度
	ReadyHeight   uint64      //准备就绪最迟高度
	WorkHeight    uint64      //组开始参与铸块的高度
	DismissHeight uint64      //组解散的高度
	Extends       string      //带外数据
}

func (gh *GroupHeader) GenHash() common.Hash {
	buf := bytes.Buffer{}
	buf.Write(gh.Parent)
	buf.Write(gh.PreGroup)
	buf.Write(common.Uint64ToByte(gh.Authority))
	buf.WriteString(gh.Name)

	//bt, _ := gh.BeginTime.MarshalBinary()
	//buf.Write(bt)
	buf.Write(gh.MemberRoot.Bytes())
	buf.Write(common.Uint64ToByte(gh.CreateHeight))
	buf.Write(common.Uint64ToByte(gh.ReadyHeight))
	buf.Write(common.Uint64ToByte(gh.WorkHeight))
	buf.Write(common.Uint64ToByte(gh.DismissHeight))
	buf.WriteString(gh.Extends)
	return common.BytesToHash(common.Sha256(buf.Bytes()))
}

type Group struct {
	Header *GroupHeader
	//不参与签名
	Id          []byte
	PubKey      []byte
	Signature   []byte
	Members     [][]byte //成员id列表
	GroupHeight uint64
}

type StateNode struct {
	Key   []byte
	Value []byte
}

func IsTestTransaction(tx *Transaction) bool {
	if tx == nil || tx.Source == nil {
		return false
	}

	source := tx.Source.GetHexString()
	for _, testAccount := range testTxAccount {
		if source == testAccount {
			return true
		}
	}
	return false
}
