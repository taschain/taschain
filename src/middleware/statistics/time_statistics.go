package statistics

import (
	"sync"
	"time"

	"bytes"
	"encoding/json"
	"fmt"
	"common"
)

const (
	KingCasting = 1
	MessageCast = 2
	MessageVerify = 3
	NewBlock = 4
)

var BatchSize = 1000
var Duration time.Duration = 5
var Lock sync.Mutex
var LogChannel = make(chan *LogObj)
var TimeChannel = make(chan int)
var IsInit = false
var WriteData = make([]*LogObj,0)
var batch int
type LogObj struct {
	Hash string
	Status int
	Time int64
	Batch  int
	Castor   string
	Node   string
}

func NewLogObj(id string)*LogObj{
	lg := new(LogObj)
	lg.Node = id
	lg.Hash = "cf545b9496a1665285aa385d9ee5542154f2fb4dcefc820b4ccb00741b88c9ed"
	lg.Castor = "cf545b9496a1665285aa385d"
	lg.Status = 2
	lg.Time = time.Now().Unix()
	lg.Batch = 1
	return lg
}
func AddLog(Hash string, Status int, Time int64, Castor string, Node string,){
	log := &LogObj{Hash:Hash,Status:Status,Time:Time,Batch:batch,Castor:Castor,Node:Node}
	PutLog(log)
}

func PutLog(data *LogObj){
	//if !HasInit(){
	//	Init()
	//}
	go func(){
		LogChannel <- data
		}()
}

func InitStatistics(config common.ConfManager){
	url = config.GetString("statistics","url","http://118.31.60.210:9090/send")
	timeout = time.Duration(config.GetInt("statistics","timeout",1)) * time.Second
	batch =  config.GetInt("statistics","batch",0)
	go ProcessLog()
	go func() {
		t := time.Tick(Duration * time.Second)

		for {
			select {
			case <-t:
				TimeChannel<-1
			}
		}
	}()
	initCount(config)
}

func HasInit()bool{
	if IsInit{
		return true
	}
	Lock.Lock()
	if IsInit{
		return true
	}else{
		IsInit = true
	}
	defer Lock.Unlock()
	return false
}

func ProcessLog(){
	for{
		select {
			case log := <-LogChannel:
				WriteData = append(WriteData,log)
				SendLog()
			case  <- TimeChannel:
				SendLogByTime()
		}
	}
}
func SendLogByTime(){
	if len(WriteData)== 0{
		return
	}
	Send()
	Clear()
}
func SendLog(){
	if len(WriteData)== 0{
		return
	}
	if len(WriteData) >= BatchSize{
		Send()
		Clear()
	}
}

func Send(){
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(WriteData)
	fmt.Printf("send batch len:%d\n",len(WriteData))
	SendPost(b)
}

func Clear(){
	WriteData = WriteData[0:0]
}



