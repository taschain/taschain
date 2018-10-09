package cli

import (
	"consensus/mediator"
	"middleware/types"
	"core"
	"common"
	"github.com/vmihailenco/msgpack"
	"consensus/groupsig"
)

/*
**  Creator: pxf
**  Date: 2018/9/30 下午4:34
**  Description: 
*/

func (api *GtasAPI) MinerApply(stake uint64, mtype int32) (*Result, error) {
	minerInfo := mediator.Proc.GetMinerInfo()
	address := common.BytesToAddress(minerInfo.ID.Serialize())
	nonce := core.BlockChainImpl.GetNonce(address)

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
		Nonce: nonce,
		Data: data,
		Source: &address,
		Value: stake,
		Type: types.TransactionTypeMinerApply,
	}
	tx.Hash = tx.GenHash()
	ok, err := core.BlockChainImpl.GetTransactionPool().Add(tx)
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
	return &Result{Message:"success", Data:ret}, nil
}