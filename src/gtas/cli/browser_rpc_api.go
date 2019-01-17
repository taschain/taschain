package cli

import (
	"core"
	"middleware/types"
	"consensus/groupsig"
	"common"
	"consensus/model"
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
	b := chain.QueryBlock(height)
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
	for _, tx := range bh.EvictedTxs {
		wrapper := chain.GetTransactionPool().GetExecuted(tx)
		if wrapper != nil {
			evictedReceipts = append(evictedReceipts, wrapper.Receipt)
		}
	}
	receipts := make([]*types.Receipt, len(bh.Transactions))
	for i, tx := range bh.Transactions {
		wrapper := chain.GetTransactionPool().GetExecuted(tx)
		if wrapper != nil {
			receipts[i] = wrapper.Receipt
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
	groups, err := core.GroupChainImpl.GetGroupsByHeight(height)
	if err != nil {
		return failResult("no more group")
	}
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
    b := chain.QueryBlock(height)
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