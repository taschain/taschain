package logical

import (
	"testing"
	"consensus/groupsig"
	"middleware"
	"common"
	"consensus/model"
	"core"
	"consensus/mediator"
	"consensus/logical"
	"taslog"
)

/*
**  Creator: pxf
**  Date: 2019/1/17 下午12:37
**  Description: 
*/
func initProcessor(conf string) *logical.Processor {
	common.InitConf(conf)
	middleware.InitMiddleware()
	common.DefaultLogger = taslog.GetLoggerByIndex(taslog.DefaultConfig, common.GlobalConf.GetString("instance", "index", ""))
	addr := common.HexToAddress(common.GlobalConf.GetString("gtas", "miner", ""))
	mdo := model.NewSelfMinerDO(addr)
	logical.InitConsensus()
	core.InitCore(false, mediator.NewConsensusHelper(mdo.ID))
	p := new(logical.Processor)
	p.Init(mdo, common.GlobalConf)
	return p
}

func TestCalcVerifyGroup(t *testing.T) {
	p := initProcessor("tas3.ini")

	top := p.MainChain.Height()
	pre := p.MainChain.QueryBlockByHeight(0)
	t.Log("start, total %v", top)
	for h := uint64(1); h <= top; h++ {
		bh := p.MainChain.QueryBlockByHeight(h)
		if bh == nil {
			continue
		}
		gid := groupsig.DeserializeId(bh.GroupId)
		expectGid := p.CalcVerifyGroupFromChain(pre, h)
		pre = bh
		if !gid.IsEqual(*expectGid) {
			t.Fatalf("gid not equal, height %v, real gid %v, expect gid %v", h, gid.GetHexString(), expectGid.GetHexString())
		}
		t.Logf("height %v ok", h)
	}
	t.Log("ok")
}


func TestProcessor_GenProveHashs(t *testing.T) {
	p := initProcessor("tas3.ini")

	var id groupsig.ID
	id.SetHexString("0x9d2961d1b4eb4af2d78cb9e29614756ab658671e453ea1f6ec26b4e918c79d02")

	b := p.MainChain.QueryBlockFloor(7)
	hash := p.GenVerifyHash(b, id)
	t.Logf(hash.String())

}