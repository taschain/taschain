package statistics

import (
	"time"
	"taslog"
	"common"
	"bytes"
	"fmt"
	"sync"
)

var count_map = make(map[string]*sync.Map)
var logger taslog.Logger

func printAndRefresh()  {
	for key,vmap := range count_map{
		count_map[key] = &sync.Map{}
		var buffer bytes.Buffer
		vmap.Range(func(code, value interface{}) bool {
			buffer.WriteString(fmt.Sprintf(" %d:%d",code,value))
			return true
		})
		logger.Infof("%s%s\n", key, buffer.String())
		//if len(vmap) > 0{
		//	pmap := vmap
		//	count_map[key] = make(map[uint32]uint32)
		//	var buffer bytes.Buffer
		//	for code,value := range pmap {
		//		buffer.WriteString(fmt.Sprintf(" %d:%d",code,value))
		//	}
		//	logger.Infof("%s%s\n", key, buffer.String())
		//	//fmt.Printf("%s %d %d\n", key, code, value)
		//}
	}
}

func AddCount(name string, code uint32)  {
	if vmap,ok := count_map[name];ok{
		if v,ok2 := vmap.Load(code);ok2{
			vmap.Store(code,v.(uint32) + uint32(1))
		} else {
			vmap.Store(code,uint32(1))
		}
	} else {
		vmap = &sync.Map{}
		vmap.Store(code,uint32(1))
		count_map[name] = vmap
	}
	//logger.Infof("%s %d",name,code)
}

func initCount(config common.ConfManager) {
	logger = taslog.GetLoggerByName("statistics" + config.GetString("instance", "index", ""))
	t1 := time.NewTimer(time.Second * 1)
	go func() {
		for {
			select {
			case <-t1.C:
				printAndRefresh()
				t1.Reset(time.Second * 1)
			}
		}
	}()
}
