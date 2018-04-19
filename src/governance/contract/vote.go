package contract

import (
	"common"
	ethcom "vm/common"
	"math/big"
)

/*
**  Creator: pxf
**  Date: 2018/4/13 下午5:56
**  Description: 
*/


type Vote struct {
	BoundContract
	ctx *CallContext
}

type VoterInfo struct {
	Addr ethcom.Address
	Voted bool
	Delegate ethcom.Address
	AsDelegates []ethcom.Address
	Approval bool
	VoteBlock uint64
	Deposit uint64
	DepositBlock uint64
}


func NewVote(ctx *CallContext, address common.Address, bc *BoundContract) (*Vote) {
	return &Vote{
		BoundContract: *newBoundContract(address, bc.abi),
		ctx: ctx,
	}
}

func (v *Vote) CheckResult() (bool, error) {
	if ret, err := v.ResultCall(v.ctx, func() interface{} {
		return bool(false)
	},  NewCallOpt(nil, "checkResult")); err != nil {
		return false, err
	} else {
		return ret.(bool), nil
	}

}

func (v *Vote) HandleDeposit() error {
	return v.NoResultCall(v.ctx, NewCallOpt(nil, "handleDeposit"))
}

func (v *Vote) UpdateCredit(pass bool) error {
	return v.NoResultCall(v.ctx, NewCallOpt(nil, "updateCredit", pass))
}

func (v *Vote) VoterInfo(addr common.Address) (*VoterInfo, error) {
	if v, err := v.ResultCall(v.ctx, func() interface{} {
		return &VoterInfo{}
	},  NewCallOpt(nil,"voterInfo", addr)); err != nil {
		return nil, err
	} else {
		return v.(*VoterInfo), nil
	}
}

func (v *Vote) VoterAddrs() ([]ethcom.Address, error) {
	if v, err := v.ResultCall(v.ctx, func() interface{} {
		return &[]ethcom.Address{}
	},  NewCallOpt(nil, "voterAddrList")); err != nil {
		return nil, err
	} else {
		return *(v.(*[]ethcom.Address)), nil
	}
}


func (v *Vote) ScoreOf(addr common.Address) (*big.Int, error) {
	if v, err := v.ResultCall(v.ctx, func() interface{} {
		return new(big.Int)
	},  NewCallOpt(nil, "scoreOf", addr)); err != nil {
		return nil, err
	} else {
		return v.(*big.Int), nil
	}
}