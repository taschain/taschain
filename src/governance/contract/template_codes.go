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
	TEMPLATE_CODE = ``
	TEMPLATE_ABI = ``
)


type TemplateCode struct {
	BaseContract
}

type VoteTemplate struct {
	code []byte
	abi string
	blockNum uint64
	author common.Address
}


func NewTemplateCode(address common.Address) (*TemplateCode, error) {
	base, err := newBaseContract(address, TEMPLATE_CODE, TEMPLATE_ABI)
	if err != nil {
		return nil, err
	}
	return &TemplateCode{
		BaseContract: *base,
	}, nil
}

func (tc *TemplateCode) AddTemplate(addr common.Address, codes []byte, abi string) error {
	return tc.NoResultCall( "addTemplate", addr, codes, abi)
}

func (tc *TemplateCode) Template(addr common.Address) (*VoteTemplate, error) {
	if ret, err := tc.ResultCall(func() interface{} {
		return &VoteTemplate{}
	}, "template", addr); err != nil {
		return nil, err
	} else {
		return ret.(*VoteTemplate), nil
	}

}

