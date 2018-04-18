package contract

import (
	"common"
	"math/big"
	tasCore "core"
	"common/abi"
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
	msg *CallMsg
	method string
	args []interface{}
	blockChain *tasCore.BlockChain
}

type ContractCaller interface {

	CallContract(ctx *CallerContext, result interface{}) (error)

}

type BoundContract struct {
	address common.Address
	abi	abi.ABI
}

func NewCallContext(method string, args ...interface{}) *CallerContext {
	return nil
}

//TODO: 状态转换接口,即执行交易TODO:
func (sc *BoundContract) call(ctx *CallerContext) ([]byte, error) {
	return nil, nil
}

func (sc *BoundContract) CallContract(ctx *CallerContext, result interface{}) (error) {
	input, err := sc.abi.Pack(ctx.method, ctx.args...)
	if err != nil {
		return err
	}

	ctx.msg.Data = input

	var output []byte
	output, err = sc.call(ctx)
	if err != nil {
		return err
	}
	if len(output) == 0 {
		return nil
	}

	return sc.abi.Unpack(result, ctx.method, output)
	// Ensure message is initialized properly.
	//call := ctx.msg
	//if call.GasPrice == nil {
	//	call.GasPrice = big.NewInt(1)
	//}
	//if call.Gas == 0 {
	//	call.Gas = 50000000
	//}
	//if call.Value == nil {
	//	call.Value = new(big.Int)
	//}

	// Set infinite balance to the fake caller account.
	//from := statedb.GetOrNewStateObject(call.From)
	//from.SetBalance(math.MaxBig256)
	//// Execute the call.
	//msg := callmsg{call}
	//
	//evmContext := core.NewEVMContext(msg, block.Header(), b.blockchain, nil)
	//// Create a new environment which holds all relevant information
	//// about the transaction and calling mechanisms.
	//vmenv := vm.NewEVM(evmContext, statedb, b.config, vm.Config{})
	//gaspool := new(core.GasPool).AddGas(math.MaxUint64)
	//
	//return core.NewStateTransition(vmenv, msg, gaspool).TransitionDb()
}





