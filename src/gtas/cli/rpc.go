package cli

import (
	"net"
	"vm/rpc"

	"fmt"
	"network"
	"strconv"
	"core"
	"encoding/hex"
	"strings"
	"log"
	"math"
	"consensus/groupsig"
	"common"
	"consensus/mediator"
	"consensus/logical"
	"middleware/types"
)

// GtasAPI is a single-method API handler to be returned by test services.
type GtasAPI struct {
}

var nicks = make(map[string]string)
var alias = make(map[string]string)

type txs []*common.Hash

var relatedTxs = make(map[string]txs)

type txdetail struct {
	Hash      string `json:"hash"`
	Height    uint64 `json:"height"`
	From      string `json:"from"`
	To        string `json:"to"`
	Value     uint64 `json:"value"`
	BlockHash string `json:"block_hash"`
}

func (api *GtasAPI) formatAccount(account string) string {
	if len(account) == 0 {
		return account
	}

	addr := nicks[account]
	if len(addr) != 0 {
		return addr
	}

	if strings.HasPrefix(account, "TAS:") {
		return account[4:]
	}

	return account
}

// T 交易接口
func (api *GtasAPI) T(from string, to string, amount uint64, code string) (*Result, error) {
	if len(from) == 0 {
		from = "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
	}
	from = api.formatAccount(from)
	to = api.formatAccount(to)
	hash, err := walletManager.transaction(from, to, amount, code)
	if err != nil {
		return nil, err
	}

	tx := relatedTxs[from]
	if nil == tx {
		tx = make([]*common.Hash, 10)
		relatedTxs[from] = tx
	}
	relatedTxs[from] = append(relatedTxs[from], hash)

	tx = relatedTxs[to]
	if nil == tx {
		tx = make([]*common.Hash, 10)
		relatedTxs[to] = tx
	}
	relatedTxs[to] = append(relatedTxs[to], hash)

	return &Result{
		Message: fmt.Sprintf("Address: %s Send %d to Address:%s", from, amount, to),
		Data:    hash.String(),
	}, nil
}

func (api *GtasAPI) Transactions(account string) (*Result, error) {
	txs := relatedTxs[api.formatAccount(account)]
	if nil == txs || 0 == len(txs) {
		return &Result{
			Message: "nil txs",
		}, nil
	}

	details := make([]*txdetail, 0)
	for _, tx := range txs {
		if nil == tx {
			continue
		}
		wrapper := core.BlockChainImpl.GetExecutedTransactionByHash(*tx)
		if nil == wrapper {
			continue
		}

		detail := &txdetail{
			Hash:      tx.Hex(),
			From:      wrapper.Transaction.Source.GetHexString(),
			To:        wrapper.Transaction.Target.GetHexString(),
			Value:     wrapper.Transaction.Value,
			Height:    wrapper.Height,
			BlockHash: wrapper.BlockHash.Hex(),
		}

		alia,ok := alias[detail.From]
		if ok {
			detail.From = alia
		}
		alia,ok = alias[detail.To]
		if ok {
			detail.To = alia
		}

		details = append(details, detail)

	}

	return &Result{
		Message: "success",
		Data:    details,
	}, nil

}

// Balance 查询余额接口
func (api *GtasAPI) Balance(account string) (*Result, error) {
	balance, err := walletManager.getBalance(api.formatAccount(account))
	if err != nil {
		return nil, err
	}
	return &Result{
		Message: fmt.Sprintf("The balance of account: %s is %d", account, balance),
		Data:    fmt.Sprintf("%d", balance),
	}, nil
}

// NewWallet 新建账户接口
func (api *GtasAPI) NewWallet(nick string) (*Result, error) {

	privKey, addr := walletManager.newWallet()
	data := make(map[string]string)
	data["private_key"] = privKey
	if len(nick) != 0 {
		if strings.HasPrefix(nick, "TAS:") {
			nick = ""
		} else {
			nicks[nick] = addr

		}

		_, ok := alias[addr]
		if !ok && len(nick) != 0 {
			alias[addr] = nick
		}

	}

	data["nick"] = nick
	addr = "TAS:" + addr[2:]
	data["address"] = addr
	return &Result{fmt.Sprintf("Please Remember Your PrivateKey!\n "+
		"PrivateKey: %s\n WalletAddress: %s\n nick: %s", privKey, addr, nick), data}, nil
}

// GetWallets 获取当前节点的wallets
func (api *GtasAPI) GetWallets() (*Result, error) {
	return &Result{"", walletManager}, nil
}

// DeleteWallet 删除本地节点指定序号的地址
func (api *GtasAPI) DeleteWallet(key string) (*Result, error) {
	walletManager.deleteWallet(key)
	return &Result{"", walletManager}, nil
}

// ClearBlock 删除本地链
func (api *GtasAPI) ClearBlock() (*Result, error) {
	err := ClearBlock()
	if err != nil {
		return nil, err
	}
	return &Result{fmt.Sprint("remove wallet file"), ""}, nil
}

// BlockHeight 块高查询
func (api *GtasAPI) BlockHeight() (*Result, error) {
	height := core.BlockChainImpl.QueryTopBlock().Height
	return &Result{fmt.Sprintf("The height of top block is %d", height), height}, nil
}

// GroupHeight 组块高查询
func (api *GtasAPI) GroupHeight() (*Result, error) {
	height := core.GroupChainImpl.Count()
	return &Result{fmt.Sprintf("The height of group is %d", height), height}, nil
}

// Vote
func (api *GtasAPI) Vote(from string, v *VoteConfig) (*Result, error) {
	config := v.ToGlobal()
	walletManager.newVote(from, config)
	return &Result{"success", ""}, nil
}

// ConnectedNodes 查询已链接的node的信息
func (api *GtasAPI) ConnectedNodes() (*Result, error) {

	nodes := network.GetNetInstance().ConnInfo()
	conns := make([]ConnInfo, 0)
	for _, n := range nodes {
		conns = append(conns, ConnInfo{Id: n.Id, Ip: n.Ip, TcpPort: n.Port})
	}
	return &Result{"", conns}, nil
}

// TransPool 查询缓冲区的交易信息。
func (api *GtasAPI) TransPool() (*Result, error) {
	transactions := core.BlockChainImpl.GetTransactionPool().GetReceived()
	transList := make([]Transactions, 0, len(transactions))
	for _, v := range transactions {
		transList = append(transList, Transactions{
			Source: v.Source.GetHexString(),
			Target: v.Target.GetHexString(),
			Value:  strconv.FormatInt(int64(v.Value), 10),
		})
	}

	return &Result{"success", transList}, nil
}

func (api *GtasAPI) GetTransaction(hash string) (*Result, error) {
	transaction, err := core.BlockChainImpl.GetTransactionByHash(common.HexToHash(hash))
	if err != nil {
		return nil, err
	}
	detail := make(map[string]interface{})
	detail["hash"] = hash
	detail["source"] = transaction.Source.Hash().Hex()
	detail["target"] = transaction.Target.Hash().Hex()
	detail["value"] = transaction.Value
	return &Result{"success", detail}, nil
}

func (api *GtasAPI) GetBlock(height uint64) (*Result, error) {
	bh := core.BlockChainImpl.QueryBlockByHeight(height)
	blockDetail := make(map[string]interface{})
	blockDetail["hash"] = bh.Hash.Hex()
	blockDetail["height"] = bh.Height
	blockDetail["pre_hash"] = bh.PreHash.Hex()
	blockDetail["pre_time"] = bh.PreTime.Format("2006-01-02 15:04:05")
	blockDetail["queue_number"] = bh.QueueNumber
	blockDetail["cur_time"] = bh.CurTime.Format("2006-01-02 15:04:05")
	var castorId groupsig.ID
	castorId.Deserialize(bh.Castor)
	blockDetail["castor"] = castorId.String()
	//blockDetail["castor"] = hex.EncodeToString(bh.Castor)
	blockDetail["group_id"] = hex.EncodeToString(bh.GroupId)
	blockDetail["signature"] = hex.EncodeToString(bh.Signature)
	trans := make([]string, len(bh.Transactions))
	for i := range bh.Transactions {
		trans[i] = bh.Transactions[i].String()
	}
	blockDetail["transactions"] = trans
	blockDetail["txs"] = len(bh.Transactions)
	blockDetail["tps"] = math.Round(float64(len(bh.Transactions)) / bh.CurTime.Sub(bh.PreTime).Seconds())
	return &Result{"success", blockDetail}, nil
}

func (api *GtasAPI) GetTopBlock() (*Result, error) {
	bh := core.BlockChainImpl.QueryTopBlock()
	blockDetail := make(map[string]interface{})
	blockDetail["hash"] = bh.Hash.Hex()
	blockDetail["height"] = bh.Height
	blockDetail["pre_hash"] = bh.PreHash.Hex()
	blockDetail["pre_time"] = bh.PreTime.Format("2006-01-02 15:04:05")
	blockDetail["queue_number"] = bh.QueueNumber
	blockDetail["cur_time"] = bh.CurTime.Format("2006-01-02 15:04:05")
	blockDetail["castor"] = hex.EncodeToString(bh.Castor)
	blockDetail["group_id"] = hex.EncodeToString(bh.GroupId)
	blockDetail["signature"] = hex.EncodeToString(bh.Signature)
	blockDetail["txs"] = len(bh.Transactions)
	blockDetail["tps"] = math.Round(float64(len(bh.Transactions)) / bh.CurTime.Sub(bh.PreTime).Seconds())

	blockDetail["tx_pool_count"] = len(core.BlockChainImpl.GetTransactionPool().GetReceived())
	blockDetail["tx_pool_total"] = core.BlockChainImpl.GetTransactionPool().GetTotalReceivedTxCount()
	return &Result{"success", blockDetail}, nil
}

func (api *GtasAPI) WorkGroupNum(height uint64) (*Result, error) {
	groups := mediator.Proc.GetCastQualifiedGroups(height)
	return &Result{"success", len(groups)}, nil
}

func convertGroup(g *types.Group) map[string]interface{} {
	gmap := make(map[string]interface{})
	if g.Id != nil && len(g.Id) != 0 {
		gmap["group_id"] = logical.GetIDPrefix(*groupsig.DeserializeId(g.Id))
		gmap["dummy"] = false
	} else {
		gmap["group_id"] = logical.GetIDPrefix(*groupsig.DeserializeId(g.Dummy))
		gmap["dummy"] = true
	}
	gmap["parent"] = logical.GetIDPrefix(*groupsig.DeserializeId(g.Parent))
	gmap["begin_height"] = g.BeginHeight
	gmap["dismiss_height"] = g.DismissHeight
	mems := make([]string, 0)
	for _, mem := range g.Members {
		memberStr := groupsig.DeserializeId(mem.Id).GetHexString()
		mems = append(mems, memberStr[0:6]+"-"+memberStr[len(memberStr)-6:])
	}
	gmap["members"] = mems
	return gmap
}

func (api *GtasAPI) GetGroupsAfter(height uint64) (*Result, error) {
	groups, err := core.GroupChainImpl.GetGroupsByHeight(height)
	if err != nil {
		return &Result{"fail", err.Error()}, nil
	}
	ret := make([]map[string]interface{}, 0)
	h := height
	for _, g := range groups {
		gmap := convertGroup(g)
		gmap["height"] = h
		h++
		ret = append(ret, gmap)
	}
	return &Result{"success", ret}, nil
}

func (api *GtasAPI) GetCurrentWorkGroup() (*Result, error) {
	height := core.BlockChainImpl.Height()
	return api.GetWorkGroup(height)
}

func (api *GtasAPI) GetWorkGroup(height uint64) (*Result, error) {
	groups := mediator.Proc.GetCastQualifiedGroups(height)
	ret := make([]map[string]interface{}, 0)

	for _, g := range groups {
		gmap := make(map[string]interface{})
		gmap["id"] = logical.GetIDPrefix(g.GroupID)
		gmap["parent"] = logical.GetIDPrefix(g.GroupID)
		mems := make([]string, 0)
		for _, mem := range g.Members {
			mems = append(mems, logical.GetIDPrefix(mem.ID))
		}
		gmap["group_members"] = mems
		gmap["begin_height"] = g.BeginHeight
		gmap["dismiss_height"] = g.DismissHeight
		ret = append(ret, gmap)
	}
	return &Result{"success", ret}, nil
}

// startHTTP initializes and starts the HTTP RPC endpoint.
func startHTTP(endpoint string, apis []rpc.API, modules []string, cors []string, vhosts []string) error {
	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
			if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
				return err
			}
		}
	}
	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return err
	}
	go rpc.NewHTTPServer(cors, vhosts, handler).Serve(listener)

	return nil
}

// StartRPC RPC 功能
func StartRPC(host string, port uint) error {
	var err error
	apis := []rpc.API{
		{Namespace: "GTAS", Version: "1", Service: &GtasAPI{}, Public: true},
	}
	for plus := 0; plus < 40; plus ++ {
		err = startHTTP(fmt.Sprintf("%s:%d", host, port+uint(plus)), apis, []string{}, []string{}, []string{})
		if err == nil {
			log.Printf("RPC serving on http://%s:%d\n", host, port+uint(plus))
			return nil
		}
		if strings.Contains(err.Error(), "address already in use") {
			log.Printf("address: %s:%d already in use\n", host, port+uint(plus))
			continue
		}
		return err
	}
	return err
}
