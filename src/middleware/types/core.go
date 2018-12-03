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
)

const (
	TxErrorCode_BalanceNotEnough = 1
	TxErrorCode_ContractAddressConflict = 2
	TxErrorCode_DeployGasNotEnough = 3
	TxErrorCode_NO_CODE = 4

	Syntax_Error=1001
	GasNotEnough = 1002

	Sys_Error = 2001
	Sys_Check_Abi_Error = 2002
	Sys_Abi_JSON_Error = 2003
)

var (
	TxErrorBalanceNotEnough = NewTransactionError(TxErrorCode_BalanceNotEnough,"balance not enough")
	TxErrorDeployGasNotEnough = NewTransactionError(TxErrorCode_DeployGasNotEnough,"gas not enough")
	TxErrorNoCode = NewTransactionError(TxErrorCode_NO_CODE,"no code")
	TxErrorAbiJson = NewTransactionError(Sys_Abi_JSON_Error,"abi json format error")
)

type Transaction struct {
	Data   []byte
	Value  uint64
	Nonce  uint64
	Source *common.Address
	Target *common.Address

	GasLimit uint64
	GasPrice uint64
	Hash     common.Hash

	ExtraData     []byte
	ExtraDataType int32
}

type TransactionError struct {
	Code int
	Message string
}

func NewTransactionError(code int,msg string) *TransactionError {
	return &TransactionError{Code:code, Message:msg}
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
	return pt[i].GasPrice < pt[j].GasPrice
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

//区块头结构
type BlockHeader struct {
	Hash         common.Hash   // 本块的hash，to do : 是对哪些数据的哈希
	Height       uint64        // 本块的高度
	PreHash      common.Hash   //上一块哈希
	PreTime      time.Time     //上一块铸块时间
	QueueNumber  uint64        //轮转序号
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
	EvictedTxs   []common.Hash
	ExtraData    []byte
	Random       []byte
}

type header struct {
	Height       uint64        // 本块的高度
	PreHash      common.Hash   //上一块哈希
	PreTime      time.Time     //上一块铸块时间
	QueueNumber  uint64        //轮转序号
	TotalQN      uint64        //整条链的QN
	CurTime      time.Time     //当前铸块时间
	Castor       []byte        //出块人ID
	GroupId      []byte        //组ID，groupsig.ID的二进制表示
	Nonce        uint64        //盐
	Transactions []common.Hash // 交易集哈希列表
	TxTree       common.Hash   // 交易默克尔树根hash
	ReceiptTree  common.Hash
	StateTree    common.Hash
	EvictedTxs   []common.Hash
	ExtraData    []byte
}

func (bh *BlockHeader) GenHash() common.Hash {
	header := &header{
		Height:       bh.Height,
		PreHash:      bh.PreHash,
		PreTime:      bh.PreTime,
		QueueNumber:  bh.QueueNumber,
		TotalQN:      bh.TotalQN,
		CurTime:      bh.CurTime,
		Castor:       bh.Castor,
		GroupId:      bh.GroupId,
		Nonce:        bh.Nonce,
		Transactions: bh.Transactions,
		TxTree:       bh.TxTree,
		ReceiptTree:  bh.ReceiptTree,
		StateTree:    bh.StateTree,
		EvictedTxs:   bh.EvictedTxs,
		ExtraData:    bh.ExtraData,
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
}

const genesis = `{"Id":null,"Members":[{"Id":"MnlyUmYydHZUelMxdDFVRWJzNThSWTM3cjVGTDlDbnoxZHFEeFRUckE3ZlBtUQAA","PubKey":"137/lYJlU7eAAMLc2P3g7lIUYL+sxnmTqLz7t1v1iWeB9cNkVHErFcda4LHC6wkH/v2XO+bDmVXcBwSU4Cv2YpOiYcb29lTT7mVzKcWQKBo3oInwR3KnYC6BUdOZPiOC"},{"Id":"NWlQbkNVUmR6VTNZczNSV2pZb01KZlhKU2pXazJ0TkxCa0o4ZzZLYWhjN1RtUQAA","PubKey":"vKvF2jTHXd8A0zDlk6/dvmsIWlXrOo/EREDdel3fztqnwfDB9PqbF7sjjtWbWJAgN7qvHe8//XV6LTkWS/n0ljd5jd/pwL3g8tFed221/5bBwOXFUo8Nk3wNPENzaCSD"},{"Id":"OU03dllYTE5HY0ZtRFRiM3A1cXJYZ0xuQXZ4Y3I1ZExOM3Q3dEJRbU5HQmNtUQAA","PubKey":"j/zj1tTCwNcOoJmc7Y4ngjeRoKmFBV1woJ+eHarveStJuGf8wFBeY/gCPLYjmcUMUXKdO1eNmfe2C9mzldGt34CZlGhZJ7sJvVhZZceZ8lrYEL0jMEojfDZkk65NgtEb"},{"Id":"d2s0RUNvb0RDMlA0UFRaaVI1OTJOVFRyeUZCem5UcEY3NnNyUng0eEE1b05tUQAA","PubKey":"5/HH7Kub2FRBXVuz6qLn1rT0sBfijzEbuFMZKXCANSBZLDMe5KVjph6AhwFRZZwgusqDux6vDU7wGvpNon5Cw6Z+Pcbj5vC0L+fseGUd3dxXo2Ya9KtpB4WX3FC+otMg"},{"Id":"TTFtR3E2emUyV2UyNVgzS1B3VTNrakFGRTlqMXFwN25GcDZkZ0toNlBVa1FtUQAA","PubKey":"owYJK49AD1uPS4FmMu0ZJT168gmgKcAJPQ11+/fMoM9sdNm8GUylBe8GvlU5TrgIiVFAmJuPDusMrJkxHUUbB18qfAtgPBNhYbK/6/4efym7vpFw+EW1XT84ELCj50MS"}],"PubKey":"1Mp/guNJtXd4o4jVG0tJhrkSthHrJ3RNWV6ksP8yo6a+xph/ugocGbQ+oUGj2qwQQByqzSYIyWFbEJ2tq9BpIMrzQJpWfTz9F68XlPeM1Pjm6y3J3TPIjeBvwCLbxj4Q","Parent":"AAAA","Dummy":"Z2VuZXNpcyBncm91cCBkdW1teQ==","Signature":null}`

func GenesisGroup() *Group {
	var group *Group
	json.Unmarshal([]byte(genesis), &group) //never mistake
	return group
}
