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
	"gtas/rpc"
	"net"

	"common"
	"consensus/groupsig"
	"consensus/mediator"
	"core"
	"encoding/hex"
	"fmt"
	"math"
	"middleware/types"
	"network"
	"strconv"
	"strings"
	"encoding/json"
	"bytes"
	"yunkuai"
)

// 分红价值统计
var BonusValueStatMap = make(map[uint64]map[string]uint64, 50)

// 分红次数统计
var BonusNumStatMap = make(map[uint64]map[string]uint64, 50)

var CastBlockStatMap = make(map[uint64]map[string]uint64, 50)

// GtasAPI is a single-method API handler to be returned by test services.
type GtasAPI struct {
}

func (api *GtasAPI) R() *Result {
	height := core.BlockChainImpl.Height()
	core.BlockChainImpl.RemoveTop()
	return &Result{
		Message: fmt.Sprintf("Removed block height: %d", height),
		//Success: false,
	}
}

// 云块交易接口
// 加入版本号的控制
func (api *GtasAPI) L(data string, extradata []byte) *Result {

	// 先判断提交内容是否重复
	result := api.S(data)
	if nil != result && nil != result.Data {
		txDetail := result.Data.(map[string]interface{})
		existed := txDetail["ExtraData"].([]byte)
		if 0 == bytes.Compare(existed, extradata) {
			return &Result{
				Message: fmt.Sprintf("data existed: %s", data),
				//Success: false,
			}
		}
	}

	addr_s := common.BytesToAddress([]byte(yunkuai.Yunkuai_s))
	addr_t := common.BytesToAddress([]byte(yunkuai.Yunkuai_t))
	data = yunkuai.GetYunKuaiProcessor().GenerateNewKey(data)

	tx := &types.Transaction{
		Data:   []byte(data),
		Value:  0,
		Nonce:  0,
		Source: &addr_s,
		Target: &addr_t,

		Type: types.TransactionYunkuai,

		GasLimit: 0,
		GasPrice: 0,

		ExtraData: extradata,
	}
	hash := tx.GenHash()
	tx.Hash = hash

	txpool := core.BlockChainImpl.GetTransactionPool()
	if nil == txpool {
		return &Result{
			Message: fmt.Sprintf("fail to add data, no chain"),
			Data:    hash.String(),
			//Success: false,
		}
	}
	flag, err := txpool.AddTransaction(tx)

	return &Result{
		Message: fmt.Sprintf("Transaction hash: %s, success: %t, error: %s", hash.String(), flag, err),
		Data:    hash.String(),
		//Success: false,
	}
}

// 云块交易查询接口
func (api *GtasAPI) S(index string) *Result {
	version := yunkuai.GetYunKuaiProcessor().GenerateLastestKey(index)
	if !yunkuai.GetYunKuaiProcessor().Contains(version) {
		return &Result{
			Message: fmt.Sprintf("not existed: %s", index),
			//Success: false,
		}
	}

	hash := yunkuai.GetYunKuaiProcessor().Get(version)
	txpool := core.BlockChainImpl.GetTransactionPool()
	tx, _ := txpool.GetTransaction(common.BytesToHash(hash))
	if nil == tx {
		return &Result{
			Message: fmt.Sprintf("not existed: %s", version),
			//Success: false,
		}
	}

	txDetail := make(map[string]interface{})
	txDetail["Data"] = tx.Data
	txDetail["ExtraData"] = tx.ExtraData
	txDetail["ExtraDataType"] = tx.ExtraDataType
	return &Result{
		Message: fmt.Sprintf("existed: %s", version),
		Data:    txDetail,
		//Success: true,
	}
}

// T 用户交易接口
func (api *GtasAPI) Tx(txRawjson string) (*Result, error) {
	var txRaw = new(txRawData)
	if err := json.Unmarshal([]byte(txRawjson), txRaw); err != nil {
		return failResult(err.Error())
	}

	trans := txRawToTransaction(txRaw)

	trans.Hash = trans.GenHash()

	if err := sendTransaction(trans); err != nil {
		return failResult(err.Error())
	}

	return successResult(trans.Hash.String())

}

//脚本交易
func (api *GtasAPI) ScriptTransferTx(privateKey string, from string, to string, amount uint64, nonce uint64, txType int, gasPrice uint64) (*Result, error) {
	var result *Result
	var err error
	var i uint64 = 0
	for ; i < 100; i++ {
		result, err = api.TxUnSafe(privateKey, to, amount, gasPrice, gasPrice, nonce+i, txType, "")
	}
	return result, err
}

// ExplorerAccount
func (api *GtasAPI) ExplorerAccount(account string) (*Result, error) {
	balance := core.BlockChainImpl.GetBalance(common.HexToAddress(account))
	nonce := core.BlockChainImpl.GetNonce(common.HexToAddress(account))

	return successResult(ExplorerAccount{Balance: balance, Nonce: nonce})

}

// Balance 查询余额接口
func (api *GtasAPI) Balance(account string) (*Result, error) {
	balance, err := walletManager.getBalance(account)
	if err != nil {
		return nil, err
	}
	return &Result{
		Message: fmt.Sprintf("The balance of account: %s is %d", account, balance),
		Data:    fmt.Sprintf("%d", balance),
	}, nil
}

// NewWallet 新建账户接口
func (api *GtasAPI) NewWallet() (*Result, error) {
	privKey, addr := walletManager.newWallet()
	data := make(map[string]string)
	data["private_key"] = privKey
	data["address"] = addr
	return successResult(data)
}

// GetWallets 获取当前节点的wallets
func (api *GtasAPI) GetWallets() (*Result, error) {
	return successResult(walletManager)
}

// DeleteWallet 删除本地节点指定序号的地址
func (api *GtasAPI) DeleteWallet(key string) (*Result, error) {
	walletManager.deleteWallet(key)
	return successResult(walletManager)
}

// BlockHeight 块高查询
func (api *GtasAPI) BlockHeight() (*Result, error) {
	height := core.BlockChainImpl.QueryTopBlock().Height
	return successResult(height)
}

// GroupHeight 组块高查询
func (api *GtasAPI) GroupHeight() (*Result, error) {
	height := core.GroupChainImpl.Count()
	return successResult(height)
}

// Vote
func (api *GtasAPI) Vote(from string, v *VoteConfig) (*Result, error) {
	//config := v.ToGlobal()
	//walletManager.newVote(from, config)
	return successResult(nil)
}

// ConnectedNodes 查询已链接的node的信息
func (api *GtasAPI) ConnectedNodes() (*Result, error) {

	nodes := network.GetNetInstance().ConnInfo()
	conns := make([]ConnInfo, 0)
	for _, n := range nodes {
		conns = append(conns, ConnInfo{Id: n.Id, Ip: n.Ip, TcpPort: n.Port})
	}
	return successResult(conns)
}

// TransPool 查询缓冲区的交易信息。
func (api *GtasAPI) TransPool() (*Result, error) {
	transactions := core.BlockChainImpl.GetTransactionPool().GetReceived()
	transList := make([]Transactions, 0, len(transactions))
	for _, v := range transactions {
		transList = append(transList, Transactions{
			Hash:   v.Hash.String(),
			Source: v.Source.GetHexString(),
			Target: v.Target.GetHexString(),
			Value:  strconv.FormatInt(int64(v.Value), 10),
		})
	}

	return successResult(transList)
}

func (api *GtasAPI) GetTransaction(hash string) (*Result, error) {
	transaction, err := core.BlockChainImpl.GetTransactionByHash(common.HexToHash(hash))
	if err != nil {
		return nil, err
	}
	detail := make(map[string]interface{})
	detail["hash"] = hash
	detail["source"] = transaction.Source.Hash().Hex()
	detail["target"] = transaction.Target.Hash().Hex()
	detail["value"] = transaction.Value
	return successResult(detail)
}

//
//var CastBlockStatMap = make(map[uint64]map[string]uint64, 50)



// startHTTP initializes and starts the HTTP RPC endpoint.
func startHTTP(endpoint string, apis []rpc.API, modules []string, cors []string, vhosts []string) error {
	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
			if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
				return err
			}
		}
	}
	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return err
	}
	go rpc.NewHTTPServer(cors, vhosts, handler).Serve(listener)
	//go rpc.NewWSServer(cors, handler).Serve(listener)
	return nil
}

var GtasAPIImpl *GtasAPI

// StartRPC RPC 功能
func StartRPC(host string, port uint) error {
	var err error
	GtasAPIImpl = &GtasAPI{}
	apis := []rpc.API{
		{Namespace: "GTAS", Version: "1", Service: GtasAPIImpl, Public: true},
	}
	for plus := 0; plus < 40; plus++ {
		err = startHTTP(fmt.Sprintf("%s:%d", host, port+uint(plus)), apis, []string{}, []string{}, []string{})
		if err == nil {
			common.DefaultLogger.Infof("RPC serving on http://%s:%d\n", host, port+uint(plus))
			return nil
		}
		if strings.Contains(err.Error(), "address already in use") {
			common.DefaultLogger.Infof("address: %s:%d already in use\n", host, port+uint(plus))
			continue
		}
		return err
	}
	return err
}
