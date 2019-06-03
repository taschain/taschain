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
	"fmt"
	"github.com/taschain/taschain/cmd/gtas/rpc"
	"github.com/taschain/taschain/common"
)

/*
**  Creator: pxf
**  Date: 2018/12/20 下午1:43
**  Description:
 */

type WalletServer struct {
	Port int
	aop  accountOp
}

func NewWalletServer(port int, aop accountOp) *WalletServer {
	ws := &WalletServer{
		Port: port,
		aop:  aop,
	}
	return ws
}

func (ws *WalletServer) Start() error {
	if ws.Port <= 0 {
		return fmt.Errorf("please input the rpcport")
	}
	apis := []rpc.API{
		{Namespace: "GTASWallet", Version: "1", Service: ws, Public: true},
	}
	host := fmt.Sprintf("127.0.0.1:%d", ws.Port)
	err := startHTTP(host, apis, []string{}, []string{}, []string{})
	if err == nil {
		fmt.Printf("Wallet RPC serving on http://%s\n", host)
		return nil
	}
	return err
}

//func response(w http.ResponseWriter, ret *Result) {
//	w.Header().Set("content-type", "application/json")
//	fmt.Fprintln(w, ret)
//}
//
//func responseError(w http.ResponseWriter, err string) {
//	ret, _ := failResult(err)
//	response(w, ret)
//}
func (ws *WalletServer) SignData(source, target, unlockPassword string, value float64, gas uint64, gaspriceStr string, txType int, nonce uint64, data string) *Result {
	gp, err := common.ParseCoin(gaspriceStr)
	if err != nil {
		return opError(fmt.Errorf("%v:%v, correct example: 100RA,100kRA,1mRA,1TAS", err, gaspriceStr))
	}
	txRaw := &txRawData{
		Target:   target,
		Value:    common.Value2RA(value),
		Gas:      gas,
		Gasprice: gp,
		TxType:   txType,
		Nonce:    nonce,
		Data:     data,
	}

	r := ws.aop.UnLock(source, unlockPassword)
	if !r.IsSuccess() {
		return r
	}
	r = ws.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)

	//ws.aop.Lock(source)
	privateKey := common.HexStringToSecKey(aci.Sk)
	pubkey := common.HexStringToPubKey(aci.Pk)
	if privateKey.GetPubKey().GetHexString() != pubkey.GetHexString() {
		return opError(fmt.Errorf("privatekey or pubkey error"))
	}
	sourceAddr := pubkey.GetAddress()
	if sourceAddr.GetHexString() != aci.Address {
		return opError(fmt.Errorf("address error"))
	}

	tranx := txRawToTransaction(txRaw)
	tranx.Hash = tranx.GenHash()
	sign := privateKey.Sign(tranx.Hash.Bytes())
	tranx.Sign = sign.Bytes()
	txRaw.Sign = sign.GetHexString()
	//fmt.Println("info:", aci.Address, aci.Pk, tx.Sign, tranx.Hash.String())
	//fmt.Printf("%+v\n", tranx)
	//
	//jsonByte, err := json.MarshalIndent(txRaw, "", "\t")
	//fmt.Println(string(jsonByte))
	//if err != nil {
	//	return opError(err)
	//}
	return opSuccess(txRaw)
}
