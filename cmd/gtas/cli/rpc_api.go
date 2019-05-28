package cli

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/mediator"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/core"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/network"
	"github.com/taschain/taschain/taslog"
	"github.com/vmihailenco/msgpack"
	"log"
	"math"
	"math/big"
	"strconv"
	"time"
)

/*
**  Creator: pxf
**  Date: 2018/9/30 下午4:34
**  Description:
 */
var BonusLogger taslog.Logger

func successResult(data interface{}) (*Result, error) {
	return &Result{
		Message: "success",
		Data:    data,
		Status:  0,
	}, nil
}
func failResult(err string) (*Result, error) {
	return &Result{
		Message: err,
		Data:    nil,
		Status:  -1,
	}, nil
}

// GtasAPI is a single-method API handler to be returned by test services.
type GtasAPI struct {
}

// T 用户交易接口
func (api *GtasAPI) Tx(txRawjson string) (*Result, error) {
	var txRaw = new(txRawData)
	if err := json.Unmarshal([]byte(txRawjson), txRaw); err != nil {
		return failResult(err.Error())
	}

	trans := txRawToTransaction(txRaw)

	trans.Hash = trans.GenHash()

	if err := sendTransaction(trans); err != nil {
		return failResult(err.Error())
	}

	return successResult(trans.Hash.String())
}

// Balance 查询余额接口
func (api *GtasAPI) Balance(account string) (*Result, error) {
	balance, err := walletManager.getBalance(account)
	if err != nil {
		return nil, err
	}
	return &Result{
		Message: fmt.Sprintf("The balance of account: %s is %v TAS", account, balance),
		Data:    fmt.Sprintf("%v", balance),
	}, nil
}

// NewWallet 新建账户接口
func (api *GtasAPI) NewWallet() (*Result, error) {
	privKey, addr := walletManager.newWallet()
	data := make(map[string]string)
	data["private_key"] = privKey
	data["address"] = addr
	return successResult(data)
}

// GetWallets 获取当前节点的wallets
func (api *GtasAPI) GetWallets() (*Result, error) {
	return successResult(walletManager)
}

// DeleteWallet 删除本地节点指定序号的地址
func (api *GtasAPI) DeleteWallet(key string) (*Result, error) {
	walletManager.deleteWallet(key)
	return successResult(walletManager)
}

// BlockHeight 块高查询
func (api *GtasAPI) BlockHeight() (*Result, error) {
	height := core.BlockChainImpl.QueryTopBlock().Height
	return successResult(height)
}

// GroupHeight 组块高查询
func (api *GtasAPI) GroupHeight() (*Result, error) {
	height := core.GroupChainImpl.Height()
	return successResult(height)
}

// Vote
func (api *GtasAPI) Vote(from string, v *VoteConfig) (*Result, error) {
	//config := v.ToGlobal()
	//walletManager.newVote(from, config)
	return successResult(nil)
}

// ConnectedNodes 查询已链接的node的信息
func (api *GtasAPI) ConnectedNodes() (*Result, error) {

	nodes := network.GetNetInstance().ConnInfo()
	conns := make([]ConnInfo, 0)
	for _, n := range nodes {
		conns = append(conns, ConnInfo{Id: n.Id, Ip: n.Ip, TcpPort: n.Port})
	}
	return successResult(conns)
}

// TransPool 查询缓冲区的交易信息。
func (api *GtasAPI) TransPool() (*Result, error) {
	transactions := core.BlockChainImpl.GetTransactionPool().GetReceived()
	transList := make([]Transactions, 0, len(transactions))
	for _, v := range transactions {
		transList = append(transList, Transactions{
			Hash:   v.Hash.String(),
			Source: v.Source.GetHexString(),
			Target: v.Target.GetHexString(),
			Value:  strconv.FormatInt(int64(v.Value), 10),
		})
	}

	return successResult(transList)
}

func (api *GtasAPI) GetTransaction(hash string) (*Result, error) {
	transaction := core.BlockChainImpl.GetTransactionByHash(false, true, common.HexToHash(hash))
	if transaction == nil {
		return failResult("transaction not exists")
	}
	detail := make(map[string]interface{})
	detail["hash"] = hash
	if transaction.Source != nil {
		detail["source"] = transaction.Source.Hash().Hex()
	}
	if transaction.Target != nil {
		detail["target"] = transaction.Target.Hash().Hex()
	}
	detail["value"] = transaction.Value

	return successResult(detail)
}

func (api *GtasAPI) GetBlockByHeight(height uint64) (*Result, error) {
	b := core.BlockChainImpl.QueryBlockByHeight(height)
	if b == nil {
		return failResult("height not exists")
	}
	bh := b.Header
	preBH := core.BlockChainImpl.QueryBlockHeaderByHash(bh.PreHash)
	block := convertBlockHeader(b)
	if preBH != nil {
		block.Qn = bh.TotalQN - preBH.TotalQN
	} else {
		block.Qn = bh.TotalQN
	}
	return successResult(block)
}

func (api *GtasAPI) GetBlockByHash(hash string) (*Result, error) {
	b := core.BlockChainImpl.QueryBlockByHash(common.HexToHash(hash))
	if b == nil {
		return failResult("height not exists")
	}
	bh := b.Header
	preBH := core.BlockChainImpl.QueryBlockHeaderByHash(bh.PreHash)
	block := convertBlockHeader(b)
	if preBH != nil {
		block.Qn = bh.TotalQN - preBH.TotalQN
	} else {
		block.Qn = bh.TotalQN
	}
	return successResult(block)
}

func (api *GtasAPI) GetBlocks(from uint64, to uint64) (*Result, error) {
	blocks := make([]*Block, 0)
	var preBH *types.BlockHeader
	for h := from; h <= to; h++ {
		b := core.BlockChainImpl.QueryBlockByHeight(h)
		if b != nil {
			block := convertBlockHeader(b)
			if preBH == nil {
				preBH = core.BlockChainImpl.QueryBlockHeaderByHash(b.Header.PreHash)
			}
			if preBH == nil {
				block.Qn = b.Header.TotalQN
			} else {
				block.Qn = b.Header.TotalQN - preBH.TotalQN
			}
			preBH = b.Header
			blocks = append(blocks, block)
		}
	}
	return successResult(blocks)
}

func (api *GtasAPI) GetTopBlock() (*Result, error) {
	bh := core.BlockChainImpl.QueryTopBlock()
	b := core.BlockChainImpl.QueryBlockByHash(bh.Hash)
	bh = b.Header

	blockDetail := make(map[string]interface{})
	blockDetail["hash"] = bh.Hash.Hex()
	blockDetail["height"] = bh.Height
	blockDetail["pre_hash"] = bh.PreHash.Hex()
	blockDetail["pre_time"] = bh.PreTime().Local().Format("2006-01-02 15:04:05")
	blockDetail["total_qn"] = bh.TotalQN
	blockDetail["cur_time"] = bh.CurTime.Local().Format("2006-01-02 15:04:05")
	blockDetail["castor"] = hex.EncodeToString(bh.Castor)
	blockDetail["group_id"] = hex.EncodeToString(bh.GroupId)
	blockDetail["signature"] = hex.EncodeToString(bh.Signature)
	blockDetail["txs"] = len(b.Transactions)
	blockDetail["elapsed"] = bh.Elapsed
	blockDetail["tps"] = math.Round(float64(len(b.Transactions)) / float64(bh.Elapsed))

	blockDetail["tx_pool_count"] = len(core.BlockChainImpl.GetTransactionPool().GetReceived())
	blockDetail["tx_pool_total"] = core.BlockChainImpl.GetTransactionPool().TxNum()
	blockDetail["miner_id"] = mediator.Proc.GetPubkeyInfo().ID.ShortS()
	return successResult(blockDetail)
}

func (api *GtasAPI) WorkGroupNum(height uint64) (*Result, error) {
	groups := mediator.Proc.GetCastQualifiedGroups(height)
	return successResult(groups)
}

func convertGroup(g *types.Group) map[string]interface{} {
	gmap := make(map[string]interface{})
	if g.Id != nil && len(g.Id) != 0 {
		gmap["group_id"] = groupsig.DeserializeId(g.Id).GetHexString()
		gmap["g_hash"] = g.Header.Hash.String()
	}
	gmap["parent"] = groupsig.DeserializeId(g.Header.Parent).GetHexString()
	gmap["pre"] = groupsig.DeserializeId(g.Header.PreGroup).GetHexString()
	gmap["begin_height"] = g.Header.WorkHeight
	gmap["dismiss_height"] = g.Header.DismissHeight
	gmap["create_height"] = g.Header.CreateHeight
	gmap["create_time"] = g.Header.BeginTime
	gmap["mem_size"] = len(g.Members)
	mems := make([]string, 0)
	for _, mem := range g.Members {
		memberStr := groupsig.DeserializeId(mem).GetHexString()
		mems = append(mems, memberStr[0:6]+"-"+memberStr[len(memberStr)-6:])
	}
	gmap["members"] = mems
	gmap["extends"] = g.Header.Extends
	return gmap
}

func (api *GtasAPI) GetGroupsAfter(height uint64) (*Result, error) {
	groups := core.GroupChainImpl.GetGroupsAfterHeight(height, math.MaxInt64)

	ret := make([]map[string]interface{}, 0)
	h := height
	for _, g := range groups {
		gmap := convertGroup(g)
		gmap["height"] = h
		h++
		ret = append(ret, gmap)
	}
	return successResult(ret)
}

func (api *GtasAPI) GetCurrentWorkGroup() (*Result, error) {
	height := core.BlockChainImpl.Height()
	return api.GetWorkGroup(height)
}

func (api *GtasAPI) GetWorkGroup(height uint64) (*Result, error) {
	groups := mediator.Proc.GetCastQualifiedGroupsFromChain(height)
	ret := make([]map[string]interface{}, 0)
	h := height
	for _, g := range groups {
		gmap := convertGroup(g)
		gmap["height"] = h
		h++
		ret = append(ret, gmap)
	}
	return successResult(ret)
}

//deprecated
func (api *GtasAPI) MinerApply(sign string, bpk string, vrfpk string, stake uint64, mtype int32) (*Result, error) {
	id := IdFromSign(sign)
	address := common.BytesToAddress(id)

	info := core.MinerManagerImpl.GetMinerById(id, byte(mtype), nil)
	if info != nil {
		return failResult("已经申请过该类型矿工")
	}

	//address := common.BytesToAddress(minerInfo.ID.Serialize())
	nonce := time.Now().UnixNano()
	pbkBytes := common.FromHex(bpk)

	miner := &types.Miner{
		Id:           id,
		PublicKey:    groupsig.DeserializePubkeyBytes(pbkBytes).Serialize(),
		VrfPublicKey: common.FromHex(vrfpk),
		Stake:        stake,
		Type:         byte(mtype),
	}
	data, err := msgpack.Marshal(miner)
	if err != nil {
		return &Result{Message: err.Error(), Data: nil}, nil
	}
	tx := &types.Transaction{
		Nonce:    uint64(nonce),
		Data:     data,
		Source:   &address,
		Value:    stake,
		Type:     types.TransactionTypeMinerApply,
		GasPrice: common.MaxUint64,
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
	js, err := json.Marshal(miner)
	if err != nil {
		return &Result{Message: err.Error(), Data: nil}, err
	}
	return &Result{Message: address.GetHexString(), Data: string(js)}, nil
}

//deprecated
func (api *GtasAPI) MinerAbort(sign string, mtype int32) (*Result, error) {
	id := IdFromSign(sign)
	address := common.BytesToAddress(id)

	nonce := time.Now().UnixNano()
	tx := &types.Transaction{
		Nonce:    uint64(nonce),
		Data:     []byte{byte(mtype)},
		Source:   &address,
		Type:     types.TransactionTypeMinerAbort,
		GasPrice: common.MaxUint64,
	}
	tx.Hash = tx.GenHash()
	ok, err := core.BlockChainImpl.GetTransactionPool().AddTransaction(tx)
	if !ok {
		return failResult(err.Error())
	}
	return successResult(nil)
}

//deprecated
func (api *GtasAPI) MinerRefund(sign string, mtype int32) (*Result, error) {
	id := IdFromSign(sign)
	address := common.BytesToAddress(id)

	nonce := time.Now().UnixNano()
	tx := &types.Transaction{
		Nonce:    uint64(nonce),
		Data:     []byte{byte(mtype)},
		Source:   &address,
		Type:     types.TransactionTypeMinerRefund,
		GasPrice: common.MaxUint64,
	}
	tx.Hash = tx.GenHash()
	ok, err := core.BlockChainImpl.GetTransactionPool().AddTransaction(tx)
	if !ok {
		return &Result{Message: err.Error(), Data: nil}, nil
	}
	return &Result{Message: "success"}, nil
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
		b := chain.QueryBlockByHeight(h)
		if b == nil {
			continue
		}
		bh := b.Header
		p := string(bh.Castor)
		if v, ok := proposerStat[p]; ok {
			proposerStat[p] = v + 1
		} else {
			proposerStat[p] = 1
		}
		g := string(bh.GroupId)
		if v, ok := groupStat[g]; ok {
			groupStat[g] = v + 1
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

func (api *GtasAPI) MinerInfo(addr string) (*Result, error) {
	morts := make([]MortGage, 0)
	id := common.HexToAddress(addr).Bytes()
	heavyInfo := core.MinerManagerImpl.GetMinerById(id, types.MinerTypeHeavy, nil)
	if heavyInfo != nil {
		morts = append(morts, *NewMortGageFromMiner(heavyInfo))
	}
	lightInfo := core.MinerManagerImpl.GetMinerById(id, types.MinerTypeLight, nil)
	if lightInfo != nil {
		morts = append(morts, *NewMortGageFromMiner(lightInfo))
	}
	return successResult(morts)
}

func (api *GtasAPI) NodeInfo() (*Result, error) {
	ni := &NodeInfo{}
	p := mediator.Proc
	ni.ID = p.GetMinerID().GetHexString()
	balance, err := walletManager.getBalance(p.GetMinerID().GetHexString())
	if err != nil {
		return failResult(err.Error())
	}
	ni.Balance = balance
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

		ni.TxPoolNum = int(core.BlockChainImpl.GetTransactionPool().TxNum())

	}
	return successResult(ni)

}

func (api *GtasAPI) PageGetBlocks(page, limit int) (*Result, error) {
	chain := core.BlockChainImpl
	total := chain.Height() + 1
	pageObject := PageObjects{
		Total: total,
		Data:  make([]interface{}, 0),
	}
	if page < 1 {
		page = 1
	}
	i := 0
	num := uint64((page - 1) * limit)
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
	total := chain.Height()
	pageObject := PageObjects{
		Total: total,
		Data:  make([]interface{}, 0),
	}

	i := 0
	b := int64(0)
	if page < 1 {
		page = 1
	}
	num := uint64((page - 1) * limit)
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
			mems = append(mems, groupsig.DeserializeId(mem).ShortS())
		}

		group := &Group{
			Height:        uint64(b + 1),
			Id:            groupsig.DeserializeId(g.Id),
			PreId:         groupsig.DeserializeId(g.Header.PreGroup),
			ParentId:      groupsig.DeserializeId(g.Header.Parent),
			BeginHeight:   g.Header.WorkHeight,
			DismissHeight: g.Header.DismissHeight,
			Members:       mems,
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
	block := convertBlockHeader(b)

	preBH := chain.QueryBlockHeaderByHash(bh.PreHash)
	block.Qn = bh.TotalQN - preBH.TotalQN

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
				if st == types.ReceiptStatusSuccessful {
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
			increase += model.Param.ProposalBonus + uint64(mb.PackBonusTx)*model.Param.PackBonus
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

func (api *GtasAPI) BlockReceipts(h string) (*Result, error) {
	chain := core.BlockChainImpl
	b := chain.QueryBlockByHash(common.HexToHash(h))
	if b == nil {
		return failResult("block not found")
	}

	evictedReceipts := make([]*types.Receipt, 0)
	//for _, tx := range bh.EvictedTxs {
	//	wrapper := chain.GetTransactionPool().GetReceipt(tx)
	//	if wrapper != nil {
	//		evictedReceipts = append(evictedReceipts, wrapper)
	//	}
	//}
	receipts := make([]*types.Receipt, len(b.Transactions))
	for i, tx := range b.Transactions {
		wrapper := chain.GetTransactionPool().GetReceipt(tx.Hash)
		if wrapper != nil {
			receipts[i] = wrapper
		}
	}
	br := &BlockReceipt{EvictedReceipts: evictedReceipts, Receipts: receipts}
	return successResult(br)
}

func (api *GtasAPI) TransDetail(h string) (*Result, error) {
	tx := core.BlockChainImpl.GetTransactionByHash(false, true, common.HexToHash(h))

	if tx != nil {
		trans := convertTransaction(tx)
		return successResult(trans)
	}
	return successResult(nil)
}

func (api *GtasAPI) Dashboard() (*Result, error) {
	blockHeight := core.BlockChainImpl.Height()
	groupHeight := core.GroupChainImpl.Height()
	workNum := len(mediator.Proc.GetCastQualifiedGroups(blockHeight))
	nodeResult, _ := api.NodeInfo()
	consResult, _ := api.ConnectedNodes()
	dash := &Dashboard{
		BlockHeight: blockHeight,
		GroupHeight: groupHeight,
		WorkGNum:    workNum,
		NodeInfo:    nodeResult.Data.(*NodeInfo),
		Conns:       consResult.Data.([]ConnInfo),
	}
	return successResult(dash)
}

func (api *GtasAPI) Nonce(addr string) (*Result, error) {
	address := common.HexToAddress(addr)
	nonce := core.BlockChainImpl.GetNonce(address)
	return successResult(nonce)
}

func (api *GtasAPI) TxReceipt(h string) (*Result, error) {
	hash := common.HexToHash(h)
	rc := core.BlockChainImpl.GetTransactionPool().GetReceipt(hash)
	if rc != nil {
		tx := core.BlockChainImpl.GetTransactionByHash(false, true, hash)
		return successResult(&core.ExecutedTransaction{
			Receipt:     rc,
			Transaction: tx,
		})
	}
	return failResult("tx not exist")
}

func (api *GtasAPI) RpcSyncBlocks(height uint64, limit int, version int) (*Result, error) {
	chain := core.BlockChainImpl
	v := chain.Version()
	if version != v {
		return failResult(fmt.Sprintf("version not support, expect version %v", v))
	}
	blocks := chain.BatchGetBlocksAfterHeight(height, limit)
	return successResult(blocks)
}