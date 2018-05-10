package cli

import (
	"common"
	"encoding/hex"
	"encoding/json"
	"governance/global"
	"log"
	"strings"
	"core"
)

// Wallets 钱包
type wallets []wallet

//
func (ws *wallets) transaction(source, target string, value uint64, code string) error {
	if source == "" {
		source = (*ws)[0].Address
	}
	nonce := core.BlockChainImpl.GetNonce(common.HexToAddress(source))
	txpool := core.BlockChainImpl.GetTransactionPool()
	if strings.HasPrefix(code, "0x") {
		code = code[2:]
	}
	codeBytes, err := hex.DecodeString(code)
	if err != nil {
		return err
	}
	transaction := genTx(0, source, target, nonce, value, codeBytes, nil, 0)
	transaction.Hash = transaction.GenHash()
	_, err = txpool.Add(transaction)
	if err != nil {
		return err
	}
	return nil
}

//存储钱包账户
func (ws *wallets) store() {
	js, err := json.Marshal(ws)
	if err != nil {
		log.Println("store wallets error")
		// TODO 输出log
	}
	(*configManager).SetString(Section, "wallets", string(js))
}

func (ws *wallets) deleteWallet(key string) {
	for i, v := range *ws {
		if v.Address == key || v.PrivateKey == key {
			*ws = append((*ws)[:i], (*ws)[i+1:]...)
			break
		}
	}
	ws.store()
}

// newWallet 新建钱包并存储到config文件中
func (ws *wallets) newWallet() (privKeyStr, walletAddress string) {
	priv := common.GenerateKey("")
	pub := priv.GetPubKey()
	address := pub.GetAddress()
	privKeyStr, walletAddress = pub.GetHexString(), address.GetHexString()
	// 加入本地钱包
	*ws = append(*ws, wallet{privKeyStr, walletAddress})
	ws.store()
	return
}

func (ws *wallets) getBalance(account string) (int64, error) {
	if account == "" && len(walletManager) > 0 {
		account = walletManager[0].Address
	}
	balance := core.BlockChainImpl.GetBalance(common.HexToAddress(account))
	return balance.Int64(), nil
}

func (ws *wallets) newVote(source string, config *global.VoteConfig) error {
	if source == "" {
		source = (*ws)[0].Address
	}
	abi, err := config.AbiEncode()
	if err != nil {
		return err
	}
	nonce := core.BlockChainImpl.GetNonce(common.HexToAddress(source))
	txpool := core.BlockChainImpl.GetTransactionPool()
	transaction := genTx(0, source, "", nonce+1, 0, abi, nil, 1)
	transaction.Hash = transaction.GenHash()
	_, err = txpool.Add(transaction)
	if err != nil {
		return err
	}
	return nil
}

func newWallets() wallets {
	var ws wallets
	s := (*configManager).GetString(Section, "wallets", "")
	if s == "" {
		return ws
	}
	err := json.Unmarshal([]byte(s), &ws)
	if err != nil {
		// TODO 输出log
		log.Println(err)
	}
	return ws
}