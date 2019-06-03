package cli

import (
	"fmt"
	"github.com/taschain/taschain/cmd/gtas/rpc"
	"github.com/taschain/taschain/common"
)

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
	privateKey := common.HexToSecKey(aci.Sk)
	pubkey := common.HexToPubKey(aci.Pk)
	if privateKey.GetPubKey().Hex() != pubkey.Hex() {
		return opError(fmt.Errorf("privatekey or pubkey error"))
	}
	sourceAddr := pubkey.GetAddress()
	if sourceAddr.Hex() != aci.Address {
		return opError(fmt.Errorf("address error"))
	}

	tranx := txRawToTransaction(txRaw)
	tranx.Hash = tranx.GenHash()
	sign := privateKey.Sign(tranx.Hash.Bytes())
	tranx.Sign = sign.Bytes()
	txRaw.Sign = sign.Hex()
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
