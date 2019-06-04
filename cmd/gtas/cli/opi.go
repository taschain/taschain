package cli

import (
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/types"
)

var (
	ErrPassword    = fmt.Errorf("password error")
	ErrUnlocked    = fmt.Errorf("please unlock the account first")
	ErrUnConnected = fmt.Errorf("please connect to one node first")
)

type txRawData struct {
	Target    string `json:"target"`
	Value     uint64 `json:"value"`
	Gas       uint64 `json:"gas"`
	Gasprice  uint64 `json:"gasprice"`
	TxType    int    `json:"tx_type"`
	Nonce     uint64 `json:"nonce"`
	Data      string `json:"data"`
	Sign      string `json:"sign"`
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
	VrfPK       string
	ID          string
	Stake       uint64
	NType       byte
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
		sign = common.HexToSign(tx.Sign).Bytes()
	} else {

	}

	return &types.Transaction{
		Data:      []byte(tx.Data),
		Value:     tx.Value,
		Nonce:     tx.Nonce,
		Target:    target,
		Type:      int8(tx.TxType),
		GasLimit:  tx.Gas,
		GasPrice:  tx.Gasprice,
		Sign:      sign,
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
	// Connect connect node by ip and port
	Connect(ip string, port int) error
	// Endpoint returns current connected ip and port
	Endpoint() string
	// SendRaw send transaction to connected node
	SendRaw(tx *txRawData) *Result
	// Balance query Balance by address
	Balance(addr string) *Result
	// MinerInfo query miner info by address
	MinerInfo(addr string) *Result

	BlockHeight() *Result

	GroupHeight() *Result

	ApplyMiner(mtype int, stake uint64, gas, gasprice uint64) *Result

	AbortMiner(mtype int, gas, gasprice uint64) *Result

	RefundMiner(mtype int, addrStr string, gas, gasprice uint64) *Result

	MinerStake(mtype int, addrStr string, refundValue, gas, gasprice uint64) *Result

	MinerCancelStake(mtype int, addrStr string, refundValue, gas, gasprice uint64) *Result

	TxInfo(hash string) *Result

	BlockByHash(hash string) *Result

	BlockByHeight(h uint64) *Result

	ViewContract(addr string) *Result

	TxReceipt(hash string) *Result
}
