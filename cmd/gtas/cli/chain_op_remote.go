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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/base"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/middleware/types"
	"github.com/vmihailenco/msgpack"
)

/*
**  Creator: pxf
**  Date: 2018/12/20 下午2:32
**  Description:
 */

type RemoteChainOpImpl struct {
	host string
	port int
	base string
	aop  accountOp
	show bool
}

func InitRemoteChainOp(ip string, port int, show bool, op accountOp) *RemoteChainOpImpl {
	ca := &RemoteChainOpImpl{
		aop:  op,
		show: show,
	}
	ca.Connect(ip, port)
	return ca
}

func (ca *RemoteChainOpImpl) Connect(ip string, port int) error {
	if ip == "" {
		return nil
	}
	ca.host = ip
	ca.port = port
	ca.base = fmt.Sprintf("http://%v:%v", ip, port)
	return nil
}

func (ca *RemoteChainOpImpl) request(method string, params ...interface{}) *Result {
	if ca.base == "" {
		return opError(ErrUnConnected)
	}

	param := RPCReqObj{
		Method:  "GTAS_" + method,
		Params:  params[:],
		ID:      1,
		Jsonrpc: "2.0",
	}

	if ca.show {
		fmt.Println("Request:")
		bs, _ := json.MarshalIndent(param, "", "\t")
		fmt.Println(string(bs))
		fmt.Println("==================================================================================")
	}

	paramBytes, err := json.Marshal(param)
	if err != nil {
		return opError(err)
	}

	resp, err := http.Post(ca.base, "application/json", bytes.NewReader(paramBytes))
	defer resp.Body.Close()
	if err != nil {
		return opError(err)
	}
	responseBytes, err := ioutil.ReadAll(resp.Body)
	ret := &RPCResObj{}
	if err := json.Unmarshal(responseBytes, ret); err != nil {
		return opError(err)
	}
	if ret.Error != nil {
		return opError(fmt.Errorf(ret.Error.Message))
	}
	return ret.Result
}

func (ca *RemoteChainOpImpl) nonce(addr string) (uint64, error) {
	ret := ca.request("nonce", addr)
	if !ret.IsSuccess() {
		return 0, fmt.Errorf(ret.Message)
	}
	return uint64(ret.Data.(float64)), nil
}

func (ca *RemoteChainOpImpl) Endpoint() string {
	return fmt.Sprintf("%v:%v", ca.host, ca.port)
}

func (ca *RemoteChainOpImpl) SendRaw(tx *txRawData) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	privateKey := common.HexToSecKey(aci.Sk)
	pubkey := common.HexToPubKey(aci.Pk)
	if privateKey.GetPubKey().Hex() != pubkey.Hex() {
		return opError(fmt.Errorf("privatekey or pubkey error"))
	}
	source := pubkey.GetAddress()
	if source.Hex() != aci.Address {
		return opError(fmt.Errorf("address error"))
	}

	nonce, err := ca.nonce(aci.Address)
	if err != nil {
		return opError(err)
	} else {
		tranx := txRawToTransaction(tx)
		tranx.Nonce = nonce + 1
		tx.Nonce = nonce + 1
		tranx.Hash = tranx.GenHash()
		sign := privateKey.Sign(tranx.Hash.Bytes())
		tranx.Sign = sign.Bytes()
		tx.Sign = sign.Hex()
		//fmt.Println("info:", aci.Address, aci.Pk, tx.Sign, tranx.Hash.String())
		//fmt.Printf("%+v\n", tranx)
	}
	jsonByte, err := json.Marshal(tx)
	if err != nil {
		return opError(err)
	}

	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	//此处要签名
	return ca.request("tx", string(jsonByte))
}

func (ca *RemoteChainOpImpl) Balance(addr string) *Result {
	return ca.request("balance", addr)
}

func (ca *RemoteChainOpImpl) MinerInfo(addr string) *Result {
	return ca.request("minerInfo", addr)
}

func (ca *RemoteChainOpImpl) BlockHeight() *Result {
	return ca.request("blockHeight")
}

func (ca *RemoteChainOpImpl) GroupHeight() *Result {
	return ca.request("groupHeight")
}

func (ca *RemoteChainOpImpl) TxInfo(hash string) *Result {
	return ca.request("transDetail", hash)
}

func (ca *RemoteChainOpImpl) BlockByHash(hash string) *Result {
	return ca.request("getBlockByHash", hash)
}

func (ca *RemoteChainOpImpl) BlockByHeight(h uint64) *Result {
	return ca.request("getBlockByHeight", h)
}

func (ca *RemoteChainOpImpl) ApplyMiner(mtype int, stake uint64, gas, gasprice uint64) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	if aci.Miner == nil {
		return opError(fmt.Errorf("the current account is not a miner account"))
	}
	source := common.HexToAddress(aci.Address)
	var bpk groupsig.Pubkey
	bpk.SetHexString(aci.Miner.BPk)

	st := uint64(0)
	if mtype == types.MinerTypeLight && common.TAS2RA(stake) < common.VerifyStake {
		fmt.Println("stake of applying verify node must > 100 TAS")
		return opError(errors.New("stake value error!"))
	} else {
		st = common.TAS2RA(stake)
	}

	miner := &types.Miner{
		ID:           source.Bytes(),
		PublicKey:    bpk.Serialize(),
		VrfPublicKey: base.Hex2VRFPublicKey(aci.Miner.VrfPk),
		Stake:        st,
		Type:         byte(mtype),
	}
	data, err := msgpack.Marshal(miner)
	if err != nil {
		return opError(err)
	}

	tx := &txRawData{
		Gas:      gas,
		Gasprice: gasprice,
		TxType:   types.TransactionTypeMinerApply,
		Data:     common.ToHex(data),
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) AbortMiner(mtype int, gas, gasprice uint64) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	if aci.Miner == nil {
		return opError(fmt.Errorf("the current account is not a miner account"))
	}
	tx := &txRawData{
		Gas:       gas,
		Gasprice:  gasprice,
		TxType:    types.TransactionTypeMinerAbort,
		Data:      string([]byte{byte(mtype)}),
		ExtraData: aci.Address,
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) RefundMiner(mtype int, addrStr string, gas, gasprice uint64) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	data := []byte{}
	data = append(data, byte(mtype))
	if addrStr == "" {
		addrStr = aci.Address
	}
	addr := common.HexToAddress(addrStr)
	data = append(data, addr.Bytes()...)
	tx := &txRawData{
		Gas:       gas,
		Gasprice:  gasprice,
		TxType:    types.TransactionTypeMinerRefund,
		Data:      common.ToHex(data),
		ExtraData: aci.Address,
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) MinerStake(mtype int, addrStr string, stakeValue, gas, gasprice uint64) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	data := []byte{}
	data = append(data, byte(mtype))
	if addrStr == "" {
		addrStr = aci.Address
	}
	stakeValue = common.TAS2RA(stakeValue)
	addr := common.HexToAddress(addrStr)
	data = append(data, addr.Bytes()...)
	data = append(data, common.Uint64ToByte(stakeValue)...)
	tx := &txRawData{
		Gas:       gas,
		Gasprice:  gasprice,
		TxType:    types.TransactionTypeMinerStake,
		Data:      common.ToHex(data),
		ExtraData: aci.Address,
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) MinerCancelStake(mtype int, addrStr string, cancelValue, gas, gasprice uint64) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	data := []byte{}
	data = append(data, byte(mtype))
	if addrStr == "" {
		addrStr = aci.Address
	}
	cancelValue = common.TAS2RA(cancelValue)
	addr := common.HexToAddress(addrStr)
	data = append(data, addr.Bytes()...)
	data = append(data, common.Uint64ToByte(cancelValue)...)
	tx := &txRawData{
		Gas:       gas,
		Gasprice:  gasprice,
		TxType:    types.TransactionTypeMinerCancelStake,
		Data:      common.ToHex(data),
		ExtraData: aci.Address,
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) ViewContract(addr string) *Result {
	return ca.request("explorerAccount", addr)
}

func (ca *RemoteChainOpImpl) TxReceipt(hash string) *Result {
	return ca.request("txReceipt", hash)
}
