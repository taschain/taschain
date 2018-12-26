//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package cli

import (
	"common"
	"encoding/json"
	"errors"
	"golang.org/x/time/rate"
	"log"
	"core"
	"sync"
	"middleware/types"
)

// Wallets 钱包
type wallets []wallet

var mutex sync.Mutex

var limiter *rate.Limiter

func init()  {
	limiter = rate.NewLimiter(200, 200)
}

//
func (ws *wallets) transaction(source, target string, value uint64, code string, cmd int32, gas, gasprice uint64) (*common.Hash, *common.Address, error) {
	if !limiter.Allow(){
		return nil, nil, errors.New("rate limit")
	}
	if source == "" {
		source = (*ws)[0].Address
	}
	nonce := core.BlockChainImpl.GetNonce(common.HexToAddress(source))
	txpool := core.BlockChainImpl.GetTransactionPool()
	//if strings.HasPrefix(code, "0x") {
	//	code = code[2:]
	//}
	//codeBytes, err := hex.DecodeString(code)
	//if err != nil {
	//	return nil, nil, err
	//}
	var transaction *types.Transaction
	var contractAddr common.Address
	var i uint64 = 0
	for ; i < 100; i++ {
		transaction = genTx(i, source, target, nonce+i, value, []byte(code), nil, 0, cmd)
		transaction.Hash = transaction.GenHash()
		_, err := txpool.AddTransaction(transaction)
		if err != nil {
			return nil, nil, err
		}
		if code != "" {
			contractAddr = common.BytesToAddress(common.Sha256(common.BytesCombine(transaction.Source[:], common.Uint64ToByte(nonce))))
		}
	}


	return &transaction.Hash, &contractAddr, nil
}

//存储钱包账户
func (ws *wallets) store() {
	js, err := json.Marshal(ws)
	if err != nil {
		log.Println("store wallets error")
		// TODO 输出log
	}
	common.GlobalConf.SetString(Section, "wallets", string(js))
}

func (ws *wallets) deleteWallet(key string) {
	mutex.Lock()
	defer mutex.Unlock()
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
	mutex.Lock()
	defer mutex.Unlock()
	priv := common.GenerateKey("")
	pub := priv.GetPubKey()
	address := pub.GetAddress()
	privKeyStr, walletAddress = pub.GetHexString(), address.GetHexString()
	// 加入本地钱包
	//*ws = append(*ws, wallet{privKeyStr, walletAddress})
	//ws.store()
	return
}

func (ws *wallets) getBalance(account string) (int64, error) {
	if account == "" && len(walletManager) > 0 {
		account = walletManager[0].Address
	}
	balance := core.BlockChainImpl.GetBalance(common.HexToAddress(account))
	return balance.Int64(), nil
}

//func (ws *wallets) newVote(source string, config *global.VoteConfig) error {
//	if source == "" {
//		source = (*ws)[0].Address
//	}
//	abi, err := config.AbiEncode()
//	if err != nil {
//		return err
//	}
//	nonce := core.BlockChainImpl.GetNonce(common.HexToAddress(source))
//	txpool := core.BlockChainImpl.GetTransactionPool()
//	transaction := genTx(0, source, "", nonce+1, 0, abi, nil, 1)
//	transaction.Hash = transaction.GenHash()
//	_, err = txpool.Add(transaction)
//	if err != nil {
//		return err
//	}
//	return nil
//}

func newWallets() wallets {
	var ws wallets
	s := common.GlobalConf.GetString(Section, "wallets", "")
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