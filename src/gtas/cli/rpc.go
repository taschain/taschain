package cli

import (
	"net"
	"vm/rpc"

	"fmt"
	"log"
	"network/p2p"
	"strconv"
	"core"
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

// Vote
func (api *GtasAPI) Vote(from, modelNum string,v *VoteConfig) (*Result, error) {
	config := v.ToGlobal()
	walletManager.newVote(from, modelNum, config)
	return &Result{"success", ""}, nil
}

// ConnectedNodes 查询已链接的node的信息
func (api *GtasAPI) ConnectedNodes() (*Result, error) {
	//defer func() {
	//	if err := recover(); err != nil {
	//		fmt.Println(err)
	//	}
	//}()
	nodes := p2p.Server.GetConnInfo()
	return &Result{"", nodes}, nil
}

// TransPool 查询缓冲区的交易信息。
func (api *GtasAPI) TransPool() (*Result, error) {
	transactions := core.BlockChainImpl.GetTransactionPool().GetReceived()
	transList := make([]Transactions, 0, 5)
	for _, v := range transactions {
		transList = append(transList, Transactions{
			Source: v.Source.GetHexString(),
			Target: v.Target.GetHexString(),
			Value:  strconv.FormatInt(int64(v.Value), 10),
		})
	}
	return &Result{"success", transList}, nil
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
	apis := []rpc.API{
		{Namespace: "GTAS", Version: "1", Service: &GtasAPI{}, Public: true},
	}
	err := startHTTP(fmt.Sprintf("%s:%d", host, port), apis, []string{}, []string{}, []string{})
	if err != nil {
		return err
	}
	log.Printf("RPC serving on http://%s:%d\n", host, port)
	return nil
}
