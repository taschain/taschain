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
	"core"
	"fmt"
	"middleware/types"
	"strings"
	"bytes"
	"yunkuai"
)

// 分红价值统计
var BonusValueStatMap = make(map[uint64]map[string]uint64, 50)

// 分红次数统计
var BonusNumStatMap = make(map[uint64]map[string]uint64, 50)

var CastBlockStatMap = make(map[uint64]map[string]uint64, 50)


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
