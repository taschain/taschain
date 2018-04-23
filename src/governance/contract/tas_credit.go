package contract

import (
	"common"
	"math/big"
)

/*
**  Creator: pxf
**  Date: 2018/4/13 下午5:56
**  Description: 
*/
const (
	CREDIT_ABI = `[{"constant":false,"inputs":[{"name":"ac","type":"address"},{"name":"delta","type":"uint32"}],"name":"addVoteAcceptCnt","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"ac","type":"address"},{"name":"delta","type":"uint32"}],"name":"addVoteCnt","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"ac","type":"address"},{"name":"v","type":"uint64"}],"name":"setLatestTransBlock","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"ac","type":"address"}],"name":"score","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"ac","type":"address"},{"name":"bound","type":"uint256"}],"name":"checkPermit","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"ac","type":"address"},{"name":"delta","type":"uint32"}],"name":"addTransCnt","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"ac","type":"address"}],"name":"creditInfo","outputs":[{"name":"transCnt","type":"uint32"},{"name":"latestTransBlock","type":"uint64"},{"name":"voteCnt","type":"uint32"},{"name":"voteAcceptCnt","type":"uint32"},{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"ac","type":"address"}],"name":"balance","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"credits","outputs":[{"name":"transCnt","type":"uint32"},{"name":"latestTransBlock","type":"uint64"},{"name":"voteCnt","type":"uint32"},{"name":"voteAcceptCnt","type":"uint32"}],"payable":false,"stateMutability":"view","type":"function"}]`
	CREDIT_CODE = `6060604052341561000f57600080fd5b6109368061001e6000396000f300606060405260043610610099576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063234907681461009e5780635f839d52146100e657806374a1ed731461012e578063776f38431461017a578063985ac176146101c7578063bade10c114610221578063d5b09cb714610269578063e3d670d71461030a578063fe5ff46814610357575b600080fd5b34156100a957600080fd5b6100e4600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803563ffffffff169060200190919050506103f1565b005b34156100f157600080fd5b61012c600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803563ffffffff1690602001909190505061046a565b005b341561013957600080fd5b610178600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803567ffffffffffffffff169060200190919050506104e3565b005b341561018557600080fd5b6101b1600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190505061054f565b6040518082815260200191505060405180910390f35b34156101d257600080fd5b610207600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091908035906020019091905050610652565b604051808215151515815260200191505060405180910390f35b341561022c57600080fd5b610267600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803563ffffffff1690602001909190505061067d565b005b341561027457600080fd5b6102a0600480803573ffffffffffffffffffffffffffffffffffffffff169060200190919050506106f6565b604051808663ffffffff1663ffffffff1681526020018567ffffffffffffffff1667ffffffffffffffff1681526020018463ffffffff1663ffffffff1681526020018363ffffffff1663ffffffff1681526020018281526020019550505050505060405180910390f35b341561031557600080fd5b610341600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610875565b6040518082815260200191505060405180910390f35b341561036257600080fd5b61038e600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610896565b604051808563ffffffff1663ffffffff1681526020018467ffffffffffffffff1667ffffffffffffffff1681526020018363ffffffff1663ffffffff1681526020018263ffffffff1663ffffffff16815260200194505050505060405180910390f35b806000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000160108282829054906101000a900463ffffffff160192506101000a81548163ffffffff021916908363ffffffff1602179055505050565b806000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600001600c8282829054906101000a900463ffffffff160192506101000a81548163ffffffff021916908363ffffffff1602179055505050565b806000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000160046101000a81548167ffffffffffffffff021916908367ffffffffffffffff1602179055505050565b60008060008060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002091506127108260000160009054906101000a900463ffffffff1663ffffffff168360000160049054906101000a900467ffffffffffffffff1667ffffffffffffffff164303028115156105e157fe5b048473ffffffffffffffffffffffffffffffffffffffff163160058460000160109054906101000a900463ffffffff160284600001600c9054906101000a900463ffffffff168560000160009054906101000a900463ffffffff16010163ffffffff16010390508092505050919050565b60008061065e8461054f565b9050828111156106715760019150610676565b600091505b5092915050565b806000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000160008282829054906101000a900463ffffffff160192506101000a81548163ffffffff021916908363ffffffff1602179055505050565b60008060008060008060008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000160009054906101000a900463ffffffff1694506000808773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000160049054906101000a900467ffffffffffffffff1693506000808773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600001600c9054906101000a900463ffffffff1692506000808773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000160109054906101000a900463ffffffff1691508573ffffffffffffffffffffffffffffffffffffffff1631905091939590929450565b60008173ffffffffffffffffffffffffffffffffffffffff16319050919050565b60006020528060005260406000206000915090508060000160009054906101000a900463ffffffff16908060000160049054906101000a900467ffffffffffffffff169080600001600c9054906101000a900463ffffffff16908060000160109054906101000a900463ffffffff169050845600a165627a7a723058202972d402109c4b78b6e73f0072b5941751706d7e621e106434a8d2e2f5a8fd360029`

)

type TasCredit struct {
	BoundContract
	ctx *CallContext
}

type CreditInfo struct {
	TransCnt uint32
	LatestTransBlock uint64
	VoteCnt uint32
	VoteAcceptCnt uint32
	Balance *big.Int
}

func NewTasCredit(ctx *CallContext, bc *BoundContract) *TasCredit {
	return &TasCredit{
		BoundContract: *bc,
		ctx: ctx,
	}
}

func (tc *TasCredit) AddTransCnt(addr common.Address, delta uint32) error {
	return tc.NoResultCall(tc.ctx, NewCallOpt(nil, "addTransCnt", addr, delta))
	//rp := func() interface{} {
	//	ret := uint32(0)
	//	return &ret
	//}
	//ret, err := tc.ResultCall(tc.ctx, rp, "addTransCnt", addr, delta)
	//if err != nil {
	//	return 0
	//}
	//return *(ret.(*uint32))
}

func (tc *TasCredit) SetLatestTransBlock(addr common.Address, block uint64) error {
	return tc.NoResultCall(tc.ctx,   NewCallOpt(nil,"setLatestTransBlock", addr, block))
}

func (tc *TasCredit) CreditInfo(addr common.Address) (*CreditInfo, error) {
	rp := func() interface{} {
		return &CreditInfo{}
	}
	ret, err := tc.ResultCall(tc.ctx, rp,  NewCallOpt(nil, "creditInfo", addr))
	if err != nil {
		return nil, err
	}
	return ret.(*CreditInfo), nil
}

func (tc *TasCredit) Score(addr common.Address) (*big.Int, error) {
	rp := func() interface{} {
		return new(big.Int)
	}
	ret, err := tc.ResultCall(tc.ctx, rp,  NewCallOpt(nil, "score", addr))
	if err != nil {
		return nil, err
	}
	return ret.(*big.Int), nil
}

func (tc *TasCredit) Balance(addr common.Address) (*big.Int, error) {
	rp := func() interface{} {
		return new(big.Int)
	}
	ret, err := tc.ResultCall(tc.ctx, rp,  NewCallOpt(nil, "balance", addr))
	if err != nil {
		return nil, err
	}
	return ret.(*big.Int), nil
}

//func (tc *TasCredit) AddVoteCnt(addr common.Address, delta uint32) error {
//	return tc.NoResultCall(tc.ctx,  "addVoteCnt", addr,  delta)
//}
//
//func (tc *TasCredit) AddVoteAcceptCnt(addr common.Address, delta uint32) error {
//	return tc.NoResultCall( tc.ctx, "addVoteAcceptCnt", addr,  delta)
//}
//
//func (tc *TasCredit) SetBlockNum(addr common.Address, num uint64) error {
//	return tc.NoResultCall( tc.ctx, "setBlockNum", addr,  num)
//}