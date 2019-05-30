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

package test

import (
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/network"
	"net/http"
	_ "net/http/pprof"
	"testing"
	"time"

	"log"
	"runtime"
)

func TestNet1(test *testing.T) {
	common.InitConf("tas1.ini")
	netCfg := network.NetworkConfig{IsSuper: true, TestMode: true, SeedIp: "10.0.0.66"}
	network.Init(common.GlobalConf, nil, netCfg)
	pprof()
	go sendMsg()
	go sendMsg()
	go sendMsg()
	sendMsg()
}

func TestNet2(test *testing.T) {
	common.InitConf("tas2.ini")
	netCfg := network.NetworkConfig{IsSuper: false, TestMode: true, SeedIp: "10.0.0.66"}
	network.Init(common.GlobalConf, nil, netCfg)
	go sendMsg()
	go sendMsg()
	go sendMsg()
	sendMsg()
}

func sendMsg() {
	for {
		m := mockMsg()
		network.GetNetInstance().Broadcast(m)
		time.Sleep(time.Millisecond * 10)
	}
}

func mockMsg() network.Message {

	body := make([]byte, 250000)
	msg := network.Message{Code: 1, Body: body}
	return msg
}

func pprof() {
	go func() {
		http.ListenAndServe("localhost:1111", nil)
	}()
}

func gc() {

	gcTick := time.NewTicker(time.Second * 5)
	for {
		<-gcTick.C
		log.Println("Force GC...")
		runtime.GC()
		//debug.FreeOSMemory()
	}
}
