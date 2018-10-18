package cli

import (
	"bytes"
	"consensus/mediator"
	"math/big"
	"middleware/types"
	"core"
	"common"
	"github.com/vmihailenco/msgpack"
	"consensus/groupsig"
	"taslog"
	"time"
	"encoding/json"
)

/*
**  Creator: pxf
**  Date: 2018/9/30 下午4:34
**  Description: 
*/
var BonusLogger = taslog.GetLogger(taslog.BonusStatConfig)

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

func (api *GtasAPI) ConsensusStat(height uint64)  (*Result, error){
	core.BlockChainImpl.GetBonusManager().StatBonusByBlockHeight(height)
	return &Result{Message:"success", Data:nil}, nil
}

func bonusStatByHeight(height uint64)  BonusInfo{
	bh := core.BlockChainImpl.QueryBlockByHeight(height)
	casterId := bh.Castor
	groupId := bh.GroupId

	// 获取验证分红的交易信息
	bonusTx := core.BlockChainImpl.GetBonusManager().GetBonusTransactionByBlockHash(bh.Hash.Bytes())

	// 从交易信息中解析出targetId列表
	reader := bytes.NewReader(bonusTx.ExtraData)
	groupIdExtra := make([]byte,common.GroupIdLength)
	addr := make([]byte,common.AddressLength)

	// 分配给每一个验证节点的分红交易
	value := big.NewInt(int64(bonusTx.Value))

	if n,_ := reader.Read(groupIdExtra);n != common.GroupIdLength{
		panic("TVMExecutor Read GroupId Fail")
	}

	mems := make([]string, 0)

	for n,_ := reader.Read(addr);n > 0;n,_ = reader.Read(addr){
		mems = append(mems, groupsig.DeserializeId(addr).ShortS())
	}

	data := BonusInfo{
		BlockHeight:height,
		BlockHash:bh.Hash,
		BonusTxHash:bonusTx.Hash,
		GroupId:groupsig.DeserializeId(groupId).ShortS(),
		CasterId:groupsig.DeserializeId(casterId).ShortS(),
		MemberIds:mems,
		BonusValue:value.Uint64(),
	}

	return data
}

func (api *GtasAPI) CastBlockAndBonusStat(height uint64) (*Result, error){
	var bonusValueMap, bonusNumMap, castBlockNumMap map[string]uint64

	if _, ok := BonusValueStatMap[height]; !ok{
		var i uint64
		for i = 1; i < height; i++{
			if _, ok := BonusValueStatMap[i]; !ok{
				break
			}
		}

		BonusLogger.Infof("i : %v", i)

		for j := i; j <= height; j++{
			bh := core.BlockChainImpl.QueryBlockByHeight(j)

			casterId := groupsig.DeserializeId(bh.Castor)

			BonusLogger.Infof("height :%v | casterId: %v", j, casterId.ShortS())

			// 获取验证分红的交易信息
			bm := core.BlockChainImpl.GetBonusManager();

			bonusTx := bm.GetBonusTransactionByBlockHash(bh.Hash.Bytes())
			if bonusTx == nil {
				BonusLogger.Infof("bonux tx is null")
				continue
			}

			BonusLogger.Infof("tx hash:", bonusTx.Hash.String())
			BonusLogger.Infof("tx type:", bonusTx.Type)
			BonusLogger.Infof("tx bonusTx.ExtraData :", bonusTx.ExtraData)


			// 从交易信息中解析出targetId列表
			reader := bytes.NewReader(bonusTx.ExtraData)
			groupIdExtra := make([]byte,common.GroupIdLength)
			addr := make([]byte,common.AddressLength)

			// 分配给每一个验证节点的分红交易
			value := big.NewInt(int64(bonusTx.Value))

			if n,_ := reader.Read(groupIdExtra);n != common.GroupIdLength{
				panic("TVMExecutor Read GroupId Fail")
			}

			bonusValuePreMap := BonusValueStatMap[j - 1]
			bonusNumPreMap := BonusNumStatMap[j - 1]
			castBlockPreMap := CastBlockStatMap[j - 1]

			bonusValueCurrentMap := make(map[string]uint64, 10)
			bonusNumCurrentMap := make(map[string]uint64, 10)
			castBlockCurrentMap := make(map[string]uint64, 10)

			for k,v := range bonusValuePreMap {
				bonusValueCurrentMap[k] = v
			}

			for k,v := range bonusNumPreMap {
				bonusNumCurrentMap[k] = v
			}

			for k,v := range castBlockPreMap {
				castBlockCurrentMap[k] = v
			}

			for n,_ := reader.Read(addr);n > 0;n,_ = reader.Read(addr){
				memId := groupsig.DeserializeId(addr).GetHexString()

				BonusLogger.Infof("memberId: %v", memId)

				if v, ok := bonusValueCurrentMap[memId]; ok{
					bonusValueCurrentMap[memId] = value.Uint64() + v
					if v, ok := bonusNumCurrentMap[memId]; ok {
						bonusNumCurrentMap[memId] = v + 1
					} else {
						bonusNumCurrentMap[memId] = 1
					}
				} else {
					bonusValueCurrentMap[memId] = value.Uint64()
					bonusNumCurrentMap[memId] = 1
				}
			}

			if v,ok := castBlockCurrentMap[casterId.GetHexString()];ok{
				castBlockCurrentMap[casterId.GetHexString()] += v
			} else {
				castBlockCurrentMap[casterId.GetHexString()] = 1
			}

			BonusValueStatMap[j] = bonusValueCurrentMap
			BonusNumStatMap[j] = bonusNumCurrentMap
			CastBlockStatMap[j] = castBlockCurrentMap
		}
	}

	bonusValueMap =  BonusValueStatMap[height]
	bonusNumMap = BonusNumStatMap[height]
	castBlockNumMap = CastBlockStatMap[height]

	BonusLogger.Infof("bonus:%v | num:%v | castBlock:%v", bonusValueMap, bonusNumMap, castBlockNumMap)

	group := core.GroupChainImpl.GetGroupByHeight(height)
	bonusStatResults := make([]BonusStatInfo,10)
	for _, mem := range group.Members {
		memId := groupsig.DeserializeId(mem.Id)

		bonusStatItem := BonusStatInfo{
			MemberId:memId.ShortS(),
			BonusNum:bonusNumMap[memId.GetHexString()],
			TotalBonusValue:bonusValueMap[memId.GetHexString()],
		}

		bonusStatResults = append(bonusStatResults, bonusStatItem)
	}

	castBlockResults := make([]CastBlockStatInfo, 10)
	iter := core.MinerManagerImpl.MinerIterator(types.MinerTypeHeavy, nil)
	for iter.Next() {
		miner, _ := iter.Current()
		minerId := groupsig.DeserializeId(miner.Id)
		castBlockItem := CastBlockStatInfo{
			CasterId: minerId.ShortS(),
			Stake:miner.Stake,
			CastBlockNum:castBlockNumMap[minerId.GetHexString()],
		}
		castBlockResults = append(castBlockResults, castBlockItem)
	}

	bonusInfo := bonusStatByHeight(height)

	result := CastBlockAndBonusResult{
		BonusInfoAtHeight:bonusInfo,
		BonusStatInfos:bonusStatResults,
		CastBlockStatInfos:castBlockResults,
	}

	return successResult(result)
}