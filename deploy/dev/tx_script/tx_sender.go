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

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Result rpc请求成功返回的可变参数部分
type Result struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// ErrorResult rpc请求错误返回的可变参数部分
type ErrorResult struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// RPCReqObj 完整的rpc请求体
type RPCReqObj struct {
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Jsonrpc string        `json:"jsonrpc"`
	ID      uint          `json:"id"`
}

// RPCResObj 完整的rpc返回体
type RPCResObj struct {
	Jsonrpc string       `json:"jsonrpc"`
	ID      uint         `json:"id"`
	Result  *Result      `json:"result,omitempty"`
	Error   *ErrorResult `json:"error,omitempty"`
}

type url struct {
	host string
	port int
}

type account struct {
	privateKey string
	address    string
}

var richAccounts []*account
var hosts []url

func main() {
	interval := flag.Duration("i", time.Second*1, "转账时间间隔")
	total := flag.Int("t", 100000000, "转账总笔数")
	urlInput := flag.String("l", "127.0.0.1:8101,127.0.0.1:8102,127.0.0.1:8103,127.0.0.1:8104,127.0.0.1:8105,127.0.0.1:8106", "随机发送地址列表")
	flag.Parse()

	loadRichAccounts()
	hosts = extractURL(urlInput)

	rand.Seed(time.Now().UnixNano())
	nonce := rand.Uint64()
	for i := 0; i < *total; i++ {
		account := getRandomSourceAccount()
		toAccount := getRandomToAccount()
		url := getRandomHost()

		nonce++
		var gasPrice uint64 = 1
		var txType = 0
		go mockSendTransaction(url.host, url.port, account.privateKey, account.address, toAccount, nonce, txType, gasPrice)
		fmt.Printf("Tx from %s to %s\n", account.address, toAccount)
		time.Sleep(*interval)
	}
}

func extractURL(urlInput *string) []url {
	urls := strings.Split(*urlInput, ",")
	result := make([]url, len(urls))

	for i, s := range urls {
		splited := strings.Split(s, ":")
		if 2 != len(splited) {
			continue
		}

		p, _ := strconv.Atoi(splited[1])
		result[i] = url{
			host: splited[0],
			port: p,
		}
	}
	return result
}

func loadRichAccounts() {
	account1 := account{"0x045c8153e5a849eef465244c0f6f40a43feaaa6855495b62a400cc78f9a6d61c76c09c3aaef393aa54bd2adc5633426e9645dfc36723a75af485c5f5c9f2c94658562fcdfb24e943cf257e25b9575216c6647c4e75e264507d2d57b3c8bc00b361", "0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103"}
	account2 := account{"0x04dce627a77ba3677ee9ba2eb6ef701957f6355ed2d57f1b9c1c73445090f652c2e13d3db50c91893f31b1e813d2786ad0a8184d701ad10a16cec77d7cbedccb085efd499bcf9b29a16c9823a0551f77a143563dc5d3465105beed48c33a72e552", "0xcad6d60fa8f6330f293f4f57893db78cf660e80d6a41718c7ad75e76795000d4"}
	account3 := account{"0x042eddccfce3ec9057df4d4b7da92aaa05a7c570937dbfed800f2f50fe36757c26755094aa4094f407efee34c5064d2dad5ff89c1573dc6cdc56c7fb08fde8b0384f7e08ad4112289d1c4486e8d7245e7f09df47bcae6172d4670a84df0984fdab", "0xca789a28069db6f1639b60a8bf1084333358672f65c6d6c2e6d58b69187fe402"}
	account4 := account{"0x04c4ba6f0655e32a04ab8fcaff3cb4d018773810c7eccd6d593ef2a904b91e2dbf940dc4b0c76d9beb1d70945c1a91ccdd5678852a1599033dd2a4865654d4aeda4c14b2d012c2d5d3bec16fa27ae9a84bcad73f159d9cc22fb421742f8008ebe9", "0x94bdb92d329dac69d7f107995a7b666d1092c63eadeae2dd495ab2e554bb155d"}
	account5 := account{"0x04a7c38de85e7860d83dbc6d8e15fd675ef8d679d9c6aced385d6301e4b3ca9b083ecc316f3a1a15ff5dcda367431be0607b5d7b8e6f22d2c950dfcc1dbaf4ac58b813319621267ff6ce0628cd709fe552754e749dd3022232370a2a9b80887218", "0xb50eea221a1eb061dea7ca20f7b7508c2d9639e3558e69f758380e32624337b5"}
	account6 := account{"0x0455c636f1383c42440054c236c3738569e61f5c6f520b3e55bed27de18e0d22ad93769c15e8344abef9583094570106d22bfcf21a684780127407c9606ee069697532d22d2932b79c156728aa032a655aefd6cb0b774a481d632808cbe0ca8046", "0xce59fd5e1c6c99d9990b08ccf685260a2b3a03889de56e91b25878a4bf2f89e9"}
	account7 := account{"0x0485e7f35c89d36e0714e6171a32dedf3ff78ee2c24be0c49ec670fe6289490ab41ee52fa4e029f362a3c407118dbebdedce8c2ab72c45b1eca0a1db2fcab1d678a5e17f56e02decffab19cb34732f1b04e5d00bace6238a247422f974ac7d6bb9", "0x5d9b2132ec1d2011f488648a8dc24f9b29ca40933ca89d8d19367280dff59a03"}
	account8 := account{"0x040a5cd05e7d3bb6fc02f97dc98a6efaefbe4b201c9b6311a796933570158ece2db9aea7061f56f76958c9a722c948156b3724fd825cc030e98f23a91a674e02a0392bf3e1b1f629894bc11cb5858d41d3553874cb8bc95f6a6038a78786e3ee5d", "0x5afb7e2617f1dd729ea3557096021e2f4eaa1a9c8fe48d8132b1f6cf13338a8f"}
	account9 := account{"0x04ef0e7fda78e0d9cb6e099455a6c5575f748f095bac918a5e6f78f558fac232aa2648395fca3b45987f4f81c3accd72ffbb713642fe3ac1bfddf5f23dbd25d60db3870a9a28b38288706061a2c23866be13ca15d8229cffd404e506e925ca8853", "0x30c049d276610da3355f6c11de8623ec6b40fd2a73bb5d647df2ae83c30244bc"}
	account10 := account{"0x04c767316638cedeff24823de652e989772205177ae6f6970d86c61057d3b3c3fc488bb0f2156135909fbbe483b6d5f3469e1f8e99e55146813a63805f8fe21e51353107d7ad24c155b34deb2c4fb8154b9f19dc3cc48a94bf10bffcb97f9903c7", "0xa2b7bc555ca535745a7a9c55f9face88fc286a8b316352afc457ffafb40a7478"}
	richAccounts = []*account{&account1, &account2, &account3, &account4, &account5, &account6, &account7, &account8, &account9, &account10}
}

func getRandomSourceAccount() *account {
	num := rand.Intn(len(richAccounts))
	return richAccounts[num]
}

func getRandomToAccount() string {
	slist := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f"}
	var result = "0x"
	for i := 0; i < 64; i++ {
		result += slist[rand.Intn(len(slist))]
	}
	return result
}

func getRandomHost() url {
	num := rand.Intn(len(hosts))
	return hosts[num]
}
func mockSendTransaction(host string, port int, privateKey string, from, to string, nounce uint64, txType int, gasPrice uint64) {
	res, err := rpcPost(host, port, "GTAS_scriptTransferTx", privateKey, from, to, 1, nounce, txType, gasPrice)
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	if res.Error != nil {
		fmt.Println("err:", res.Error.Message)
		return
	}

	if res.Result == nil {
		fmt.Println(host, ":", port, "result:", res)
	} else {
		fmt.Println(host, ":", port, "result:", res.Result.Message, " hash:", res.Result.Data)
	}
}

// 通用的rpc的请求方法。
func rpcPost(addr string, port int, method string, params ...interface{}) (*RPCResObj, error) {
	obj := RPCReqObj{Method: method, Params: params, Jsonrpc: "2.0", ID: 1}
	objBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(fmt.Sprintf("http://%s:%d", addr, port), "application/json", bytes.NewReader(objBytes))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	responseBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ioutil.ReadAll err:", err.Error())
		return nil, err
	}
	var resJSON RPCResObj
	if responseBytes != nil && len(responseBytes) != 0 {
		if err := json.Unmarshal(responseBytes, &resJSON); err != nil {
			fmt.Println("responseBytes:", responseBytes)
			fmt.Println("json.Unmarshal err:", err.Error())
			return nil, err
		}
	}
	return &resJSON, nil
}
