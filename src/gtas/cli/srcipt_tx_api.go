package cli

import (
	"common"
	"fmt"
)

//脚本交易
//批量打转账交易  不能对外暴露
//func (api *GtasAPI) ScriptTransferTx(privateKey string, from string, to string, amount uint64, nonce uint64, txType int, gasPrice uint64) (*Result, error) {
//	var result *Result
//	var err error
//	var i uint64 = 0
//	for ; i < 100; i++ {
//		result, err = api.TxUnSafe(privateKey, to, amount, gasPrice, gasPrice, nonce+i, txType, "")
//	}
//	return result, err
//}

func (api *GtasAPI) ScriptTransferTx(privateKey string, from string, to string, amount uint64, nonce uint64, txType int, gasPrice uint64) (*Result, error) {
	return api.TxUnSafe(privateKey, to, amount, gasPrice, gasPrice, nonce, txType, "")
}

func (api *GtasAPI) TxUnSafe(privateKey, target string, value, gas, gasprice, nonce uint64, txType int, data string) (*Result, error) {
	txRaw := &txRawData{
		Target:   target,
		Value:    common.TAS2RA(value),
		Gas:      gas,
		Gasprice: gasprice,
		Nonce:    nonce,
		TxType:   txType,
		Data:     data,
	}
	sk := common.HexStringToSecKey(privateKey)
	if sk == nil {
		return failResult(fmt.Sprintf("parse private key fail:%v", privateKey))
	}
	trans := txRawToTransaction(txRaw)
	trans.Hash = trans.GenHash()
	sign := sk.Sign(trans.Hash.Bytes())
	trans.Sign = &sign

	if err := sendTransaction(trans); err != nil {
		return failResult(err.Error())
	}
	return successResult(trans.Hash.String())
}
