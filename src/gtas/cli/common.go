package cli

import (
	"bytes"
	"common"
	"core"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

// 获取rpc接口的message,如果发生错误，error返回result中的错误提示
func getMessage(addr string, port uint, method string, params ...interface{}) (string, error) {
	res, err := rpcPost(addr, port, method, params...)
	if err != nil {
		return "", err
	}
	if res.Error != nil {
		return "", errors.New(res.Error.Message)
	}
	return res.Result.Message, nil
}

// 通用的rpc的请求方法。
func rpcPost(addr string, port uint, method string, params ...interface{}) (*RPCResObj, error) {
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

func genHash(hash string) []byte {
	bytes3 := []byte(hash)
	return core.Sha256(bytes3)
}

func genTx(hash string, price uint64, source string, target string, nonce uint64, value uint64, data []byte, extraData []byte,
	extraDataType int16) *core.Transaction {

	sourcebyte := common.HexToAddress(source)
	targetbyte := common.HexToAddress(target)

	return &core.Transaction{
		Data:          data,
		GasPrice:      price,
		Hash:          common.BytesToHash(genHash(hash)),
		Source:        &sourcebyte,
		Target:        &targetbyte,
		Nonce:         nonce,
		Value:         value,
		ExtraData:     extraData,
		ExtraDataType: extraDataType,
	}
}

func getRandomString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytess := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytess[r.Intn(len(bytess))])
	}
	return string(result)
}
