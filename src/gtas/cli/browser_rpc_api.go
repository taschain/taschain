package cli

import (
	"common"
	"consensus/groupsig"
	"consensus/mediator"
	"consensus/model"
	"core"
	"github.com/pmylund/sortutil"
	"middleware/types"
	"tns"
)

// 区块链浏览器
// 账号信息查询
func (api *GtasAPI) ExplorerAccount(hash string) (*Result, error) {

	accoundDb := core.BlockChainImpl.LatestStateDB()
	if accoundDb == nil {
		return nil, nil
	}
	address := common.HexToAddress(hash)
	if !accoundDb.Exist(address) {
		return failResult("Account not Exist!")
	}
	account := ExplorerAccount{}
	account.Balance = accoundDb.GetBalance(address)
	account.Nonce = accoundDb.GetNonce(address)
	account.CodeHash = accoundDb.GetCodeHash(address).String()
	account.Code = string(accoundDb.GetCode(address)[:])
	account.Type = 0
	if len(account.Code) > 0 {
		account.Type = 1
		account.StateData = make(map[string]interface{})

		iter := accoundDb.DataIterator(common.HexToAddress(hash), "")
		for iter.Next() {
			k := string(iter.Key[:])
			v := string(iter.Value[:])
			account.StateData[k] = v

		}
	}
	return successResult(account)
}

//区块链浏览器
////查询铸块和分红信息
//func (api *GtasAPI) CastBlockAndBonusStat(height uint64) (*Result, error) {
//	var bonusValueMap, bonusNumMap, castBlockNumMap map[string]uint64
//
//	if _, ok := BonusValueStatMap[height]; !ok {
//		var i uint64 = 1
//		for ; i <= height; i++ {
//			if _, ok := BonusValueStatMap[i]; !ok {
//				break
//			}
//		}
//
//		for j := i; j <= height; j++ {
//			bh := core.BlockChainImpl.QueryBlockByHeight(j)
//
//			bonusValuePreMap := BonusValueStatMap[j-1]
//			bonusNumPreMap := BonusNumStatMap[j-1]
//			castBlockPreMap := CastBlockStatMap[j-1]
//
//			// 获取验证分红的交易信息
//			// 此方法取到的分红交易有时候为空
//			var bonusTx *types.Transaction
//			if bonusTx = core.BlockChainImpl.GetBonusManager().GetBonusTransactionByBlockHash(bh.Hash.Bytes()); bonusTx == nil || bh.Castor == nil {
//				BonusLogger.Infof("[Bonus or Castor is NIL] height: %v, blockHash: %v", j, bh.Hash.ShortS())
//				BonusValueStatMap[j] = bonusValuePreMap
//				BonusNumStatMap[j] = bonusNumPreMap
//				CastBlockStatMap[j] = castBlockPreMap
//				continue
//			}
//
//			// 从交易信息中解析出targetId列表
//			_, memIds, _, value := mediator.Proc.MainChain.GetBonusManager().ParseBonusTransaction(bonusTx)
//
//			bonusValueCurrentMap := make(map[string]uint64)
//			bonusNumCurrentMap := make(map[string]uint64)
//			castBlockCurrentMap := make(map[string]uint64)
//
//			for k, v := range bonusValuePreMap {
//				bonusValueCurrentMap[k] = v
//			}
//
//			for k, v := range bonusNumPreMap {
//				bonusNumCurrentMap[k] = v
//			}
//
//			for k, v := range castBlockPreMap {
//				castBlockCurrentMap[k] = v
//			}
//
//			for _, mv := range memIds {
//				memId := groupsig.DeserializeId(mv).GetHexString()
//
//				if v, ok := bonusValueCurrentMap[memId]; ok {
//					bonusValueCurrentMap[memId] = value + v
//					if v, ok := bonusNumCurrentMap[memId]; ok {
//						bonusNumCurrentMap[memId] = v + 1
//					} else {
//						bonusNumCurrentMap[memId] = 1
//					}
//				} else {
//					bonusValueCurrentMap[memId] = value
//					bonusNumCurrentMap[memId] = 1
//				}
//			}
//
//			casterId := groupsig.DeserializeId(bh.Castor)
//			if v, ok := castBlockCurrentMap[casterId.GetHexString()]; ok {
//				castBlockCurrentMap[casterId.GetHexString()] = v + 1
//			} else {
//				castBlockCurrentMap[casterId.GetHexString()] = 1
//			}
//
//			BonusValueStatMap[j] = bonusValueCurrentMap
//			BonusNumStatMap[j] = bonusNumCurrentMap
//			CastBlockStatMap[j] = castBlockCurrentMap
//		}
//	}
//
//	bonusValueMap = BonusValueStatMap[height]
//	bonusNumMap = BonusNumStatMap[height]
//	castBlockNumMap = CastBlockStatMap[height]
//
//	bonusStatResults := make([]BonusStatInfo, 0, 10)
//	lightMinerIter := core.MinerManagerImpl.MinerIterator(types.MinerTypeHeavy, nil)
//	for lightMinerIter.Next() {
//		miner, _ := lightMinerIter.Current()
//		minerId := groupsig.DeserializeId(miner.Id)
//		bonusStatItem := BonusStatInfo{
//			MemberId:        minerId.ShortS(),
//			MemberIdW:       minerId.GetHexString(),
//			BonusNum:        bonusNumMap[minerId.GetHexString()],
//			TotalBonusValue: bonusValueMap[minerId.GetHexString()],
//		}
//
//		bonusStatResults = append(bonusStatResults, bonusStatItem)
//	}
//
//	castBlockResults := make([]CastBlockStatInfo, 0, 10)
//	heavyIter := core.MinerManagerImpl.MinerIterator(types.MinerTypeHeavy, nil)
//	for heavyIter.Next() {
//		miner, _ := heavyIter.Current()
//		minerId := groupsig.DeserializeId(miner.Id)
//		castBlockItem := CastBlockStatInfo{
//			CasterId:     minerId.ShortS(),
//			CasterIdW:    minerId.GetHexString(),
//			Stake:        miner.Stake,
//			CastBlockNum: castBlockNumMap[minerId.GetHexString()],
//		}
//		castBlockResults = append(castBlockResults, castBlockItem)
//	}
//
//	bonusInfo := bonusStatByHeight(height)
//
//	result := CastBlockAndBonusResult{
//		BonusInfoAtHeight:  bonusInfo,
//		BonusStatInfos:     bonusStatResults,
//		CastBlockStatInfos: castBlockResults,
//	}
//
//	return successResult(result)
//}

//区块链浏览器
//查询块详情
func (api *GtasAPI) ExplorerBlockDetail(height uint64) (*Result, error) {
	chain := core.BlockChainImpl
	b := chain.QueryBlockCeil(height)
	if b == nil {
		return failResult("QueryBlock error")
	}
	bh := b.Header
	block := convertBlockHeader(bh)

	trans := make([]Transaction, 0)

	for _, tx := range b.Transactions {
		trans = append(trans, *convertTransaction(tx))
	}

	evictedReceipts := make([]*types.Receipt, 0)
	//for _, tx := range bh.EvictedTxs {
	//	wrapper := chain.GetTransactionPool().GetReceipt(tx)
	//	if wrapper != nil {
	//		evictedReceipts = append(evictedReceipts, wrapper)
	//	}
	//}
	receipts := make([]*types.Receipt, len(bh.Transactions))
	for i, tx := range bh.Transactions {
		wrapper := chain.GetTransactionPool().GetReceipt(tx)
		if wrapper != nil {
			receipts[i] = wrapper
		}
	}

	bd := &ExplorerBlockDetail{
		BlockDetail:     BlockDetail{Block: *block, Trans: trans},
		EvictedReceipts: evictedReceipts,
		Receipts:        receipts,
	}
	return successResult(bd)
}


//区块链浏览器
//查询组信息
func (api *GtasAPI) ExplorerGroupsAfter(height uint64) (*Result, error) {
	groups := core.GroupChainImpl.GetGroupsAfterHeight(height, common.MaxInt64)

	ret := make([]map[string]interface{}, 0)
	h := height
	for _, g := range groups {
		gmap := explorerConvertGroup(g)
		gmap["height"] = h
		h++
		ret = append(ret, gmap)
	}
	return successResult(ret)
}

//账号绑定查询
func (api *GtasAPI)TnsGetAddress(account string) (*Result, error) {
	accoundDb := core.BlockChainImpl.LatestStateDB()
	address := tns.GetAddressByAccount(accoundDb,account)
	return successResult(address)
}

func explorerConvertGroup(g *types.Group) map[string]interface{} {
	gmap := make(map[string]interface{})
	if g.Id != nil && len(g.Id) != 0 {
		gmap["id"] = groupsig.DeserializeId(g.Id).GetHexString()
		gmap["hash"] = g.Header.Hash
	}
	gmap["parent_id"] = groupsig.DeserializeId(g.Header.Parent).GetHexString()
	gmap["pre_id"] = groupsig.DeserializeId(g.Header.PreGroup).GetHexString()
	gmap["begin_time"] = g.Header.BeginTime
	gmap["create_height"] = g.Header.CreateHeight
	gmap["work_height"] = g.Header.WorkHeight
	gmap["dismiss_height"] = g.Header.DismissHeight
	mems := make([]string, 0)
	for _, mem := range g.Members {
		memberStr := groupsig.DeserializeId(mem).GetHexString()
		mems = append(mems, memberStr)
	}
	gmap["members"] = mems
	return gmap
}

func (api *GtasAPI) ExplorerBlockBonus(height uint64) (*Result, error) {
	chain := core.BlockChainImpl
    b := chain.QueryBlockCeil(height)
	if b == nil {
		return failResult("nil block")
	}
	bh := b.Header

	ret := &ExploreBlockBonus{
		ProposalId: groupsig.DeserializeId(bh.Castor).GetHexString(),
	}
	bonusNum := uint64(0)
	if b.Transactions != nil {
		for _, tx := range b.Transactions {
			if tx.Type == types.TransactionTypeBonus {
				bonusNum++
			}
		}
	}
	ret.ProposalBonus = model.Param.ProposalBonus + bonusNum*model.Param.PackBonus
	if bonusTx := chain.GetBonusManager().GetBonusTransactionByBlockHash(bh.Hash.Bytes()); bonusTx != nil {
		genBonus := convertBonusTransaction(bonusTx)
		genBonus.Success = true
		ret.VerifierBonus = *genBonus
	}
	return successResult(ret)
}

//监控平台调用块同步
func (api *GtasAPI) MonitorBlocks(begin, end uint64) (*Result, error) {
	chain := core.BlockChainImpl
	if begin > end {
		end = begin
	}
	var pre *types.Block

	blocks := make([]*BlockDetail, 0)
	for h := begin; h <= end; h++ {
		b := chain.QueryBlockCeil(h)
		if b == nil {
			continue
		}
		bh := b.Header
		block := convertBlockHeader(bh)

		if pre == nil {
			pre = chain.QueryBlockByHash(bh.PreHash)
		}
		if pre == nil {
			block.Qn = bh.TotalQN
		} else {
			block.Qn = bh.TotalQN-pre.Header.TotalQN
		}

		trans := make([]Transaction, 0)

		for _, tx := range b.Transactions {
			trans = append(trans, *convertTransaction(tx))
		}

		bd := &BlockDetail{
			Block:     *block,
			Trans: trans,
		}
		pre = b
		blocks = append(blocks, bd)
	}
	return successResult(blocks)
}

func (api *GtasAPI) MonitorNodeInfo() (*Result, error) {
	bh := core.BlockChainImpl.Height()
	gh := core.GroupChainImpl.LastGroup().GroupHeight

	ni := &NodeInfo{}

	ret, _ := api.NodeInfo()
	if ret != nil && ret.IsSuccess() {
		ni = ret.Data.(*NodeInfo)
	}
	ni.BlockHeight = bh
	ni.GroupHeight = gh
	if ni.MortGages != nil {
		for _, mg := range ni.MortGages {
			if mg.Type == "重节点" {
				ni.VrfThreshold = mediator.Proc.GetVrfThreshold(common.TAS2RA(mg.Stake))
				break
			}
		}
	}
	return successResult(ni)
}

func (api *GtasAPI) MonitorAllMiners() (*Result, error)  {
    miners := mediator.Proc.GetAllMinerDOs()
    totalStake := uint64(0)
    maxStake := uint64(0)
	for _, m := range miners {
		if m.AbortHeight == 0 && m.NType == types.MinerTypeHeavy {
			totalStake += m.Stake
			if maxStake < m.Stake {
				maxStake = m.Stake
			}
		}
	}
	sortutil.AscByField(miners, "Stake")
	data := make(map[string]interface{})
	data["miners"] = miners
	data["maxStake"] = maxStake
	data["totalStake"] = totalStake
    return successResult(data)
}
