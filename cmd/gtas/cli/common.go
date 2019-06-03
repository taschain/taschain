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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
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