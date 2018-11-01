package global

import (
	"governance/contract"
	"core"
	"common"
	"governance/util"
	"vm/crypto"
	"taslog"
)

/*
**  Creator: pxf
**  Date: 2018/4/19 上午9:56
**  Description: 
*/


const (
	DEPLOY_ACCOUNT = `_tas_gov_contract_admin_`
	VOTE_SCORE_MIN = 5
	LAUNCH_VOTE_SCORE_MIN = 100
	VOTER_CNT_MIN = 5
	APPROVAL_VOTER_MIN = 3
	CONF_SECTION = "gov"
)

type GOV struct {
	CreditContract *contract.BoundContract
	CodeContract   *contract.BoundContract
	VoteContract   *contract.BoundContract
	VoteAddrPool   *contract.BoundContract
	ParamStore		*contract.BoundContract
	BlockChain     core.BlockChain
	ParamWrapper   *ParamWrapper

	VoteScoreMin	uint64	//能投票的最低分数
	LaunchVoteScoreMin	uint64	//能发起投票的最低分数
	VoterCntMin		uint64	//最低参与投票人数
	ApprovalVoterMin	uint64	//投票通过的最低人数

	Logger 	taslog.Logger

	init           bool
}

type NewGovParam struct {
	creditAddr common.Address
	codeAddr common.Address
	poolAddr common.Address
	paramStoreAddr common.Address
	bc *core.BlockChain
}

var gov *GOV

func newGOV(p *NewGovParam) (*GOV) {

	cfm := common.GlobalConf.GetSectionManager(CONF_SECTION)

	return &GOV{
		CreditContract: contract.BuildBoundContract(p.creditAddr, contract.CREDIT_ABI),
		CodeContract:   contract.BuildBoundContract(p.codeAddr, contract.TEMPLATE_ABI),
		VoteContract:   contract.BuildBoundContract(common.Address{}, contract.VOTE_ABI),
		VoteAddrPool:   contract.BuildBoundContract(p.poolAddr, contract.VOTE_ADDR_POOL_ABI),
		ParamStore:   contract.BuildBoundContract(p.paramStoreAddr, contract.PARAM_STORE_ABI),
		ParamWrapper:   NewParamWrapper(),
		BlockChain:     p.bc,
		VoteScoreMin: uint64(cfm.GetInt("voter_score_min", VOTE_SCORE_MIN)),
		LaunchVoteScoreMin: uint64(cfm.GetInt("launch_voter_score_min", LAUNCH_VOTE_SCORE_MIN)),
		VoterCntMin: uint64(cfm.GetInt("voter_cnt_min", VOTER_CNT_MIN)),
		ApprovalVoterMin: uint64(cfm.GetInt("approval_voter_min", APPROVAL_VOTER_MIN)),
		Logger: taslog.GetLoggerByName(cfm.GetString("logger_name", "gov")),
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
func (g *GOV) NewVoteAddrPoolInst(ctx *contract.CallContext) *contract.VoteAddrPool {
	return contract.NewVoteAddrPool(ctx, g.VoteAddrPool)
}

func (g *GOV) NewParamStoreInst(ctx *contract.CallContext) *contract.ParamStore {
	return contract.NewParamStore(ctx, g.ParamStore)
}

func ShowVoterInfo(address common.Address)  {
	callctx := contract.ChainTopCallContext()
	vote := gov.NewVoteInst(callctx, address)
	vs, err := vote.VoterAddrs()
	if err != nil {
		gov.Logger.Error("获取地址", err)
	}
	for _, voter := range vs {
		voteInfo, err := vote.VoterInfo(util.ToTASAddress(voter))
		if err != nil {
			gov.Logger.Error("获取投票信息", err)
		}
		gov.Logger.Info(common.ToHex(voter.Bytes()), voteInfo)
	}
}

func InitGov(bc core.BlockChain) bool {
	if gov != nil && gov.init {
		return true
	}

	deployAcc := util.ToETHAddress(util.String2Address("3"))
	creditAddr := crypto.CreateAddress(deployAcc, 0)
	codeAddr := crypto.CreateAddress(deployAcc, 1)
	votePoolAddr := crypto.CreateAddress(deployAcc, 2)
	psAddr := crypto.CreateAddress(deployAcc, 3)

	param := &NewGovParam{
		codeAddr: util.ToTASAddress(codeAddr),
		creditAddr: util.ToTASAddress(creditAddr),
		poolAddr: util.ToTASAddress(votePoolAddr),
		paramStoreAddr: util.ToTASAddress(psAddr),
		bc: bc,
	}

	gov = newGOV(param)

	callctx := contract.ChainTopCallContext()
	gov.ParamWrapper.load(gov.NewParamStoreInst(callctx))

	bc.SetVoteProcessor(NewChainEventProcessor())
	return true
}