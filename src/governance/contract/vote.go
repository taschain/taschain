package contract

import (
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/4/13 下午5:56
**  Description: 
*/

const (
	VOTE_CODE = ``
	VOTE_ABI = ``
)


type Vote struct {
	BaseContract
}


func NewVote(address common.Address) (*Vote, error) {
	base, err := newBaseContract(address, VOTE_CODE, VOTE_ABI)
	if err != nil {
		return nil, err
	}
	return &Vote{
		BaseContract: *base,
	}, nil
}

func (v *Vote) CheckResult() (bool, error) {
	if ret, err := v.ResultCall(func() interface{} {
		return bool(false)
	}, "checkResult"); err != nil {
		return false, err
	} else {
		return ret.(bool), nil
	}

}

func (v *Vote) HandleDeposit() error {
	return v.NoResultCall("handleDeposit")
}
