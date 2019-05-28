package cli

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/httplib"
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
	req := httplib.Post(ca.base)

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

	req, err := req.JSONBody(param)
	if err != nil {
		return opError(err)
	}
	ret := &RPCResObj{}
	err = req.ToJSON(ret)

	if err != nil {
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
	privateKey := common.HexStringToSecKey(aci.Sk)
	pubkey := common.HexStringToPubKey(aci.Pk)
	if privateKey.GetPubKey().GetHexString() != pubkey.GetHexString() {
		return opError(fmt.Errorf("privatekey or pubkey error"))
	}
	source := pubkey.GetAddress()
	if source.GetHexString() != aci.Address {
		return opError(fmt.Errorf("address error"))
	}

	if nonce, err := ca.nonce(aci.Address); err != nil {
		return opError(err)
	} else {
		tranx := txRawToTransaction(tx)
		tranx.Nonce = nonce + 1
		tx.Nonce = nonce + 1
		tranx.Hash = tranx.GenHash()
		sign := privateKey.Sign(tranx.Hash.Bytes())
		tranx.Sign = sign.Bytes()
		tx.Sign = sign.GetHexString()
		//fmt.Println("info:", aci.Address, aci.Pk, tx.Sign, tranx.Hash.String())
		//fmt.Printf("%+v\n", tranx)

		jsonByte, err := json.Marshal(tx)
		if err != nil {
			return opError(err)
		}

		ca.aop.(*AccountManager).resetExpireTime(aci.Address)
		//此处要签名
		return ca.request("tx", string(jsonByte))

	}

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
	if mtype == types.MinerTypeLight {
		fmt.Println("stake of applying verify node is hardened as 100 Tas")
		st = common.VerifyStake
	} else {
		st = common.TAS2RA(stake)
	}

	miner := &types.Miner{
		Id:           source.Bytes(),
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
		Gas:      gas,
		Gasprice: gasprice,
		TxType:   types.TransactionTypeMinerRefund,
		Data:     common.ToHex(data),
		ExtraData: aci.Address,
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) MinerStake(mtype int, addrStr string, refundValue, gas, gasprice uint64) *Result {
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
	refundValue = common.TAS2RA(refundValue)
	addr := common.HexToAddress(addrStr)
	data = append(data, addr.Bytes()...)
	data = append(data, common.Uint64ToByte(refundValue)...)
	tx := &txRawData{
		Gas:      gas,
		Gasprice: gasprice,
		TxType:   types.TransactionTypeMinerStake,
		Data:     common.ToHex(data),
		ExtraData: aci.Address,
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) MinerCancelStake(mtype int, addrStr string, refundValue, gas, gasprice uint64) *Result {
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
	refundValue = common.TAS2RA(refundValue)
	addr := common.HexToAddress(addrStr)
	data = append(data, addr.Bytes()...)
	data = append(data, common.Uint64ToByte(refundValue)...)
	tx := &txRawData{
		Gas:      gas,
		Gasprice: gasprice,
		TxType:   types.TransactionTypeMinerCancelStake,
		Data:     common.ToHex(data),
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
