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
	"common"
	"consensus/groupsig"
	"math/big"
	"middleware/types"
	"time"
)

// Result rpc请求成功返回的可变参数部分
type Result struct {
	Message string      `json:"message"`
	Status  int         `json:"status"`
	Data    interface{} `json:"data"`
}

func (r *Result) IsSuccess() bool {
	return r.Status == 0
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
	Hash   string `json:"hash"`
	Source string `json:"source"`
	Target string `json:"target"`
	Value  string `json:"value"`
	Height uint64 `json:"height"`
	BlockHash string `json:"block_hash"`
}

type PubKeyInfo struct {
	PubKey string `json:"pub_key"`
	ID     string `json:"id"`
}

type ConnInfo struct {
	Id      string `json:"id"`
	Ip      string `json:"ip"`
	TcpPort string `json:"tcp_port"`
}

type GroupStat struct {
	Dismissed bool  `json:"dismissed"`
	VCount    int32 `json:"v_count"`
}

type ProposerStat struct {
	Stake      uint64  `json:"stake"`
	StakeRatio float64 `json:"stake_ratio"`
	PCount     int32   `json:"p_count"`
}

type CastStat struct {
	Group    map[string]GroupStat    `json:"group"`
	Proposer map[string]ProposerStat `json:"proposer"`
}

type MortGage struct {
	Stake       uint64 `json:"stake"`
	ApplyHeight uint64 `json:"apply_height"`
	AbortHeight uint64 `json:"abort_height"`
	Type        string `json:"type"`
}

func NewMortGageFromMiner(miner *types.Miner) *MortGage {
	t := "重节点"
	if miner.Type == types.MinerTypeLight {
		t = "轻节点"
	}
	mg := &MortGage{
		Stake:       uint64(common.RA2TAS(miner.Stake)),
		ApplyHeight: miner.ApplyHeight,
		AbortHeight: miner.AbortHeight,
		Type:        t,
	}
	return mg
}

type NodeInfo struct {
	ID        string     `json:"id"`
	Balance   float64     `json:"balance"`
	Status    string     `json:"status"`
	WGroupNum int        `json:"w_group_num"`
	AGroupNum int        `json:"a_group_num"`
	NType     string     `json:"n_type"`
	TxPoolNum int        `json:"tx_pool_num"`
	BlockHeight uint64 `json:"block_height"`
	GroupHeight uint64 `json:"group_height"`
	MortGages []MortGage `json:"mort_gages"`
	VrfThreshold float64 `json:"vrf_threshold"`
}

type PageObjects struct {
	Total uint64        `json:"count"`
	Data  []interface{} `json:"data"`
}

type Block struct {
	Height      uint64        `json:"height"`
	Hash        common.Hash   `json:"hash"`
	PreHash     common.Hash   `json:"pre_hash"`
	CurTime     time.Time     `json:"cur_time"`
	PreTime     time.Time     `json:"pre_time"`
	Castor      groupsig.ID   `json:"castor"`
	GroupID     groupsig.ID   `json:"group_id"`
	Prove       string      `json:"prove"`
	TotalQN     uint64        `json:"total_qn"`
	Qn          uint64        `json:"qn"`
	TxNum       uint64        `json:"txs"`
	StateRoot   common.Hash   `json:"state_root"`
	TxRoot      common.Hash   `json:"tx_root"`
	ReceiptRoot common.Hash   `json:"receipt_root"`
	ProveRoot   common.Hash   `json:"prove_root"`
	Random      string        `json:"random"`
}

type BlockDetail struct {
	Block
	GenBonusTx *BonusTransaction `json:"gen_bonus_tx"`
	//Signature groupsig.Signature `json:"signature"`
	//Random 	groupsig.Signature `json:"random"`
	Trans        []Transaction        `json:"trans"`
	BodyBonusTxs []BonusTransaction   `json:"body_bonus_txs"`
	MinerBonus   []*MinerBonusBalance `json:"miner_bonus"`
	PreTotalQN   uint64               `json:"pre_total_qn"`
}

type BlockReceipt struct {
	Receipts        []*types.Receipt `json:"receipts"`
	EvictedReceipts []*types.Receipt `json:"evictedReceipts"`
}

type ExplorerBlockDetail struct {
	BlockDetail
	Receipts        []*types.Receipt `json:"receipts"`
	EvictedReceipts []*types.Receipt `json:"evictedReceipts"`
}

type Group struct {
	Height        uint64      `json:"height"`
	Id            groupsig.ID `json:"id"`
	PreId         groupsig.ID `json:"pre_id"`
	ParentId      groupsig.ID `json:"parent_id"`
	BeginHeight   uint64      `json:"begin_height"`
	DismissHeight uint64      `json:"dismiss_height"`
	Members       []string    `json:"members"`
}

type MinerBonusBalance struct {
	ID            groupsig.ID `json:"id"`
	Proposal      bool        `json:"proposal"`      //是否有提案
	PackBonusTx   int         `json:"pack_bonus_tx"` //打包分红交易个数
	VerifyBlock   int         `json:"verify_block"`  //验证块数
	PreBalance    *big.Int    `json:"pre_balance"`
	CurrBalance   *big.Int    `json:"curr_balance"`
	ExpectBalance *big.Int    `json:"expect_balance"`
	Explain       string      `json:"explain"`
}

type Transaction struct {
	Data   []byte          `json:"data"`
	Value  float64          `json:"value"`
	Nonce  uint64          `json:"nonce"`
	Source *common.Address `json:"source"`
	Target *common.Address `json:"target"`
	Type   int8           `json:"type"`

	GasLimit uint64      `json:"gas_limit"`
	GasPrice uint64      `json:"gas_price"`
	Hash     common.Hash `json:"hash"`

	ExtraData     []byte `json:"extra_data"`
	ExtraDataType int8  `json:"extra_data_type"`
}

type BonusTransaction struct {
	Hash         common.Hash   `json:"hash"`
	BlockHash    common.Hash   `json:"block_hash"`
	GroupID      groupsig.ID   `json:"group_id"`
	TargetIDs    []groupsig.ID `json:"target_ids"`
	Value        uint64        `json:"value"`
	StatusReport string        `json:"status_report"`
	Success      bool          `json:"success"`
}

type Dashboard struct {
	BlockHeight uint64     `json:"block_height"`
	GroupHeight uint64     `json:"group_height"`
	WorkGNum    int        `json:"work_g_num"`
	NodeInfo    *NodeInfo  `json:"node_info"`
	Conns       []ConnInfo `json:"conns"`
}
//
//type BonusInfo struct {
//	BlockHeight uint64      `json:"block_height"`
//	BlockHash   common.Hash `json:"block_hash"`
//	BonusTxHash common.Hash `json:"bonus_tx_hash"`
//	GroupId     string      `json:"group_id"`
//	CasterId    string      `json:"caster_id"`
//	GroupIdW     string      `json:"group_id_w"`
//	CasterIdW    string      `json:"caster_id_W"`
//	MemberIds   []string    `json:"members"`
//	BonusValue  uint64      `json:"bonus_value"`
//}
//
//type BonusStatInfo struct {
//	MemberId        string `json:"member_id"`
//	MemberIdW        string `json:"member_id_w"`
//	BonusNum        uint64 `json:"bonus_num"`
//	TotalBonusValue uint64 `json:"total_bonus_value"`
//}
//
//type CastBlockStatInfo struct {
//	CasterId     string `json:"caster_id"`
//	CasterIdW     string `json:"caster_id_w"`
//	Stake        uint64 `json:"stake"`
//	CastBlockNum uint64 `json:"cast_block_num"`
//}
//
//type CastBlockAndBonusResult struct {
//	BonusInfoAtHeight  BonusInfo           `json:"bonus_info_at_height"`
//	BonusStatInfos     []BonusStatInfo     `json:"bonuses"`
//	CastBlockStatInfos []CastBlockStatInfo `json:"cast_blocks"`
//}

type ExplorerAccount struct {
	Balance   *big.Int `json:"balance"`
	Nonce     uint64   `json:"nonce"`
	Type      uint32   `json:"type"`
	CodeHash  string   `json:"code_hash"`
	Code      string   `json:"code"`
	StateData map[string]interface{}   `json:"state_data"`
}

type ExploreBlockBonus struct {
	ProposalId string `json:"proposal_id"`
	ProposalBonus uint64 `json:"proposal_bonus"`
	VerifierBonus BonusTransaction `json:"verifier_bonus"`
}