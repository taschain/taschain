package contract

import (
	"common"
	tasCore "core"
	"common/abi"
	"vm/core/vm"
	"vm/common/math"
	"governance/util"
	"vm/core"
	"strings"
	"vm/crypto"
	"math/big"
	"errors"
	"middleware/types"
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
	GasPrice uint64          // wei <-> gas exchange ratio
	Value    uint64          // amount of wei sent along with the call
	Data     []byte          // input data, usually an ABI-encoded contract method invocation
}

type CallOpt struct {
	msg    *CallMsg
	method string
	args   []interface{}
}

type CallContext struct {
	bh    *types.BlockHeader
	bc    *tasCore.BlockChain
	state vm.StateDB
}

type BoundContract struct {
	address common.Address
	abi     abi.ABI
	//code    []byte
}
type ResultProvider func() interface{}

func newBoundContract(address common.Address, abi abi.ABI) *BoundContract {
	return &BoundContract{
		address: address,
		abi:     abi,
		//code:    common.Hex2Bytes(codes),
	}
}

func BuildBoundContract(address common.Address, abis string) *BoundContract {
	_abi, _ := abi.JSON(strings.NewReader(abis))
	return newBoundContract(address, _abi)
}

func NewCallOpt(msg *CallMsg, method string, args ...interface{}) *CallOpt {
	return &CallOpt{
		msg:    msg,
		method: method,
		args:   args,
	}
}

func NewDefaultCallMsg(from common.Address, to *common.Address, input []byte) *CallMsg {
	return &CallMsg{
		From:     from,
		To:       to,
		Value:    0,
		Gas:      math.MaxUint64,
		GasPrice: 1,
		Data:     input,
	}
}

func NewSimulateCallMsg(from common.Address, to *common.Address, gas uint64) *CallMsg {
	return &CallMsg{
		From:     from,
		To:       to,
		Value:    0,
		Gas:      gas,
		GasPrice: 1,
	}
}

func NewCallContext(bh *types.BlockHeader, bc *tasCore.BlockChain, db vm.StateDB) *CallContext {
	return &CallContext{
		bh:    bh,
		bc:    bc,
		state: db,
	}
}
func ChainTopCallContext() *CallContext {
	bc := tasCore.BlockChainImpl
	return NewCallContext(bc.QueryTopBlock(), bc, bc.LatestStateDB())
}

func call(ctx *CallContext, msg *CallMsg) ([]byte, *types.Transaction, error) {
	tx := &types.Transaction{
		Source:   &msg.From,
		Target:   msg.To,
		Nonce:    ctx.state.GetNonce(util.ToETHAddress(msg.From)),
		Value:    msg.Value,
		GasLimit: msg.Gas,
		GasPrice: msg.GasPrice,
		Data:     msg.Data,
	}

	gp := new(core.GasPool).AddGas(tx.GasLimit)

	//executor := tasCore.NewEVMExecutor(ctx.blockChain)

	context := tasCore.NewEVMContext(tx, ctx.bh, ctx.bc)
	vmenv := vm.NewEVM(context, ctx.state, tasCore.TestnetChainConfig, vm.Config{})

	ret, _, fail, err := tasCore.NewSession(ctx.state, tx, gp, nil).Run(vmenv)
	if err != nil {
		return nil, nil, err
	}
	if fail {
		return nil, nil, errors.New("vm error")
	}
	return ret, tx, nil
}

func infiniteBalance(db vm.StateDB, account string) common.Address {
	source := common.StringToAddress(account)
	//设置该账户余额
	db.AddBalance(util.ToETHAddress(source), new(big.Int).SetUint64(math.MaxUint64))
	return source
}
func (sc *BoundContract) GetAddress() common.Address {
	return sc.address
}

func (sc *BoundContract) GetAbi() *abi.ABI {
	return &sc.abi
}

func (sc *BoundContract) CallContract(ctx *CallContext, opt *CallOpt, result interface{}) (error) {
	input, err := sc.abi.Pack(opt.method, opt.args...)
	if err != nil {
		return err
	}

	var output []byte

	var msg *CallMsg
	if opt.msg == nil {
		source := infiniteBalance(ctx.state, "_sys_gov_call_")
		msg = NewDefaultCallMsg(source, &sc.address, input)
	} else {
		msg = opt.msg
		msg.Data = input
	}

	output, _, err = call(ctx, msg)
	if err != nil {
		return err
	}
	if len(output) == 0 {
		return nil
	}

	return sc.abi.Unpack(result, opt.method, output)

}

func (sc *BoundContract) NoResultCall(ctx *CallContext, opt *CallOpt) error {
	return sc.CallContract(ctx, opt, 0)
}

func (sc *BoundContract) ResultCall(ctx *CallContext, rp ResultProvider, opt *CallOpt) (interface{}, error) {
	ret := rp()
	if err := sc.CallContract(ctx, opt, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func Deploy(ctx *CallContext, from string, code []byte) (common.Address, []byte, error) {
	source := infiniteBalance(ctx.state, from)
	msg := NewDefaultCallMsg(source, nil, code)
	ret, tx, err := call(ctx, msg)
	if err != nil {
		return common.Address{}, nil, err
	}

	addr := util.ToTASAddress(crypto.CreateAddress(util.ToETHAddress(*tx.Source), tx.Nonce))
	return addr, ret, nil
}

func SimulateDeployContract(ctx *CallContext, from string, abis string, codes string, args ...interface{}) (common.Address, []byte, error) {
	sc := BuildBoundContract(common.Address{}, abis)
	input, err := sc.abi.Pack("", args...)
	if err != nil {
		return common.Address{}, nil, err
	}

	code := common.Hex2Bytes(codes)

	return Deploy(ctx, from, append(code, input...))

}
