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