package cli

import (
	"fmt"
	"gtas/rpc"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/12/20 下午1:43
**  Description: 
*/

type WalletServer struct {
	Port int
	aop accountOp
}

func NewWalletServer(port int, aop accountOp) *WalletServer {
	ws := &WalletServer{
		Port: port,
		aop: aop,
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
	} else {
		return err
	}
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


func (ws *WalletServer) SignData(source, target, unlockPassword string, value float64, gas, gasprice uint64, txType int, nonce uint64, data string) (*Result) {
	txRaw := &txRawData{
		Target: target,
		Value: common.Value2RA(value),
		Gas: gas,
		Gasprice: gasprice,
		TxType: txType,
		Nonce: nonce,
		Data: data,
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
	tranx.Sign = &sign
	txRaw.Sign = tranx.Sign.GetHexString()
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
