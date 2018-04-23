package governance

import (
	"testing"
	"governance/contract"
)

/*
**  Creator: pxf
**  Date: 2018/4/20 下午4:06
**  Description: 
*/

func showBalance(credit *contract.TasCredit, t *testing.T) {
	for idx, voter := range voters {
		b, err := credit.Balance(voter)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("余额", idx, b)
	}

	b, err := credit.Balance(voteAddress)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("投票合约余额", b)
}

func TestPrepare(t *testing.T) {
	prepare()
}


func TestVote(t *testing.T) {
	prepare()

	////block = chain.CastingBlock()
	////callctx = contract.NewCallContext(block, chain, state)
	//t.Log("============第二块==============")
	//deployVote()
	//
	//callctx := contract.NewCallContext(block, chain , state)
	//credit := gov.NewTasCreditInst(callctx)
	//showBalance(credit, t)
	//
	//for _, voter := range voters {
	//	addDeposit(&voter, 30)
	//	balance := state.GetBalance(util.ToETHAddress(voter))
	//	t.Log(common.ToHex(voter.Bytes()), balance)
	//}
	//
	//t.Log("上链2....")
	//state = newStateDB()
	//t.Log("===========")
	//for _, voter := range voters {
	//	balance := state.GetBalance(util.ToETHAddress(voter))
	//	t.Log(common.ToHex(voter.Bytes()), balance)
	//}
	//
	//block = chain.CastingBlock()
	//success := InsertChain(block, state)
	//if !success {
	//	t.Fatal("上链失败2")
	//}
	//
	//
	//t.Log("交保证金后=====================")
	//callctx = contract.NewCallContext(block, chain, state)
	//credit = gov.NewTasCreditInst(callctx)
	//showBalance(credit, t)
	//
	//vote := gov.NewVoteInst(callctx, voteAddress)
	//addrs, err := vote.VoterAddrs()
	//t.Log("投票地址:", len(addrs), addrs, err)
	//
	//t.Log("============第三块==============")
	//
	////代理
	//delegate(&voters[0], voters[9])
	//delegate(&voters[1], voters[9])
	//
	////投票
	//for i, voter := range voters {
	//	if i > 6 {
	//		doVote(&voter, true)
	//	} else {
	//		doVote(&voter, false)
	//	}
	//}
	//
	//block = chain.CastingBlock()
	//state = newStateDB()
	//success = InsertChain(block, state)
	//if !success {
	//	t.Fatal("上链失败3")
	//}
	//
	//t.Log("投票后=====================")
	//callctx = contract.NewCallContext(block, chain , state)
	//credit = gov.NewTasCreditInst(callctx)
	//showBalance(credit, t)
	//
	//def := gov.ParamManager.GetParamByIndex(2)
	//t.Log("参数生效前", def)
	//
	//success = InsertChain(chain.CastingBlock(), newStateDB())
	//if !success {
	//	t.Fatal("上链失败4")
	//}
	//
	//success = InsertChain(chain.CastingBlock(), newStateDB())
	//if !success {
	//	t.Fatal("上链失败5")
	//}
	//
	//block = chain.CastingBlock()
	//state = newStateDB()
	//success = InsertChain(block, state)
	//if !success {
	//	t.Fatal("上链失败6")
	//}
	//
	//def = gov.ParamManager.GetParamByIndex(2)
	//t.Log("参数生效后", def)
	//
	//t.Log("生效后=====================")
	//callctx = contract.NewCallContext(block, chain , state)
	//credit = gov.NewTasCreditInst(callctx)
	//showBalance(credit, t)

}