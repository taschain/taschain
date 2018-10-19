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
	"fmt"
	"math/big"
	"log"
	types2 "storage/core/types"
	"consensus/model"
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
		block := convertBlockHeader(bh)
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
	b := chain.QueryBlockByHash(common.HexToHash(h))
	if b == nil {
		return successResult(nil)
	}
	bh := b.Header
	block := convertBlockHeader(bh)

	preBH := chain.QueryBlockHeaderByHash(bh.PreHash)

	castor := block.Castor.GetHexString()

	trans := make([]Transaction, 0)
	bonusTxs := make([]BonusTransaction, 0)
	minerBonus := make(map[string]*MinerBonusBalance)
	uniqueBonusBlockHash := make(map[common.Hash]byte)
	minerVerifyBlockHash := make(map[string][]common.Hash)
	blockVerifyBonus := make(map[common.Hash]uint64)

	minerBonus[castor] = genMinerBalance(block.Castor, bh)

	for _, tx := range b.Transactions {
		if tx.Type == types.TransactionTypeBonus {
			btx := *convertBonusTransaction(tx)
			if st, err := mediator.Proc.MainChain.GetTransactionPool().GetTransactionStatus(tx.Hash); err != nil {
				log.Printf("getTransactions statue error, hash %v, err %v", tx.Hash.Hex(), err)
				btx.StatusReport = "获取状态错误" + err.Error()
			} else {
				if st == types2.ReceiptStatusSuccessful {
					btx.StatusReport = "成功"
					btx.Success = true
				} else {
					btx.StatusReport = "失败"
				}
			}
			bonusTxs = append(bonusTxs, btx)
			blockVerifyBonus[btx.BlockHash] = btx.Value
			for _, tid := range btx.TargetIDs {
				if _, ok := minerBonus[tid.GetHexString()]; !ok {
					minerBonus[tid.GetHexString()] = genMinerBalance(tid, bh)
				}
				if !btx.Success {
					continue
				}
				if hs, ok := minerVerifyBlockHash[tid.GetHexString()]; ok {
					find := false
					for _, h := range hs {
						if h == btx.BlockHash {
							find = true
							break
						}
					}
					if !find {
						hs = append(hs, btx.BlockHash)
						minerVerifyBlockHash[tid.GetHexString()] = hs
					}
				} else {
					hs = make([]common.Hash, 0)
					hs = append(hs, btx.BlockHash)
					minerVerifyBlockHash[tid.GetHexString()] = hs
				}
			}
			if btx.Success {
				uniqueBonusBlockHash[btx.BlockHash] = 1
			}
		} else {
			trans = append(trans, *convertTransaction(tx))
		}
	}

	mbs := make([]*MinerBonusBalance, 0)
	for id, mb := range minerBonus {
		mb.Explain = ""
		increase := uint64(0)
		if id == castor {
			mb.Proposal = true
			mb.PackBonusTx = len(uniqueBonusBlockHash)
			increase += model.Param.ProposalBonus + uint64(mb.PackBonusTx) * model.Param.PackBonus
			mb.Explain = fmt.Sprintf("提案 打包分红交易%v个", mb.PackBonusTx)
		}
		if hs, ok := minerVerifyBlockHash[id]; ok {
			for _, h := range hs {
				increase += blockVerifyBonus[h]
			}
			mb.VerifyBlock = len(hs)
			mb.Explain = fmt.Sprintf("%v 验证%v块", mb.Explain, mb.VerifyBlock)
		}
		mb.ExpectBalance = new(big.Int).SetUint64(mb.PreBalance.Uint64() + increase)
		mbs = append(mbs, mb)
	}

	var genBonus *BonusTransaction
	if bonusTx := chain.GetBonusManager().GetBonusTransactionByBlockHash(bh.Hash.Bytes()); bonusTx != nil {
		genBonus = convertBonusTransaction(bonusTx)
	}

	bd := &BlockDetail{
		Block:        *block,
		GenBonusTx:   genBonus,
		Trans:        trans,
		BodyBonusTxs: bonusTxs,
		MinerBonus:   mbs,
		PreTotalQN:   preBH.TotalQN,
	}
	return successResult(bd)
}

func (api *GtasAPI) TransDetail(h string) (*Result, error) {
	tx, err := core.BlockChainImpl.GetTransactionByHash(common.HexToHash(h))
	if err != nil {
		return failResult(err.Error())
	}
	if tx != nil {
		trans := convertTransaction(tx)
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