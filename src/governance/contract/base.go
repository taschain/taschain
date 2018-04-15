package contract

import (
	"common"
	"strings"
	"common/abi"
)

/*
**  Creator: pxf
**  Date: 2018/4/13 下午5:56
**  Description: 
*/

type BaseContract struct {
	code []byte
	caller ContractCaller
}

type ResultProvider func() interface{}

func newBaseContract(addr common.Address, codes string, abis string) (*BaseContract, error) {
	abi, err := abi.JSON(strings.NewReader(CREDIT_ABI))
	if err != nil {
		return nil, err
	}

	caller := &BoundContract{
		address: addr,
		abi: abi,
	}
	return &BaseContract{
		code: common.Hex2Bytes(codes),
		caller: caller,
	}, nil
}

func (tc *BaseContract) NoResultCall(method string, value ...interface{}) error {
	ctx := NewCallContext(method, value...)
	return tc.caller.CallContract(ctx, 0)
}

func (tc *BaseContract) ResultCall(rp ResultProvider, method string, args ...interface{}) (interface{}, error) {
	ctx := NewCallContext(method, args...)
	ret := rp()
	if err := tc.caller.CallContract(ctx, ret); err != nil {
		return nil, err
	}
	return ret, nil
}