package contract

import (
	"common"
	"math/big"
)

/*
**  Creator: pxf
**  Date: 2018/4/9 下午1:47
**  Description: 
*/
type CallMsg struct {
	From     common.Address  // the sender of the 'transaction'
	To       *common.Address // the destination contract (nil for contract creation)
	Gas      uint64          // if 0, the call executes with near-infinite gas
	GasPrice *big.Int        // wei <-> gas exchange ratio
	Value    *big.Int        // amount of wei sent along with the call
	Data     []byte          // input data, usually an ABI-encoded contract method invocation
}


type CallerContext struct {
	address common.Address
	msg *CallMsg
	method string
	args []interface{}
}

type ContractCaller interface {

	CallContract(ctx *CallerContext) (interface{}, error)

	decodeResult(result []byte) (interface{}, error)
}

type SimpleContractCaller struct {


}

func (*SimpleContractCaller) CallContract(ctx *CallerContext) (interface{}, error) {
	panic("implement me")
}

func (*SimpleContractCaller) decodeResult(result []byte) (interface{}, error) {
	panic("implement me")
}




