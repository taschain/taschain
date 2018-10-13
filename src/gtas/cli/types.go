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

package cli

import (
	"middleware/types"
	"common"
	"time"
	"consensus/groupsig"
	"math/big"
)

// Result rpc请求成功返回的可变参数部分
type Result struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// ErrorResult rpc请求错误返回的可变参数部分
type ErrorResult struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// RPCReqObj 完整的rpc请求体
type RPCReqObj struct {
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Jsonrpc string        `json:"jsonrpc"`
	ID      uint          `json:"id"`
}

// RPCResObj 完整的rpc返回体
type RPCResObj struct {
	Jsonrpc string       `json:"jsonrpc"`
	ID      uint         `json:"id"`
	Result  *Result      `json:"result,omitempty"`
	Error   *ErrorResult `json:"error,omitempty"`
}

// 缓冲池交易列表中的transactions
type Transactions struct {
	Hash string `json:"hash"`
	Source string `json:"source"`
	Target string `json:"target"`
	Value  string `json:"value"`
}

type PubKeyInfo struct {
	PubKey string `json:"pub_key"`
	ID string `json:"id"`
}

type ConnInfo struct {
	Id      string `json:"id"`
	Ip      string `json:"ip"`
	TcpPort string `json:"tcp_port"`
}

type GroupStat struct {
	Dismissed bool `json:"dismissed"`
	VCount		int32 `json:"v_count"`
}

type ProposerStat struct {
	Stake 	uint64 `json:"stake"`
	StakeRatio float64 `json:"stake_ratio"`
	PCount	int32 `json:"p_count"`
}

type CastStat struct {
	Group map[string]GroupStat `json:"group"`
	Proposer map[string]ProposerStat `json:"proposer"`
}

type MortGage struct {
	Stake uint64 `json:"stake"`
	ApplyHeight uint64 `json:"apply_height"`
	AbortHeight uint64 `json:"abort_height"`
	Type string `json:"type"`
}

func NewMortGageFromMiner(miner *types.Miner) *MortGage {
	t := "重节点"
	if miner.Type == types.MinerTypeLight {
		t = "轻节点"
	}
	mg := &MortGage{
		Stake: miner.Stake,
		ApplyHeight: miner.ApplyHeight,
		AbortHeight: miner.AbortHeight,
		Type: t,
	}
	return mg
}

type NodeInfo struct {
	ID string `json:"id"`
	Balance uint64 `json:"balance"`
	Status string `json:"status"`
	WGroupNum int `json:"w_group_num"`
	AGroupNum int `json:"a_group_num"`
	NType string `json:"n_type"`
	TxPoolNum int `json:"tx_pool_num"`
	MortGages []MortGage `json:"mort_gages"`
}

type PageObjects struct {
	Total uint64 `json:"count"`
	Data []interface{} `json:"data"`
}

type Block struct {
	Height uint64 `json:"height"`
	Hash common.Hash `json:"hash"`
	PreHash common.Hash `json:"pre_hash"`
	CurTime time.Time `json:"cur_time"`
	PreTime time.Time `json:"pre_time"`
	Castor groupsig.ID `json:"castor"`
	GroupID groupsig.ID `json:"group_id"`
	Prove  *big.Int `json:"prove"`
	Txs 	[]common.Hash `json:"txs"`
}

type BlockDetail struct {
	Block
	TxCnt 	int `json:"tx_cnt"`
	BonusHash common.Hash `json:"bonus_hash"`
	Signature groupsig.Signature `json:"signature"`
	Random 	groupsig.Signature `json:"random"`
}

type Group struct {
	Height uint64 `json:"height"`
	Id groupsig.ID `json:"id"`
	PreId groupsig.ID `json:"pre_id"`
	ParentId groupsig.ID `json:"parent_id"`
	BeginHeight uint64 `json:"begin_height"`
	DismissHeight uint64 `json:"dismiss_height"`
	Members []string `json:"members"`
}


type Transaction struct {
	Data   []byte `json:"data"`
	Value  uint64 `json:"value"`
	Nonce  uint64 `json:"nonce"`
	Source *common.Address `json:"source"`
	Target *common.Address `json:"target"`
	Type   int32 `json:"type"`

	GasLimit uint64 `json:"gas_limit"`
	GasPrice uint64 `json:"gas_price"`
	Hash     common.Hash `json:"hash"`

	ExtraData     []byte `json:"extra_data"`
	ExtraDataType int32 `json:"extra_data_type"`
}

type Dashboard struct {
	BlockHeight uint64 `json:"block_height"`
	GroupHeight uint64 `json:"group_height"`
	WorkGNum int `json:"work_g_num"`
	NodeInfo *NodeInfo `json:"node_info"`
	Conns []ConnInfo `json:"conns"`
}