package monitor

import (
	"time"
	"common"
	"fmt"
	"taslog"
	"middleware/notify"
	"middleware/types"
	"encoding/json"
	"github.com/hashicorp/golang-lru"
)

/*
**  Creator: pxf
**  Date: 2019/5/23 下午2:20
**  Description: 
*/

var traceLogger = taslog.GetLogger(taslog.PerformTraceConfig)

const dateFormte = "2006-01-02 15:04:05.000"

type PerformTraceLogger struct {
	Name 		string
	Hash 		string
	Height 		uint64
	Begin 		time.Time
	End 		time.Time
	OperTime 	time.Time
	Parent 		string
	Desc 		string
	TxNum 		int
}

type blockTraceLogs struct {
	enable bool
	logs *lru.Cache
}

var btlogs = &blockTraceLogs{
	logs: common.MustNewLRUCache(2000),
}

func InitPerformTraceLogger() {
	btlogs.enable = true
	notify.BUS.Subscribe(notify.BlockAddSucc, btlogs.onBlockAddSuccess)
}

func (btl *blockTraceLogs) addLog(log *PerformTraceLogger)  {
	if !btl.enable {
		return
	}
	if v, ok := btl.logs.Get(log.Hash); ok {
		logs := v.([]*PerformTraceLogger)
		logs = append(logs, log)
		btl.logs.Add(log.Hash, logs)
	} else {
		logs := make([]*PerformTraceLogger, 0)
		logs = append(logs, log)
		btl.logs.Add(log.Hash, logs)
	}
}

func (btl *blockTraceLogs) onBlockAddSuccess(message notify.Message)  {
	block := message.GetData().(*types.Block)

	hash := block.Header.Hash.String()
	if v, ok := btl.logs.Get(hash); ok {
		logs := v.([]*PerformTraceLogger)
		for _, log := range logs {
			bs, _ := json.Marshal(log)
			traceLogger.Infof(string(bs))
		}
		btl.logs.Remove(hash)
	}
}


func NewPerformTraceLogger(name string, hash common.Hash, height uint64) *PerformTraceLogger {
    return &PerformTraceLogger{
    	Name: name,
    	Hash: hash.String(),
    	Height: height,
    	Begin: time.Now(),
    	End: 	time.Unix(0, 0),
	}
}

func (ti *PerformTraceLogger) SetHash(hash common.Hash)  {
    ti.Hash = hash.String()
}

func (ti *PerformTraceLogger) SetHeight(h uint64)  {
	ti.Height = h
}

func (ti *PerformTraceLogger) SetEnd()  {
    ti.End = time.Now()
}
func (ti *PerformTraceLogger) SetParent(p string)  {
	ti.Parent = p
}

func (ti *PerformTraceLogger) SetTxNum(num int)  {
    ti.TxNum = num
}
func (ti *PerformTraceLogger) Log(format string, params ...interface{})  {
	if format != "" {
		ti.Desc = fmt.Sprintf(format, params...)
	}
	if ti.End.Unix() == 0 {
		ti.End = time.Now()
	}
	ti.OperTime = time.Now()
	btlogs.addLog(ti)
	//traceLogger.Infof("%v [%v]Hash:%v,Height:%v,Cost:%v,Desc:%v", ti.Begin.Format(dateFormte), ti.Name, ti.Hash, ti.Height,  ti.End.Sub(ti.Begin).String(), ti.Desc)
}

