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