package global

import (
	"testing"
	"core"
	"governance/contract"
)

/*
**  Creator: pxf
**  Date: 2018/4/27 上午9:50
**  Description: 
*/

func TestParamWrapper(t *testing.T) {
	core.Clear(core.DefaultBlockChainConfig())
	core.InitCore()
	chain := core.BlockChainImpl
	chain.GetTransactionPool().Clear()


	callctx := contract.ChainTopCallContext()

	addr, _,  err := contract.SimulateDeployContract(callctx, "123", contract.PARAM_STORE_ABI, contract.PARAM_STORE_CODE)

	if err != nil {
		t.Fatal("部署失败", err)
	}

	boundcontract := contract.BuildBoundContract(addr, contract.PARAM_STORE_ABI)
	ps := contract.NewParamStore(callctx, boundcontract)

	pw := NewParamWrapper()

	//pw.load(ps)

	t.Log(pw.GetGasPriceMin(ps))
	t.Log(pw.GetBlockFixAward(ps))
	t.Log(pw.GetVoterCountMin(ps))
}