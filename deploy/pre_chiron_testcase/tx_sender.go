package main

import (
	"net/http"
	"fmt"
	"bytes"
	"io/ioutil"
	"encoding/json"
	"math/rand"
	"flag"
	"time"
	"strings"
	"errors"
	"strconv"
)

// 通用的rpc的请求方法。
func rpcPost(addr string, port int, method string, params ...interface{}) (*RPCResObj, error) {
	obj := RPCReqObj{
		Method:  method,
		Params:  params,
		Jsonrpc: "2.0",
		ID:      1,
	}
	objBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(
		fmt.Sprintf("http://%s:%d", addr, port),
		"application/json",
		bytes.NewReader(objBytes),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var resJSON RPCResObj
	if err := json.Unmarshal(responseBytes, &resJSON); err != nil {
		return nil, err
	}
	return &resJSON, nil
}

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

func getRichAccount(addr string, port int, accounts []string) (string, error) {
	accountsCopy := accounts[:]
	for {
		if len(accountsCopy) == 0 {
			return "", errors.New("accounts error")
		}
		num := rand.Intn(len(accountsCopy))
		return accounts[num],nil
		//res, err := rpcPost(addr, port, "GTAS_balance", accounts[num])
		//if err != nil {
		//  return "", err
		//}
		//if res.Error != nil {
		//  return "", errors.New(res.Error.Message)
		//}
		//fmt.Println(res.Result.Data)
		//value, err := strconv.Atoi(res.Result.Data.(string))
		//if err != nil {
		//  return "", err
		//}
		//if value > 0 {
		//  return accounts[num], nil
		//}
		//accountsCopy = append(accountsCopy[:num], accountsCopy[num+1:]...)
	}
}

func transaction(addr string, port int, from, to string,nounce uint64) {
	res, err := rpcPost(addr, port, "GTAS_tx", from, to, 1, "",nounce,0)
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	if res.Error != nil {
		fmt.Println("err:", res.Error.Message)
		return
	}
	fmt.Println("suc:", res.Result.Message)
	//fmt.Printf("addr:%s,port:%d,from:%s,to:%s,nounce:%d\n",addr,port,from,to,nounce)
}

type url struct {
	host string
	port int
}

func main() {
	interval := flag.Duration("i", time.Second*3, "转账时间间隔")
	total := flag.Int("t", 100, "转账总笔数")
	accountsString := flag.String("a", "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b,d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35,4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce", "互转账号，用逗号隔开")
	list := flag.String("l", "127.0.0.1:8101,127.0.0.1:8102", "随机发送地址列表")
	flag.Parse()

	accounts := strings.Split(*accountsString, ",")
	urls := strings.Split(*list, ",")

	urlList := parse(urls)
	length := len(urlList)

	for i := 0; i < *total; i++ {
		url := urlList[i%length]
		account, err := getRichAccount(url.host, url.port, accounts)
		if err != nil {
			fmt.Println("err: ", err)
			continue
		}
		// 交易发给所有节点
		rand.Seed(time.Now().UnixNano())
		nounce :=rand.Uint64()
		toAccount := getRandomToAccount()
		for j := 0; j < len(urlList); j++ {
			urlInner := urlList[j]
			go transaction(urlInner.host, urlInner.port, account, toAccount,nounce)

		}
		fmt.Printf("Tx from %s to %s\n", account,toAccount)
		time.Sleep(*interval)
	}
}
func parse(urls []string) []url {
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

func getRandomToAccount()string{
	slist := []string{"0","1","2","3","4","5","6","7","8","9","a","b","c","d","e","f"}
	var result string
	for i:=0;i<64;i++{
		result += slist[rand.Intn(len(slist))]
	}
	return result
}