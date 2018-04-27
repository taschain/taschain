package contract

import (
	"testing"
	"core"
	"governance/util"
	"vm/common/math"
)

/*
**  Creator: pxf
**  Date: 2018/4/25 下午6:51
**  Description: 
*/

func TestVoteAddrPool_AddVote(t *testing.T) {
	core.Clear(core.DefaultBlockChainConfig())
	core.InitCore()
	chain := core.BlockChainImpl
	chain.GetTransactionPool().Clear()

	//block := chain.CastingBlock()

	callctx := ChainTopCallContext()

	addr, _,  err := SimulateDeployContract(callctx, "123", VOTE_ADDR_POOL_ABI, VOTE_ADDR_POOL_CODE)

	if err != nil {
		t.Fatal("部署失败", err)
	}

	boundcontract := BuildBoundContract(addr, VOTE_ADDR_POOL_ABI)
	pool := &VoteAddrPool{
		BoundContract: *boundcontract,
		ctx: callctx,
	}

	voteAddr := &VoteAddr{
		StatBlock: 1,
		EffectBlock: 2,
		TxHash: util.String2Hash("twww"),
		Addr: util.String2Address("ccc"),
	}
	ret, err := pool.AddVote(voteAddr)
	if err != nil {
		t.Fatal("addvote出错", err)
	}
	if !ret {
		t.Fatal("addvote 失败", ret)
	}

	_vote, err := pool.GetVoteAddr(voteAddr.Addr)
	if err != nil {
		t.Fatal("get state vote err", err)
	}
	t.Log(_vote)

	stats, err := pool.GetCurrentStatVoteHashes()
	if err != nil {
		t.Fatal("get state vote err", err)
	}
	t.Log(stats)


	stats, err = pool.GetCurrentEffectVoteHashes()
	if err != nil {
		t.Fatal("get effect vote err", err)
	}
	t.Log(stats)
}

func TestName(t *testing.T) {
	t.Log(uint64(math.MaxUint64))
}

