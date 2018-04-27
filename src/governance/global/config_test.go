package global

import (
	"testing"
	"core"
	"governance/contract"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/4/19 下午5:50
**  Description: 
*/

func TestConfig(t *testing.T) {
	common.InitConf("test.ini")
	err := core.InitCore()
	if err != nil {
		t.Fatal("初始化失败", err)
	}
	chain := core.BlockChainImpl
	//latestBlock := chain.QueryTopBlock()
	//state := core.NewStateDB(latestBlock.StateTree, chain)

	ctx := contract.NewCallContext(chain.QueryTopBlock(), chain, chain.LatestStateDB())

	//部署合约1
	creditAddr, _, _ := contract.SimulateDeployContract(ctx, DEPLOY_ACCOUNT, contract.CREDIT_ABI, contract.CREDIT_CODE)

	//部署合约2
	addr, _, _ := contract.SimulateDeployContract(ctx, DEPLOY_ACCOUNT, contract.TEMPLATE_ABI, contract.TEMPLATE_CODE)

	//初始化治理环境
	param := &NewGovParam{
		creditAddr: creditAddr,
		codeAddr: addr,
		bc: chain,
	}
	gov = newGOV(param)
	_ = gov.NewTasCreditInst(ctx)
	_ = gov.NewTemplateCodeInst(ctx)

	cfg := &VoteConfig{
		TemplateName: "test_template",
		PIndex: 2,
		PValue: "103",
		Custom: false,
		Desc: "描述",
		DepositMin: 10,
		TotalDepositMin: 20,
		VoterCntMin: 4,
		ApprovalDepositMin: 14,
		ApprovalVoterCntMin: 4,
		DeadlineBlock: 20000,
		StatBlock: 20003,
		EffectBlock: 2303223,
		DepositGap: 1332,
	}

	ret, err := cfg.AbiEncode()
	if err != nil {
		t.Fatal(err)
	}

	var convert []byte
	_cfg, err := AbiDecodeConfig(ret)
	convert, err = _cfg.convert()

	////部署投票合约
	//voteAddr, _, err := contract.SimulateDeployContract(ctx, DEPLOY_ACCOUNT , contract.VOTE_ABI, contract.VOTE_CODE,
	//	creditAddr,
	//	cfg.DepositMin,
	//	cfg.TotalDepositMin,
	//	cfg.VoterCntMin,
	//	cfg.ApprovalDepositMin,
	//	cfg.ApprovalVoterCntMin,
	//	cfg.DeadlineBlock,
	//	cfg.StatBlock,
	//	cfg.EffectBlock,
	//	cfg.DepositGap,
	//	corei.VoteScoreMin,
	//	corei.LaunchVoteScoreMin)
	var realCfg = new(VoteConfig)
	err = gov.VoteContract.GetAbi().Unpack(realCfg, "", convert)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(realCfg)
}