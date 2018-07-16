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
)

// GtasAPI is a single-method API handler to be returned by test services.
type GtasAPI struct {
}

// T 交易接口
func (api *GtasAPI) T(from string, to string, amount uint64, code string) (*Result, error) {
	err := walletManager.transaction(from, to, amount, code)
	if err != nil {
		return nil, err
	}
	return &Result{
		Message: fmt.Sprintf("Address: %s Send %d to Address:%s", from, amount, to),
		Data:    "",
	}, nil
}

// Balance 查询余额接口
func (api *GtasAPI) Balance(account string) (*Result, error) {
	// TODO 查询余额接口
	balance, err := walletManager.getBalance(account)
	if err != nil {
		return nil, err
	}
	return &Result{
		Message: fmt.Sprintf("The balance of account: %s is %d", account, balance),
		Data:    fmt.Sprintf("%d", balance),
	}, nil
}

// NewWallet 新建账户接口
func (api *GtasAPI) NewWallet() (*Result, error) {
	privKey, addr := walletManager.newWallet()
	data := make(map[string]string)
	data["private_key"] = privKey
	data["address"] = addr
	return &Result{fmt.Sprintf("Please Remember Your PrivateKey!\n "+
		"PrivateKey: %s\n WalletAddress: %s", privKey, addr), data}, nil
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

	nodes :=network.Network.ConnInfo()
	conns := make([]ConnInfo,0)
	for _,n := range nodes{
		conns = append(conns,ConnInfo{Id:n.Id,Ip:n.Ip,TcpPort:n.Port})
	}
	return &Result{"", nodes}, nil
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
	blockDetail["castor"] = castorId.GetString()
	//blockDetail["castor"] = hex.EncodeToString(bh.Castor)
	blockDetail["group_id"] = hex.EncodeToString(bh.GroupId)
	blockDetail["signature"] = hex.EncodeToString(bh.Signature)
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
