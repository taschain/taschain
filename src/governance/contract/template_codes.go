package contract

import (
	"common"
	ethcom "vm/common"
)

/*
**  Creator: pxf
**  Date: 2018/4/13 下午5:56
**  Description: 
*/


type TemplateCode struct {
	BoundContract
	ctx *CallContext
}

type VoteTemplate struct {
	Code []byte
	Abi string
	BlockNum uint64
	Author ethcom.Address
}


func NewTemplateCode(ctx *CallContext, bc *BoundContract) (*TemplateCode) {
	return &TemplateCode{
		BoundContract: *bc,
		ctx:ctx,
	}
}

func (tc *TemplateCode) AddTemplate(addr common.Address, codes []byte, abi string) error {
	return tc.NoResultCall(tc.ctx,  NewCallOpt(nil, "addTemplate", addr, codes, abi))
}

func (tc *TemplateCode) Template(addr common.Address) (*VoteTemplate, error) {
	if ret, err := tc.ResultCall(tc.ctx, func() interface{} {
		return &VoteTemplate{}
	},  NewCallOpt(nil, "template", addr)); err != nil {
		return nil, err
	} else {
		return ret.(*VoteTemplate), nil
	}

}

