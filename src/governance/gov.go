package governance

import (
	"governance/contract"
	"governance/vote"
	"common"
	"core"
	"governance/param"
)

/*
**  Creator: pxf
**  Date: 2018/4/4 下午2:32
**  Description: 
*/

type GovContext struct {
	CreditContract *contract.TasCredit
	CodeContract	*contract.TemplateCode
	VotePool 	*vote.VotePool
	bc 	*core.BlockChain
	pm	*param.ParamManager
}

func NewGovContext(creditAddr common.Address, codeAddr common.Address, bc *core.BlockChain) (*GovContext, error) {
	var (
		code *contract.TemplateCode
		credit *contract.TasCredit
		err error
	)
	code, err = contract.NewTemplateCode(codeAddr)
	if err != nil {
		return nil, err
	}

	credit, err = contract.NewTasCredit(creditAddr)
	if err != nil {
		return nil, err
	}

	return &GovContext{
		CreditContract: credit,
		CodeContract: code,
		VotePool: vote.NewVotePool(),
		bc : bc,
	}, nil
}

func InitGov() bool {
	////加载配置文件
	//cm := param.NewConfINIManager("tas.conf")
	//cm.GetString("","")
	return true
}