package global

import (
	"testing"
	"governance/contract"
	"core"
	"common"
	"vm/common/math"
	"math/big"
	"governance/util"
)

/*
**  Creator: pxf
**  Date: 2018/4/17 下午3:10
**  Description: 
*/

func TestTasCredit(t *testing.T) {
	chain := core.InitBlockChain()
	latestBlock := chain.QueryTopBlock()
	state := core.NewStateDB(latestBlock.StateTree, chain)

	ctx := contract.NewCallContext(chain.CastingBlock(), chain, state)

	addr, code, err := contract.SimulateDeployContract(ctx, DEPLOY_ACCOUNT, contract.CREDIT_ABI, contract.CREDIT_CODE)
	if err != nil {
		t.Fatal(err)
	}

	gov = newGOV(addr, common.Address{}, chain)

	credit := gov.NewTasCreditInst(ctx)

	address := common.StringToAddress("u1")
	state.AddBalance(util.ToETHAddress(address), new(big.Int).SetUint64(12345))

	t.Log("addr ", addr, ", codelen ", len(code))

	credit.AddTransCnt(address, 10)
	credit.AddTransCnt(address, 13)
	credit.SetLatestTransBlock(address, math.MaxUint64 -10)

	creditInfo, _ := credit.CreditInfo(address)

	address2 := common.StringToAddress("u2")
	state.AddBalance(util.ToETHAddress(address2), new(big.Int).SetUint64(23344212))

	credit.AddTransCnt(address2, 321)
	credit.AddTransCnt(address2, 13)
	credit.SetLatestTransBlock(address2, math.MaxUint64 -13333333330)

	creditInfo2, err := credit.CreditInfo(address2)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(creditInfo)
	t.Log(creditInfo2)
}

func TestTemplateCode(t *testing.T) {
	chain := core.InitBlockChain()
	latestBlock := chain.QueryTopBlock()
	state := core.NewStateDB(latestBlock.StateTree, chain)

	ctx := contract.NewCallContext(chain.CastingBlock(), chain, state)

	creditAddr, _, _ := contract.SimulateDeployContract(ctx, DEPLOY_ACCOUNT, contract.CREDIT_ABI, contract.CREDIT_CODE)

	addr, _, _ := contract.SimulateDeployContract(ctx, DEPLOY_ACCOUNT, contract.TEMPLATE_ABI, contract.TEMPLATE_CODE)

	gov = newGOV(creditAddr, addr, chain)
	credit := gov.NewTasCreditInst(ctx)
	templateCode := gov.NewTemplateCodeInst(ctx)


	address := common.StringToAddress("111")
	t.Log(credit.CreditInfo(address))
	templateCode.AddTemplate("111", "3132323223ff", "abidddd")
	vt, _ := templateCode.Template("111")

	t.Log(vt)

	address = common.StringToAddress("134")
	templateCode.AddTemplate("123", "a123322334dddfff", "afgfargarg")
	vt, _ = templateCode.Template("123")

	t.Log(vt)
}

func TestVote(t *testing.T) {
	common.InitConf("../test.ini")
	chain := core.InitBlockChain()
	latestBlock := chain.QueryTopBlock()
	state := core.NewStateDB(latestBlock.StateTree, chain)

	ctx := contract.NewCallContext(chain.CastingBlock(), chain, state)

	//部署合约1
	creditAddr, _, _ := contract.SimulateDeployContract(ctx, DEPLOY_ACCOUNT, contract.CREDIT_ABI, contract.CREDIT_CODE)

	//部署合约2
	addr, _, _ := contract.SimulateDeployContract(ctx, DEPLOY_ACCOUNT, contract.TEMPLATE_ABI, contract.TEMPLATE_CODE)

	//初始化治理环境
	gov = newGOV(creditAddr, addr, chain)
	_ = gov.NewTasCreditInst(ctx)
	_ = gov.NewTemplateCodeInst(ctx)

	//创建一个测试账户, 并设置余额
	testAccount := "test"
	address := common.StringToAddress(testAccount)
	state.AddBalance(util.ToETHAddress(address), new(big.Int).SetUint64(math.MaxUint64))
	testAccount2 := "test2"
	address2 := common.StringToAddress(testAccount2)
	state.AddBalance(util.ToETHAddress(address2), new(big.Int).SetUint64(math.MaxUint64))
	//创建一个投票代理账户, 并设置余额
	delegateAccount := "delegate"
	delegateAddr := common.StringToAddress(delegateAccount)
	state.AddBalance(util.ToETHAddress(delegateAddr), new(big.Int).SetUint64(math.MaxUint64))

	//部署投票合约
	voteAddr, _, err := contract.SimulateDeployContract(ctx, DEPLOY_ACCOUNT , contract.VOTE_ABI, contract.VOTE_CODE, creditAddr, uint64(1), uint64(1), uint64(1), uint64(1), uint64(1), uint64(10), uint64(11), uint64(12), uint64(1), uint64(1), uint64(5))

	if err != nil {
		t.Fatal(err)
	}

	vote := gov.NewVoteInst(ctx, voteAddr)

	callMsg := contract.NewSimulateCallMsg(address, &voteAddr, 5000000)
	callMsg2 := contract.NewSimulateCallMsg(address2, &voteAddr, 5000000)

	//缴纳保证金
	err = vote.NoResultCall(ctx, contract.NewCallOpt(callMsg, "addDeposit", uint64(10)))
	if err != nil {
		t.Fatal("缴纳保证金失败!", err)
	}

	//代理
	err = vote.NoResultCall(ctx, contract.NewCallOpt(callMsg, "delegateTo", delegateAddr))
	if err != nil {
		t.Fatal(err)
	}

	//缴纳保证金
	err = vote.NoResultCall(ctx, contract.NewCallOpt(callMsg2, "addDeposit", uint64(120)))
	if err != nil {
		t.Fatal("缴纳保证金失败!", err)
	}

	//代理
	err = vote.NoResultCall(ctx, contract.NewCallOpt(callMsg2, "delegateTo", delegateAddr))
	if err != nil {
		t.Fatal(err)
	}

	delegateCallMsg := contract.NewSimulateCallMsg(delegateAddr, &voteAddr, 5000000)
	//代理账户缴纳保证金
	err = vote.NoResultCall(ctx, contract.NewCallOpt(delegateCallMsg, "addDeposit", uint64(23)))
	if err != nil {
		t.Fatal("代理人缴纳保证金失败!", err)
	}

	//voterInfo, _ := vote.VoterInfo(address)
	//score, _ := vote.ScoreOf(address)
	//t.Log("投票人voterinfo:", voterInfo, "score: ", score)
	//
	//
	//voterInfo, _ = vote.VoterInfo(delegateAddr)
	//score, _ = vote.ScoreOf(delegateAddr)
	//t.Log("代理投票人voterinfo:", voterInfo, "score: ", score)



	//代理人投票
	err = vote.NoResultCall(ctx, contract.NewCallOpt(delegateCallMsg, "vote", true))
	if err != nil {
		t.Log(err)
	}


	voterInfo, _ := vote.VoterInfo(address)
	t.Log("投票信息: ", voterInfo)
	voterInfo, _ = vote.VoterInfo(address2)
	t.Log("投票信息2: ", voterInfo)
	voterInfo, _ = vote.VoterInfo(delegateAddr)
	t.Log("代理人投票信息: ", voterInfo)

	voterAddrs, _ := vote.VoterAddrs()
	t.Log("投票成功后: ", " voterAddr: ", voterAddrs)
}