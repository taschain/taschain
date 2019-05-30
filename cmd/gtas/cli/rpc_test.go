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
	"encoding/json"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/core"
	"github.com/taschain/taschain/taslog"
	"log"
	"testing"
)

func TestRPC(t *testing.T) {
	gtas := NewGtas()
	gtas.simpleInit("tas.ini")
	common.DefaultLogger = taslog.GetLoggerByIndex(taslog.DefaultConfig, common.GlobalConf.GetString("instance", "index", ""))
	err := gtas.fullInit(true,true,"",0,"127.0.0.1","super",false,"testkey",false,100)
	if err != nil {
		t.Error(err)
	}
	host := "127.0.0.1"
	senderAddr := common.HexStringToAddress("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103")
	nonce := core.BlockChainImpl.GetNonce(senderAddr)
	privateKey := common.HexStringToSecKey("0x045c8153e5a849eef465244c0f6f40a43feaaa6855495b62a400cc78f9a6d61c76c09c3aaef393aa54bd2adc5633426e9645dfc36723a75af485c5f5c9f2c94658562fcdfb24e943cf257e25b9575216c6647c4e75e264507d2d57b3c8bc00b361")

	tx := &txRawData{Target:"0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb",Value:10,Gas:1000,Gasprice:10000,TxType:0,Nonce:nonce}
	tranx := txRawToTransaction(tx)
	tranx.Hash = tranx.GenHash()
	sign := privateKey.Sign(tranx.Hash.Bytes())
	tranx.Sign = sign.Bytes()
	tx.Sign = sign.GetHexString()

	txdata,err:= json.Marshal(tx)
	if err != nil{
		t.Error(err)
	}
	var port uint = 8080
	StartRPC(host, port)
	tests := []struct {
		method string
		params []interface{}
	}{
		{"GTAS_newWallet", nil},
		{"GTAS_tx", []interface{}{string(txdata)}},
		{"GTAS_balance", []interface{}{"0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"GTAS_blockHeight", nil},
		{"GTAS_getWallets", nil},
		//{},
	}
	for _, test := range tests {
		res, err := rpcPost(host, port, test.method, test.params...)
		if err != nil {
			t.Errorf("%s failed: %v", test.method, err)
			continue
		}
		if res.Error != nil {
			t.Errorf("%s failed: %v", test.method, res.Error.Message)
			continue
		}
		data, _ := json.Marshal(res.Result.Data)
		log.Printf("%s response data: %s", test.method, data)
	}
}
