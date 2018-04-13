package contract

import (
	"common"
	"common/abi"
	"strings"
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
	code []byte
	caller ContractCaller
}

func NewTasCredit(address common.Address) (*TasCredit, error) {
	abi, err := abi.JSON(strings.NewReader(CREDIT_ABI))
	if err != nil {
		return nil, err
	}

	caller := &BoundContract{
		address: address,
		abi: abi,
	}

	return &TasCredit{
		code: 	common.Hex2Bytes(CREDIT_CODE),
		caller: caller,
	}, nil
}

func (tc *TasCredit) noResultCall(addr common.Address, method string, value ...interface{})  {
	ctx := NewCallContext(method, value...)
	tc.caller.CallContract(ctx, 0)
}
func (tc *TasCredit) AddTransCnt(addr common.Address, delta uint32)  {
	tc.noResultCall(addr, "addTransCnt", delta)
}

func (tc *TasCredit) SetLatestTransBlock(addr common.Address, block uint64)  {
	tc.noResultCall(addr, "setLatestTransBlock", block)
}

func (tc *TasCredit) AddVoteCnt(addr common.Address, delta uint32)  {
	tc.noResultCall(addr, "addVoteCnt", delta)
}

func (tc *TasCredit) AddVoteAcceptCnt(addr common.Address, delta uint32)  {
	tc.noResultCall(addr, "addVoteAcceptCnt", delta)
}

func (tc *TasCredit) SetBlockNum(addr common.Address, num uint64)  {
	tc.noResultCall(addr, "setBlockNum", num)
}