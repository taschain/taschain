package test

import (
	"testing"
	"network"
	"common"
	"time"
	"net/http"
	_ "net/http/pprof"

	"log"
	"runtime"
)

func TestNet1(test *testing.T) {
	common.InitConf("tas1.ini")
	netCfg :=  network.NetworkConfig{IsSuper:true,TestMode:true,SeedIp: "10.0.0.66"}
	network.Init(common.GlobalConf, nil, netCfg)
	pprof()
	go sendMsg()
	go sendMsg()
	go sendMsg()
	sendMsg()
}

func TestNet2(test *testing.T) {
	common.InitConf("tas2.ini")
	netCfg :=  network.NetworkConfig{IsSuper:false,TestMode:true,SeedIp: "10.0.0.66"}
	network.Init(common.GlobalConf, nil, netCfg)
	go sendMsg()
	go sendMsg()
	go sendMsg()
	sendMsg()
}

func sendMsg(){
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

func pprof(){
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
