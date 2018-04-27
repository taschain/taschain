package contract

import (
	"testing"
	"core"
	"governance/util"
)

/*
**  Creator: pxf
**  Date: 2018/4/25 下午6:51
**  Description: 
*/

func TestParamStore(t *testing.T) {
	core.Clear(core.DefaultBlockChainConfig())
	core.InitCore()
	chain := core.BlockChainImpl
	chain.GetTransactionPool().Clear()

	state := chain.LatestStateDB()

	callctx := NewCallContext(chain.QueryTopBlock(), chain, state)

	addr, _,  err := SimulateDeployContract(callctx, "123", PARAM_STORE_ABI, PARAM_STORE_CODE)

	if err != nil {
		t.Fatal("部署失败", err)
	}

	boundcontract := BuildBoundContract(addr, PARAM_STORE_ABI)
	ps := &ParamStore{
		BoundContract: *boundcontract,
		ctx: callctx,
	}

	for i := 0; i < 3; i ++ {
		meta, err := ps.GetCurrentMeta(uint32(i))
		if err != nil {
			t.Fatal("获取meta失败",err)
		}
		t.Log(i, meta)
	}

	err = ps.AddFuture(0, &ParamMeta{Value:"20", TxHash:util.String2Hash("23"), EffectBlock:2})
	if err != nil {
		t.Fatal("addfuture失败", err)
	} else {
		len, _:= ps.FutureLength(0)
		t.Log("futurelength", len)
	}
	fm ,err := ps.GetFutureMeta(0, 0)
	if err != nil {
		t.Fatal("获取future失败", err)
	}
	t.Log("apply前", fm)

	chain.AddBlockOnChain(chain.CastingBlock())

	callctx = ChainTopCallContext()
	ps = &ParamStore{
		BoundContract: *boundcontract,
		ctx: callctx,
	}

	err = ps.ApplyFuture(0)
	if err != nil {
		t.Fatal("applyfuture fail", err)
	}
	len, _:= ps.FutureLength(0)
	t.Log("futurelength", len)

	fm ,err = ps.GetFutureMeta(0, 0)
	if err != nil {
		t.Fatal("获取future失败", err)
	}
	t.Log("apply后", fm)

	meta, err := ps.GetCurrentMeta(uint32(0))
	if err != nil {
		t.Fatal("获取meta失败",err)
	}
	t.Log( meta)

}


