package global

import (
	"governance/contract"
	"governance/param"
	"core"
	"common"
	"governance/util"
	"vm/crypto"
)

/*
**  Creator: pxf
**  Date: 2018/4/19 上午9:56
**  Description: 
*/


const (
	DEPLOY_ACCOUNT = `_tas_gov_contract_admin_`
	VOTE_SCORE_MIN = 10
	LAUNCH_VOTE_SCORE_MIN = 100
	VOTER_CNT_MIN = 5
	APPROVAL_VOTER_MIN = 3
	CONF_SECTION = "gov"
)

type GOV struct {
	CreditContract *contract.BoundContract
	CodeContract   *contract.BoundContract
	VoteContract   *contract.BoundContract
	VotePool       *VotePool
	BlockChain     *core.BlockChain
	ParamManager   *param.ParamManager
	VoteScoreMin	uint64	//能投票的最低分数
	LaunchVoteScoreMin	uint64	//能发起投票的最低分数
	VoterCntMin		uint64	//最低参与投票人数
	ApprovalVoterMin	uint64	//投票通过的最低人数


	init           bool
}

var gov *GOV

func newGOV(creditAddr common.Address, codeAddr common.Address, bc *core.BlockChain) (*GOV) {
	cfm := common.GlobalConf.GetSectionManager(CONF_SECTION)

	return &GOV{
		CreditContract: contract.BuildBoundContract(creditAddr, contract.CREDIT_ABI),
		CodeContract:   contract.BuildBoundContract(codeAddr, contract.TEMPLATE_ABI),
		VoteContract:   contract.BuildBoundContract(common.Address{}, contract.VOTE_ABI),
		VotePool:       NewVotePool(),
		ParamManager:   param.NewParamManager(bc),
		BlockChain:     bc,
		VoteScoreMin: uint64(cfm.GetInt("voter_score_min", VOTE_SCORE_MIN)),
		LaunchVoteScoreMin: uint64(cfm.GetInt("launch_voter_score_min", LAUNCH_VOTE_SCORE_MIN)),
		VoterCntMin: uint64(cfm.GetInt("voter_cnt_min", VOTER_CNT_MIN)),
		ApprovalVoterMin: uint64(cfm.GetInt("approval_voter_min", APPROVAL_VOTER_MIN)),
		init:           true,
	}
}

func GetGOV() *GOV {
	if gov == nil || !gov.init {
		panic("gov module not init!")
	}
	return gov
}

func (g *GOV) NewVoteInst(ctx *contract.CallContext, address common.Address) *contract.Vote {
	return contract.NewVote(ctx, address, g.VoteContract)
}

func (g *GOV) NewTasCreditInst(ctx *contract.CallContext) *contract.TasCredit {
	return contract.NewTasCredit(ctx, g.CreditContract)
}

func (g *GOV) NewTemplateCodeInst(ctx *contract.CallContext) *contract.TemplateCode {
	return contract.NewTemplateCode(ctx, g.CodeContract)
}

func InitGov(bc *core.BlockChain) bool {
	if gov != nil && gov.init {
		return true
	}

	creditAddr := crypto.CreateAddress(util.ToETHAddress(common.StringToAddress(DEPLOY_ACCOUNT)), 0)
	codeAddr := crypto.CreateAddress(util.ToETHAddress(common.StringToAddress(DEPLOY_ACCOUNT)), 1)
	gov = newGOV(util.ToTASAddress(creditAddr), util.ToTASAddress(codeAddr), bc)
	return true
}