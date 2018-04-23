package governance

import (
	"testing"
	"common"
	"core"
	"governance/contract"
	"governance/global"
	"math/big"
	"governance/util"
)

/*
**  Creator: pxf
**  Date: 2018/4/20 上午10:10
**  Description: 
*/

func TestGetRealCode(t *testing.T) {
	common.InitConf("test.ini")
	chain := core.InitBlockChain()
	latestBlock := chain.QueryTopBlock()
	state := core.NewStateDB(latestBlock.StateTree, chain)

	block := chain.CastingBlock()
	ctx := contract.NewCallContext(block, chain, state)

	//部署合约1
	creditAddr, _, _ := contract.SimulateDeployContract(ctx, global.DEPLOY_ACCOUNT, contract.CREDIT_ABI, contract.CREDIT_CODE)

	//部署合约2
	templateAddr, _, _ := contract.SimulateDeployContract(ctx, global.DEPLOY_ACCOUNT, contract.TEMPLATE_ABI, contract.TEMPLATE_CODE)

	t.Log(creditAddr, templateAddr)

	//初始化治理环境
	global.InitGov(chain)
	gov := global.GetGOV()
	credit := gov.NewTasCreditInst(ctx)
	template := gov.NewTemplateCodeInst(ctx)

	tname := "vote_temp_1"
	//添加模板
	template.AddTemplate(tname, contract.VOTE_CODE, contract.VOTE_ABI)

	//查看模板
	tc, err := template.Template(tname)
	if err != nil {
		t.Fatal("获取代码模板失败", err)
	} else {
		t.Log("模板:", tc)
	}

	//配置项
	cfg := &global.VoteConfig{
		TemplateName: tname,
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

	//配置项编码
	input, err := cfg.AbiEncode()
	if err != nil {
		t.Fatal(err)
	}

	//获取真正的执行代码
	var code []byte
	code, err = GetRealCode(block, state, tname, input)

	err = credit.AddTransCnt(common.StringToAddress("vote_launcher"), 100)
	if err != nil {
		t.Fatal(err)
	}

	//部署合约
	voteAddr, _, err := contract.Deploy(ctx, "vote_launcher", code)
	if err != nil {
		t.Fatal("部署投票失败:", err)
	}
	vote := gov.NewVoteInst(ctx, voteAddr)

	//查看部署的合约配置信息
	config, err := vote.ResultCall(ctx, func() interface{} {
		return new(global.VoteConfig)
	}, contract.NewCallOpt(nil, "config"))
	t.Log(config, err)

	//创建投票的测试账户
	account := common.StringToAddress("vote_ac")
	state.AddBalance(util.ToETHAddress(account), new(big.Int).SetUint64(99999999))

	//账户余额
	b,_ := credit.Balance(account)
	t.Log("余额:", b)

	voterInfo, _ := vote.VoterInfo(account)
	t.Log("投票信息1: ", voterInfo)

	creditInfo, _ := credit.CreditInfo(account)
	t.Log("信用信息:", creditInfo)

	depositMsg := &contract.CallMsg{
		From: account,
		To: &voteAddr,
		Value: uint64(20),
		Gas: 500000,
		GasPrice: 1,
	}
	//depositMsg := contract.NewSimulateCallMsg(account, &voteAddr, 500000)

	//can, err := vote.ResultCall(ctx, func() interface{} {
	//	return new(bool)
	//}, contract.NewCallOpt(depositMsg, "_canVote"))
	//t.Log(*(can.(*bool)), err)

	//缴纳保证金
	err = vote.NoResultCall(ctx, contract.NewCallOpt(depositMsg, "addDeposit", depositMsg.Value))
	if err != nil {
		t.Fatal("缴纳保证金失败!", err)
	}
	b , err = credit.Balance(account)
	t.Log("缴纳后余额:", b, err)

	//投票
	voteMsg := &contract.CallMsg{
		From: account,
		To: &voteAddr,
		Value: 0,
		Gas: 1000000,
		GasPrice: 1,
	}
	err = vote.NoResultCall(ctx, contract.NewCallOpt(voteMsg, "vote", true))
	if err != nil {
		t.Log("投票失败", err)
	}
	b , err = credit.Balance(account)
	t.Log("投票后余额:", b, err)

	voterInfo, _ = vote.VoterInfo(account)
	t.Log("投票信息: ", voterInfo)
}