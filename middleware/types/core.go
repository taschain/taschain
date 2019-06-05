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
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/time"
	"math/big"
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
	Success                            = 0
	TxErrorCodeBalanceNotEnough        = 1
	TxErrorCodeContractAddressConflict = 2
	TxErrorCodeDeployGasNotEnough      = 3
	TxErrorCodeNoCode                  = 4

	SyntaxError  = 1001
	GasNotEnough = 1002

	SysError                    = 2001
	SysCheckABIError            = 2002
	SysABIJSONError             = 2003
	SysContractCallMaxDeepError = 2004
)

var (
	NoCodeErr            = 4
	NoCodeErrorMsg       = "get code from address %s,but no code!"
	ABIJSONError         = 2003
	ABIJSONErrorMsg      = "abi json format error"
	CallMaxDeepError     = 2004
	CallMaxDeepErrorMsg  = "call max deep cannot more than 8"
	InitContractError    = 2005
	InitContractErrorMsg = "contract init error"
)

var (
	TxErrorBalanceNotEnough   = NewTransactionError(TxErrorCodeBalanceNotEnough, "balance not enough")
	TxErrorDeployGasNotEnough = NewTransactionError(TxErrorCodeDeployGasNotEnough, "gas not enough")
	TxErrorABIJSON            = NewTransactionError(SysABIJSONError, "abi json format error")
)

type TransactionError struct {
	Code    int
	Message string
}

func NewTransactionError(code int, msg string) *TransactionError {
	return &TransactionError{Code: code, Message: msg}
}

const (
	TransactionTypeTransfer         = 0
	TransactionTypeContractCreate   = 1
	TransactionTypeContractCall     = 2
	TransactionTypeBonus            = 3
	TransactionTypeMinerApply       = 4
	TransactionTypeMinerAbort       = 5
	TransactionTypeMinerRefund      = 6
	TransactionTypeMinerCancelStake = 7
	TransactionTypeMinerStake       = 8

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

	ExtraData     []byte          `msgpack:"ed"`
	ExtraDataType int8            `msgpack:"et,omitempty"`
	Sign          []byte          `msgpack:"si"`
	Source        *common.Address `msgpack:"src"` //don't streamlize
}

// GenHash generate hash. source,sign is within the hash calculation range
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
	buffer.WriteByte(byte(tx.Type))
	buffer.Write(common.Uint64ToByte(tx.GasLimit))
	buffer.Write(common.Uint64ToByte(tx.GasPrice))
	if tx.ExtraData != nil {
		buffer.Write(tx.ExtraData)
	}
	buffer.WriteByte(byte(tx.ExtraDataType))

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

func (tx Transaction) GetData() []byte            { return tx.Data }
func (tx Transaction) GetGasLimit() uint64        { return tx.GasLimit }
func (tx Transaction) GetValue() uint64           { return tx.Value }
func (tx Transaction) GetSource() *common.Address { return tx.Source }
func (tx Transaction) GetTarget() *common.Address { return tx.Target }
func (tx Transaction) GetHash() common.Hash       { return tx.Hash }

// PriorityTransactions is a transaction array that determines the priority based on gasprice.
// Gasprice is placed low
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
	GroupID    []byte
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
	ID           []byte
	PublicKey    []byte
	VrfPublicKey []byte
	ApplyHeight  uint64
	Stake        uint64
	AbortHeight  uint64
	Type         byte
	Status       byte
}

// BlockHeader is block head structure
type BlockHeader struct {
	Hash        common.Hash    // The hash of this block
	Height      uint64         // The height of this block
	PreHash     common.Hash    // The hash of previous block
	Elapsed     int32          // The length of time from the last block
	ProveValue  []byte         // Vrf prove
	TotalQN     uint64         // QN of the entire chain
	CurTime     time.TimeStamp // Current block time
	Castor      []byte         // Castor ID
	GroupID     []byte         // Group ID，binary representation of groupsig.ID
	Signature   []byte         // Group signature
	Nonce       int32          // Salt
	TxTree      common.Hash    // Transaction Merkel root hash
	ReceiptTree common.Hash
	StateTree   common.Hash
	ExtraData   []byte
	Random      []byte
}

func (bh *BlockHeader) GenHash() common.Hash {
	buf := bytes.NewBuffer([]byte{})

	buf.Write(common.UInt64ToByte(bh.Height))

	buf.Write(bh.PreHash.Bytes())

	buf.Write(common.Int32ToByte(bh.Elapsed))

	buf.Write(bh.ProveValue)

	buf.Write(common.UInt64ToByte(bh.TotalQN))

	buf.Write(bh.CurTime.Bytes())

	buf.Write(bh.Castor)

	buf.Write(bh.GroupID)

	buf.Write(common.Int32ToByte(bh.Nonce))

	buf.Write(bh.TxTree.Bytes())
	buf.Write(bh.ReceiptTree.Bytes())
	buf.Write(bh.StateTree.Bytes())
	if bh.ExtraData != nil {
		buf.Write(bh.ExtraData)
	}

	return common.BytesToHash(common.Sha256(buf.Bytes()))
}

func (bh *BlockHeader) PreTime() time.TimeStamp {
	return bh.CurTime.Add(int64(-bh.Elapsed))
}

func (bh *BlockHeader) HasTransactions() bool {
	return bh.TxTree != common.EmptyHash
}

type Block struct {
	Header       *BlockHeader
	Transactions []*Transaction
}

func (b *Block) GetTransactionHashs() []common.Hash {
	if b.Transactions == nil {
		return []common.Hash{}
	}
	hashs := make([]common.Hash, 0)
	for _, tx := range b.Transactions {
		hashs = append(hashs, tx.Hash)
	}
	return hashs
}

type Member struct {
	ID     []byte
	PubKey []byte
}

type GroupHeader struct {
	Hash          common.Hash // Group header hash
	Parent        []byte      // Parent group ID, which create the current group
	PreGroup      []byte      // Previous group ID on group chain
	Authority     uint64      // The authority given by the parent group
	Name          string      // The name given by the parent group
	BeginTime     time.TimeStamp
	MemberRoot    common.Hash // Group members list hash
	CreateHeight  uint64      // Height of the group created
	ReadyHeight   uint64      // Latest height of ready
	WorkHeight    uint64      // Height of work
	DismissHeight uint64      // Height of dismiss
	Extends       string      // Extend data
}

func (gh *GroupHeader) GenHash() common.Hash {
	buf := bytes.Buffer{}
	buf.Write(gh.Parent)
	buf.Write(gh.PreGroup)
	buf.Write(common.Uint64ToByte(gh.Authority))
	buf.WriteString(gh.Name)

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
	Header      *GroupHeader
	ID          []byte
	PubKey      []byte
	Signature   []byte
	Members     [][]byte // Member id list
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

type BlockWeight struct {
	TotalQN uint64
	PV      *big.Int
}

type PvFunc func(pvBytes []byte) *big.Int

var DefaultPVFunc PvFunc

func (bw *BlockWeight) MoreWeight(bw2 *BlockWeight) bool {
	return bw.Cmp(bw2) > 0
}

func (bw *BlockWeight) Cmp(bw2 *BlockWeight) int {
	if bw.TotalQN > bw2.TotalQN {
		return 1
	} else if bw.TotalQN < bw2.TotalQN {
		return -1
	}
	return bw.PV.Cmp(bw2.PV)
}

func NewBlockWeight(bh *BlockHeader) *BlockWeight {
	return &BlockWeight{
		TotalQN: bh.TotalQN,
		PV:      DefaultPVFunc(bh.ProveValue),
	}
}

func (bw *BlockWeight) String() string {
	return fmt.Sprintf("%v-%v", bw.TotalQN, bw.PV.Uint64())
}
