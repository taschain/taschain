package cli

import (
	"consensus/mediator"
	"middleware/types"
	"core"
	"common"
	"github.com/vmihailenco/msgpack"
	"consensus/groupsig"
	"time"
	"encoding/json"
)

/*
**  Creator: pxf
**  Date: 2018/9/30 下午4:34
**  Description: 
*/
func successResult(data interface{}) (*Result, error) {
	return &Result{
		Message:"success",
		Data:data,
	}, nil
}
func failResult(err string) (*Result, error) {
	return &Result{
		Message:err,
		Data:nil,
	}, nil
}

func (api *GtasAPI) MinerApply(stake uint64, mtype int32) (*Result, error) {
	info := core.MinerManagerImpl.GetMinerById(mediator.Proc.GetMinerID().Serialize(), byte(mtype), nil)
	if info != nil {
		return failResult("已经申请过该类型矿工")
	}

	minerInfo := mediator.Proc.GetMinerInfo()
	address := common.BytesToAddress(minerInfo.ID.Serialize())
	nonce := time.Now().UnixNano()

	miner := &types.Miner{
		Id: minerInfo.ID.Serialize(),
		PublicKey: minerInfo.PK.Serialize(),
		VrfPublicKey: minerInfo.VrfPK,
		Stake: stake,
		Type: byte(mtype),
	}
	data, err := msgpack.Marshal(miner)
	if err != nil {
		return &Result{Message:err.Error(), Data:nil}, nil
	}
	tx := &types.Transaction{
		Nonce: uint64(nonce),
		Data: data,
		Source: &address,
		Value: stake,
		Type: types.TransactionTypeMinerApply,
	}
	tx.Hash = tx.GenHash()
	ok, err := core.BlockChainImpl.GetTransactionPool().AddTransaction(tx)
	if !ok {
		return failResult(err.Error())
	}
	return successResult(nil)
}

func (api *GtasAPI) MinerQuery(mtype int32) (*Result, error) {
	minerInfo := mediator.Proc.GetMinerInfo()
	address := common.BytesToAddress(minerInfo.ID.Serialize())
	miner := core.MinerManagerImpl.GetMinerById(address[:], byte(mtype), nil)
	js,err := json.Marshal(miner)
	if err != nil {
		return &Result{Message:err.Error(), Data:nil}, err
	}
	return &Result{Message:address.GetHexString(),Data:string(js)}, nil
}

func (api *GtasAPI) MinerAbort(mtype int32) (*Result, error) {
	minerInfo := mediator.Proc.GetMinerInfo()
	address := common.BytesToAddress(minerInfo.ID.Serialize())
	nonce := time.Now().UnixNano()
	tx := &types.Transaction{
		Nonce: uint64(nonce),
		Data: []byte{byte(mtype)},
		Source: &address,
		Type: types.TransactionTypeMinerAbort,
	}
	tx.Hash = tx.GenHash()
	ok, err := core.BlockChainImpl.GetTransactionPool().AddTransaction(tx)
	if !ok {
		return failResult(err.Error())
	}
	return successResult(nil)
}

func (api *GtasAPI) MinerRefund(mtype int32) (*Result, error) {
	minerInfo := mediator.Proc.GetMinerInfo()
	address := common.BytesToAddress(minerInfo.ID.Serialize())
	nonce := time.Now().UnixNano()
	tx := &types.Transaction{
		Nonce: uint64(nonce),
		Data: []byte{byte(mtype)},
		Source: &address,
		Type: types.TransactionTypeMinerRefund,
	}
	tx.Hash = tx.GenHash()
	ok, err := core.BlockChainImpl.GetTransactionPool().AddTransaction(tx)
	if !ok {
		return &Result{Message:err.Error(), Data:nil}, nil
	}
	return &Result{Message:"success"}, nil
}

//铸块统计
func (api *GtasAPI) CastStat(begin uint64, end uint64) (*Result, error) {
	proposerStat := make(map[string]int32)
	groupStat := make(map[string]int32)

	chain := core.BlockChainImpl
	if end == 0 {
		end = chain.QueryTopBlock().Height
	}

	for h := begin; h < end; h++ {
		bh := chain.QueryBlockByHeight(h)
		if bh == nil {
			continue
		}
		p := string(bh.Castor)
		if v, ok := proposerStat[p]; ok {
			proposerStat[p] = v+1
		} else {
			proposerStat[p] = 1
		}
		g := string(bh.GroupId)
		if v, ok := groupStat[g]; ok {
			groupStat[g] = v+1
		} else {
			groupStat[g] = 1
		}
	}
	pmap := make(map[string]int32)
	gmap := make(map[string]int32)

	for key, v := range proposerStat {
		id := groupsig.DeserializeId([]byte(key))
		pmap[id.GetHexString()] = v
	}
	for key, v := range groupStat {
		id := groupsig.DeserializeId([]byte(key))
		gmap[id.GetHexString()] = v
	}
	ret := make(map[string]map[string]int32)
	ret["proposer"] = pmap
	ret["group"] = gmap
	return successResult(ret)
}

func (api *GtasAPI) NodeInfo() (*Result, error) {
	ni := &NodeInfo{}
	p := mediator.Proc
	ac := p.MainChain.(core.AccountRepository)
	ni.ID = p.GetMinerID().GetHexString()
	bi := ac.GetBalance(p.GetMinerID().ToAddress())
	if bi != nil {
		ni.Balance = bi.Uint64()
	}
	if !p.Ready() {
		ni.Status = "节点未准备就绪"
	} else {
		ni.Status = "运行中"
		morts := make([]MortGage, 0)
		t := "--"
		heavyInfo := core.MinerManagerImpl.GetMinerById(p.GetMinerID().Serialize(), types.MinerTypeHeavy, nil)
		if heavyInfo != nil {
			morts = append(morts, *NewMortGageFromMiner(heavyInfo))
			if heavyInfo.AbortHeight == 0 {
				t = "重节点"
			}
		}
		lightInfo := core.MinerManagerImpl.GetMinerById(p.GetMinerID().Serialize(), types.MinerTypeLight, nil)
		if lightInfo != nil {
			morts = append(morts, *NewMortGageFromMiner(lightInfo))
			if lightInfo.AbortHeight == 0 {
				t += " 轻节点"
			}
		}
		ni.NType = t
		ni.MortGages = morts

		wg, ag := p.GetJoinedWorkGroupNums()
		ni.WGroupNum = wg
		ni.AGroupNum = ag

		if txs := core.BlockChainImpl.GetTransactionPool().GetReceived(); txs != nil {
			ni.TxPoolNum = len(txs)
		}
	}
	return successResult(ni)

}

func (api *GtasAPI) PageGetBlocks(page, limit int) (*Result, error) {
	chain := core.BlockChainImpl
	total := chain.Height()+1
	pageObject := PageObjects{
		Total: total,
		Data: make([]interface{}, 0),
	}
	if page < 1 {
		page = 1
	}
	i := 0
	num := uint64((page - 1)* limit)
	if total < num {
		return successResult(pageObject)
	}
	b := int64(total - num)

	for i < limit && b >= 0 {
		bh := chain.QueryBlockByHeight(uint64(b))
		b--
		if bh == nil {
			continue
		}
		block := &Block{
			Height: bh.Height,
			Hash: bh.Hash,
			PreHash: bh.PreHash,
			CurTime: bh.CurTime,
			PreTime: bh.PreTime,
			Castor: groupsig.DeserializeId(bh.Castor),
			GroupID: groupsig.DeserializeId(bh.GroupId),
			Prove: bh.ProveValue,
			Txs: bh.Transactions,
		}
		pageObject.Data = append(pageObject.Data, block)
		i++
	}
	return successResult(pageObject)
}

func (api *GtasAPI) PageGetGroups(page, limit int) (*Result, error) {
	chain := core.GroupChainImpl
	total := chain.Count()
	pageObject := PageObjects{
		Total: total,
		Data: make([]interface{}, 0),
	}

	i := 0
	b := int64(0)
	if page < 1 {
		page = 1
	}
	num := uint64((page - 1)* limit)
	if total < num {
		return successResult(pageObject)
	}
	b = int64(total - num)

	for i < limit && b >= 0 {
		g := chain.GetGroupByHeight(uint64(b))
		b--
		if g == nil {
			continue
		}

		mems := make([]string, 0)
		for _, mem := range g.Members {
			mems = append(mems, groupsig.DeserializeId(mem.Id).ShortS())
		}

		group := &Group{
			Height: uint64(b+1),
			Id: groupsig.DeserializeId(g.Id),
			PreId: groupsig.DeserializeId(g.PreGroup),
			ParentId: groupsig.DeserializeId(g.Parent),
			BeginHeight: g.BeginHeight,
			DismissHeight: g.DismissHeight,
			Members: mems,
		}
		pageObject.Data = append(pageObject.Data, group)
		i++
	}
	return successResult(pageObject)
}

func (api *GtasAPI) BlockDetail(h string) (*Result, error) {
	chain := core.BlockChainImpl
	bh := chain.QueryBlockByHash(common.HexToHash(h))
	block := &Block{
		Height: bh.Height,
		Hash: bh.Hash,
		PreHash: bh.PreHash,
		CurTime: bh.CurTime,
		PreTime: bh.PreTime,
		Castor: groupsig.DeserializeId(bh.Castor),
		GroupID: groupsig.DeserializeId(bh.GroupId),
		Prove: bh.ProveValue,
		Txs: bh.Transactions,
	}
	bonus := chain.GetBonusManager().GetBonusTransactionByBlockHash(bh.Hash.Bytes())
	var bonusHash common.Hash
	if bonus != nil {
		bonusHash = bonus.Hash
	}
	bd := &BlockDetail{
		Block: *block,
		TxCnt: len(block.Txs),
		BonusHash: bonusHash,
		Signature: *groupsig.DeserializeSign(bh.Signature),
		Random: *groupsig.DeserializeSign(bh.Random),
	}
	return successResult(bd)
}

func (api *GtasAPI) TransDetail(h string) (*Result, error) {
	tx, err := core.BlockChainImpl.GetTransactionByHash(common.HexToHash(h))
	if err != nil {
		return failResult(err.Error())
	}
	if tx != nil {
		trans := &Transaction{
			Hash: tx.Hash,
			Source: tx.Source,
			Target: tx.Target,
			Type: tx.Type,
			GasLimit: tx.GasLimit,
			GasPrice: tx.GasPrice,
			Data: tx.Data,
			ExtraData: tx.ExtraData,
			ExtraDataType: tx.ExtraDataType,
			Nonce: tx.Nonce,
			Value: tx.Value,
		}
		return successResult(trans)
	}
	return successResult(nil)
}

func (api *GtasAPI) Dashboard() (*Result, error) {
    blockHeight := core.BlockChainImpl.Height()
    groupHeight := core.GroupChainImpl.Count()
    workNum := len(mediator.Proc.GetCastQualifiedGroups(blockHeight))
    nodeResult, _ := api.NodeInfo()
    consResult, _ := api.ConnectedNodes()
    dash := &Dashboard{
    	BlockHeight: blockHeight,
    	GroupHeight: groupHeight,
    	WorkGNum: workNum,
    	NodeInfo: nodeResult.Data.(*NodeInfo),
    	Conns: consResult.Data.([]ConnInfo),
	}
	return successResult(dash)
}