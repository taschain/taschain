package contract

import (
	"testing"
	"core"
	"governance/util"
	"math/big"
	"common"
	"math"
)

/*
**  Creator: pxf
**  Date: 2018/4/18 下午4:13
**  Description: 
*/

const (
	ABI = `[{"constant":false,"inputs":[{"name":"ac","type":"address"},{"name":"value","type":"uint64"}],"name":"canDeposit","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"creditAddr","type":"address"},{"name":"ac","type":"address"}],"name":"test1","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"voters","outputs":[{"name":"voted","type":"bool"},{"name":"delegate","type":"address"},{"name":"approval","type":"bool"},{"name":"voteBlock","type":"uint64"},{"name":"deposit","type":"uint64"},{"name":"depositBlock","type":"uint64"}],"payable":false,"stateMutability":"view","type":"function"}]`
	CODE = `6060604052341561000f57600080fd5b6104688061001e6000396000f300606060405260043610610057576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806339e2e80b1461005c57806363f76a6a146100c0578063a3ec138d1461012c575b600080fd5b341561006757600080fd5b6100a6600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803567ffffffffffffffff1690602001909190505061020c565b604051808215151515815260200191505060405180910390f35b34156100cb57600080fd5b610116600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803573ffffffffffffffffffffffffffffffffffffffff169060200190919050506102ca565b6040518082815260200191505060405180910390f35b341561013757600080fd5b610163600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190505061038a565b60405180871515151581526020018673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001851515151581526020018467ffffffffffffffff1667ffffffffffffffff1681526020018367ffffffffffffffff1667ffffffffffffffff1681526020018267ffffffffffffffff1667ffffffffffffffff168152602001965050505050505060405180910390f35b60008060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000160009054906101000a900460ff1615151561026657fe5b60016000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000160006101000a81548160ff0219169083151502179055506001905092915050565b6000808390508073ffffffffffffffffffffffffffffffffffffffff1663e3d670d7846040518263ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001915050602060405180830381600087803b151561036a57600080fd5b5af1151561037757600080fd5b5050506040518051905091505092915050565b60006020528060005260406000206000915090508060000160009054906101000a900460ff16908060000160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff16908060020160009054906101000a900460ff16908060020160019054906101000a900467ffffffffffffffff16908060020160099054906101000a900467ffffffffffffffff16908060020160119054906101000a900467ffffffffffffffff169050865600a165627a7a72305820d49504ad3af05f651a51d8ec696f6a7db2d5f3b3164d57e9e1d54890b4f75edd0029`
)



func TestNewTasCredit(t *testing.T) {
	core.InitCore()
	chain := core.BlockChainImpl
	//latestBlock := chain.QueryTopBlock()
	state := chain.LatestStateDB()

	ctx := ChainTopCallContext()

	//部署合约1
	addr, _, _ := SimulateDeployContract(ctx, "test1", ABI, CODE)
	caller := BuildBoundContract(addr, ABI)

	testAccount := "testAC"
	address := common.StringToAddress(testAccount)
	state.AddBalance(util.ToETHAddress(address), new(big.Int).SetUint64(math.MaxUint64))

	callMsg := NewDefaultCallMsg(address, &addr, nil)

	ret := new(big.Int)
	err := caller.CallContract(ctx, NewCallOpt(callMsg, "test1", addr, address), ret)
	if err != nil {
		t.Log(err)
	} else {
		t.Log(ret)
	}

	//testAccount = "testAC"
	//address = common.StringToAddress(testAccount)
	//state.AddBalance(util.ToETHAddress(address), new(big.Int).SetUint64(math.MaxUint64))
	//
	//
	//callMsg = NewDefaultCallMsg(address, &addr, nil)
	//
	//b, err = caller.ResultCall(ctx, func() interface{} {
	//	return new(bool)
	//}, NewCallOpt(callMsg, "canDeposit", address, uint64(10)))
	//
	//if err != nil {
	//	t.Log(err)
	//} else {
	//	t.Log(*(b.(*bool)))
	//}
}