package cli

import (
	"common"
	"encoding/hex"
	"encoding/json"
	"governance/global"
	"log"
	"strings"
)

// Wallets 钱包
type wallets []wallet

//
func (ws *wallets) transaction(source, target string, value uint64, code string) error {
	nonce := blockChain.GetNonce(common.HexToAddress(source))
	txpool := blockChain.GetTransactionPool()
	if strings.HasPrefix(code, "0x") {
		code = code[2:]
	}
	codeBytes, err := hex.DecodeString(code)
	if err != nil {
		return err
	}
	txpool.Add(genTx(getRandomString(8), 1, source, target, nonce+1, value, codeBytes))
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
	balance := blockChain.GetBalance(common.HexToAddress(account))
	return balance.Int64(), nil
}

func newVote(config *global.VoteConfig) error {
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
