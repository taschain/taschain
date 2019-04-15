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
	"bytes"
	"common"
	"encoding/json"
	"time"
	"utility"
)

type AddBlockOnChainSituation string

const (
	Sync                  AddBlockOnChainSituation = "sync"
	NewBlock              AddBlockOnChainSituation = "newBlock"
	FutureBlockCache      AddBlockOnChainSituation = "futureBlockCache"
	DependGroupBlock      AddBlockOnChainSituation = "dependGroupBlock"
	LocalGenerateNewBlock AddBlockOnChainSituation = "localGenerateNewBlock"
	MergeFork             AddBlockOnChainSituation = "mergeFork"
)

type AddBlockResult int8

const (
	AddBlockFailed            AddBlockResult = -1
	AddBlockSucc              AddBlockResult = 0
	BlockExisted              AddBlockResult = 1
	BlockTotalQnLessThanLocal AddBlockResult = 2
	Forking                   AddBlockResult = 3
)
const (
	SUCCESS                             = 0
	TxErrorCode_BalanceNotEnough        = 1
	TxErrorCode_ContractAddressConflict = 2
	TxErrorCode_DeployGasNotEnough      = 3
	TxErrorCode_NO_CODE                 = 4
	TxErrorCode_AccounNotExist      	= 5

	Syntax_Error = 1001
	GasNotEnough = 1002

	Sys_Error                        = 2001
	Sys_Check_Abi_Error              = 2002
	Sys_Abi_JSON_Error               = 2003
	Sys_CONTRACT_CALL_MAX_DEEP_Error = 2004
)

var (
	NO_CODE_ERROR           = 4
	NO_CODE_ERROR_MSG       = "get code from address %s,but no code!"
	ABI_JSON_ERROR          = 2003
	ABI_JSON_ERROR_MSG      = "abi json format error"
	CALL_MAX_DEEP_ERROR     = 2004
	CALL_MAX_DEEP_ERROR_MSG = "call max deep cannot more than 8"
	INIT_CONTRACT_ERROR     = 2005
	INIT_CONTRACT_ERROR_MSG = "contract init error"
)

var (
	TxErrorBalanceNotEnough   = NewTransactionError(TxErrorCode_BalanceNotEnough, "balance not enough")
	TxErrorDeployGasNotEnough = NewTransactionError(TxErrorCode_DeployGasNotEnough, "gas not enough")
	TxErrorAbiJson            = NewTransactionError(Sys_Abi_JSON_Error, "abi json format error")
	TxErrorAccounNotExist= NewTransactionError(TxErrorCode_AccounNotExist, "accoun not exist")

)

type TransactionError struct {
	Code    int
	Message string
}

func NewTransactionError(code int, msg string) *TransactionError {
	return &TransactionError{Code: code, Message: msg}
}

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



//tx data with source
type Transaction struct {
	Data   []byte          `msgpack:"dt,omitempty"`
	Value  uint64          `msgpack:"v"`
	Nonce  uint64          `msgpack:"nc"`
	Target *common.Address `msgpack:"tg,omitempty"`
	Type   int8            `msgpack:"tp"`

	GasLimit uint64      `msgpack:"gl"`
	GasPrice uint64      `msgpack:"gp"`
	Hash     common.Hash `msgpack:"h"`

	ExtraData     []byte `msgpack:"ed"`
	ExtraDataType int8   `msgpack:"et,omitempty"`
	//PubKey *common.PublicKey
	//Sign *common.Sign
	TargetAccount string   `msgpack:"tgac,omitempty"`
	SourceAccount string   `msgpack:"tgac,omitempty"`
	Sign   []byte          `msgpack:"si"`
	Source *common.Address `msgpack:"src"`	//don't streamlize

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


	buffer.WriteByte(byte(tx.Type))
	buffer.Write(common.Uint64ToByte(tx.GasLimit))
	buffer.Write(common.Uint64ToByte(tx.GasPrice))
	if tx.ExtraData != nil {
		buffer.Write(tx.ExtraData)
	}
	buffer.WriteByte(byte(tx.ExtraDataType))
	if len(tx.TargetAccount) > 0 {
		buffer.Write([]byte(tx.TargetAccount))
	} else if tx.Target != nil {
		buffer.Write(tx.Target.Bytes())
	}
	if len(tx.SourceAccount) > 0 {
		buffer.Write([]byte(tx.SourceAccount))
	}
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func (tx *Transaction) HexSign() string {
	return common.ToHex(tx.Sign)
}


func (tx *Transaction) RecoverSource() error {
	if tx.Source != nil || tx.Type == TransactionTypeBonus {
		return nil
	}
	sign := common.BytesToSign(tx.Sign)
	pk, err := sign.RecoverPubkey(tx.Hash.Bytes())
	if err == nil {
		src := pk.GetAddress()
		tx.Source = &src
	}
	return err
}

//type Transactions []*Transaction
//
//func (c Transactions) Len() int {
//	return len(c)
//}
//func (c Transactions) Swap(i, j int) {
//	c[i], c[j] = c[j], c[i]
//}
//func (c Transactions) Less(i, j int) bool {
//	return c[i].Nonce < c[j].Nonce
//}

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
	ProveValue   []byte      //vrf prove
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
	//EvictedTxs   []common.Hash
}

type header struct {
	Height       uint64        // 本块的高度
	PreHash      common.Hash   //上一块哈希
	PreTime      time.Time     //上一块铸块时间
	ProveValue   []byte      //轮转序号
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
	//EvictedTxs   []common.Hash
}

func (bh *BlockHeader) GenHash() common.Hash {
	buf := bytes.NewBuffer([]byte{})

	buf.Write(utility.UInt64ToByte(bh.Height))

	buf.Write(bh.PreHash.Bytes())

	pt, _ := bh.PreTime.MarshalBinary()
	buf.Write(pt)

	buf.Write(bh.ProveValue)

	buf.Write(utility.UInt64ToByte(bh.TotalQN))

	ct, _ := bh.CurTime.MarshalBinary()
	buf.Write(ct)

	buf.Write(bh.Castor)

	buf.Write(bh.GroupId)

	buf.Write(utility.UInt64ToByte(bh.Nonce))

	if bh.Transactions != nil {
		for _, tx := range bh.Transactions {
			buf.Write(tx.Bytes())
		}
	}
	buf.Write(bh.TxTree.Bytes())
	buf.Write(bh.ReceiptTree.Bytes())
	buf.Write(bh.StateTree.Bytes())
	if bh.ExtraData != nil {
		buf.Write(bh.ExtraData)
	}
	buf.Write(bh.ProveRoot.Bytes())

	return common.BytesToHash(common.Sha256(buf.Bytes()))
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
		//EvictedTxs:   bh.EvictedTxs,
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

func (gh *GroupHeader) DismissedAt(h uint64) bool {
	return gh.DismissHeight <= h
}

func (gh *GroupHeader) WorkAt(h uint64) bool {
	return !gh.DismissedAt(h) && gh.WorkHeight <= h
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

func (g *Group) MemberExist(id []byte) bool {
	for _, mem := range g.Members {
		if bytes.Equal(mem, id) {
			return true
		}
	}
	return false
}

type StateNode struct {
	Key   []byte
	Value []byte
}

