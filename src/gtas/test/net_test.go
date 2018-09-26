package test

import (
	"testing"
	"network"
	"common"
	"fmt"
	"time"
	"net/http"
	_ "net/http/pprof"

	"log"
	"runtime"
)

func TestNet1(test *testing.T) {
	common.InitConf("tas1.ini")
	id, _ := network.Init(common.GlobalConf, true, nil, nil, true, "10.0.0.66")
	fmt.Printf("id:%s\n", id)
	pprof()
	go sendMsg()
	go sendMsg()
	go sendMsg()
	sendMsg()
}

func TestNet2(test *testing.T) {
	common.InitConf("tas2.ini")
	id, _ := network.Init(common.GlobalConf, false, nil, nil, true, "10.0.0.66")
	fmt.Printf("id:%s\n", id)
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