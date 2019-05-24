package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

//func TestBlock_Height(t *testing.T) {
//	res, err := rpcPost("47.99.193.22", 8101, "GTAS_blockHeight")
//	if nil == err {
//		fmt.Println(res.Result)
//	}
//}

func TestR(t *testing.T) {
	res, err := rpcPost("120.77.41.14", 8101, "GTAS_r")
	//res, err := rpcPost("112.74.61.71", 8106, "GTAS_r")
	if nil == err {
		fmt.Println(res.Result)
	} else {
		fmt.Println(err)
	}
}

//func TestL(t *testing.T) {
//	res, err := rpcPost("120.77.41.14", 8101, "GTAS_l","key3",[]byte("123"))
//	if nil == err {
//		fmt.Println(res.Result)
//	}else {
//		fmt.Println(err)
//	}
//}
//
//func TestS(t *testing.T) {
//	res, err := rpcPost("120.77.41.14", 8101, "GTAS_s","key3")
//	if nil == err {
//		result := res.Result
//		fmt.Println(result.Message)
//		fmt.Println(result.Success)
//		if result.Success{
//			data:=result.Data.(map[string]interface {})
//			str := data["ExtraData"].(string)
//			bytess:=[]byte(str)
//			fmt.Println(bytess)
//		}
//
//
//		fmt.Println(result.Data)
//	}else {
//		fmt.Println(err)
//	}
//}
//
//
// 通用的rpc的请求方法。
func rpcPost(addr string, port int, method string, params ...interface{}) (*RPCResObj, error) {
	obj := RPCReqObj{
		Method: method,
		Params: params,
		//Jsonrpc: "2.0",
		//ID:      1,
	}
	objBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(objBytes))
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

// RPCResObj 完整的rpc返回体
type RPCResObj struct {
	Jsonrpc string       `json:"jsonrpc"`
	ID      uint         `json:"id"`
	Result  *Result      `json:"result,omitempty"`
	Error   *ErrorResult `json:"error,omitempty"`
}

// Result rpc请求成功返回的可变参数部分
type Result struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Success bool        `json:"succ"`
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
