package cli

import (
	"fmt"
	"middleware/types"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/12/20 下午2:38
**  Description: 
*/

var (
	ErrPassword = fmt.Errorf("password error")
	ErrUnlocked = fmt.Errorf("please unlock the account first")
	ErrUnConnected = fmt.Errorf("please connect to one node first")
)

type txRawData struct {
	//from string
	Target   string `json:"target"`
	Value    uint64 `json:"value"`
	Gas      uint64 `json:"gas"`
	Gasprice uint64 `json:"gasprice"`
	TxType   int `json:"tx_type"`
	Nonce    uint64 `json:"nonce"`
	Data     string `json:"data"`
	Sign     string `json:"sign"`
	ExtraData string `json:"extra_data"`
}

func opError(err error) *Result {
	ret, _ := failResult(err.Error())
	return ret
}

func opSuccess(data interface{}) *Result {
	ret, _ := successResult(data)
	return ret
}

type MinerInfo struct {
	PK          string
	VrfPK 		string
	ID          string
	Stake       uint64
	NType  		byte
	ApplyHeight uint64
	AbortHeight uint64
}

func txRawToTransaction(tx *txRawData) *types.Transaction {
	var target *common.Address
	if tx.Target != "" {
		t := common.HexToAddress(tx.Target)
		target = &t
	}
	var sign []byte
	if tx.Sign != "" {
		sign = common.HexStringToSign(tx.Sign).Bytes()
	} else {

	}

	return &types.Transaction{
		Data: []byte(tx.Data),
		Value: tx.Value,
		Nonce: tx.Nonce,
		//Source: &source,
		Target: target,
		Type: int8(tx.TxType),
		GasLimit: tx.Gas,
		GasPrice: tx.Gasprice,
		Sign: sign,
		ExtraData: []byte(tx.ExtraData),
	}
}

type accountOp interface {

	NewAccount(password string, miner bool) *Result

	AccountList() *Result

	Lock(addr string) *Result

	UnLock(addr string, password string) *Result

	AccountInfo() *Result

	DeleteAccount() *Result

	Close()
}

type chainOp interface {

	Connect(ip string, port int) error

	Endpoint() string

	SendRaw(tx *txRawData) *Result

	Balance(addr string) *Result

	MinerInfo(addr string) *Result

	BlockHeight() *Result

	GroupHeight() *Result

	ApplyMiner(mtype int, stake uint64, gas, gasprice uint64) *Result

	AbortMiner(mtype int, gas, gasprice uint64) *Result

	RefundMiner(mtype int, gas, gasprice uint64) *Result

	TxInfo(hash string) *Result

	BlockByHash(hash string) *Result

	BlockByHeight(h uint64) *Result

	ViewContract(addr string) *Result

	TxReceipt(hash string) *Result
}