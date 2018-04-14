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
	CREDIT_CODE = ``
	CREDIT_ABI = ``
)


type TasCredit struct {
	BaseContract
}

func NewTasCredit(address common.Address) (*TasCredit, error) {
	base, err := newBaseContract(address, CREDIT_CODE, CREDIT_ABI)
	if err != nil {
		return nil, err
	}
	return &TasCredit{
		BaseContract: *base,
	}, nil
}


func (tc *TasCredit) AddTransCnt(addr common.Address, delta uint32) error {
	return tc.NoResultCall( "addTransCnt", addr, delta)
}

func (tc *TasCredit) SetLatestTransBlock(addr common.Address, block uint64) error {
	return tc.NoResultCall( "setLatestTransBlock", addr, block)
}

func (tc *TasCredit) AddVoteCnt(addr common.Address, delta uint32) error {
	return tc.NoResultCall( "addVoteCnt", addr,  delta)
}

func (tc *TasCredit) AddVoteAcceptCnt(addr common.Address, delta uint32) error {
	return tc.NoResultCall( "addVoteAcceptCnt", addr,  delta)
}

func (tc *TasCredit) SetBlockNum(addr common.Address, num uint64) error {
	return tc.NoResultCall( "setBlockNum", addr,  num)
}