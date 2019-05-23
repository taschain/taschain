package monitor

import (
	"time"
	"common"
	"fmt"
	"taslog"
)

/*
**  Creator: pxf
**  Date: 2019/5/23 下午2:20
**  Description: 
*/

var traceLogger = taslog.GetLogger(taslog.PerformTraceConfig)

const dateFormte = "2006-01-02 15:04:05"

type PerformTraceLogger struct {
	Name 		string
	Hash 		string
	Height 		uint64
	Begin 		time.Time
	Desc 		string
}

func NewPerformTraceLogger(name string, hash common.Hash, height uint64) *PerformTraceLogger {
    return &PerformTraceLogger{
    	Name: name,
    	Hash: hash.String(),
    	Height: height,
    	Begin: time.Now(),
	}
}

func (ti *PerformTraceLogger) SetHash(hash common.Hash)  {
    ti.Hash = hash.String()
}

func (ti *PerformTraceLogger) SetHeight(h uint64)  {
	ti.Height = h
}

func (ti *PerformTraceLogger) Log(format string, params ...interface{})  {
	if format != "" {
		ti.Desc = fmt.Sprintf(format, params...)
	}

	traceLogger.Infof("%v [%v]Hash:%v,Height:%v,Cost:%v,Desc:%v", ti.Begin.Format(dateFormte), ti.Name, ti.Hash, ti.Height,  time.Since(ti.Begin).String(), ti.Desc)
}

