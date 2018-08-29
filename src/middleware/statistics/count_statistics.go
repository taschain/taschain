package statistics

import (
	"time"
	"taslog"
	"common"
	"bytes"
	"fmt"
	"sync"
)

type countItem struct {
	lock *sync.RWMutex
	innerMap *sync.Map
}

var count_map = make(map[string]*countItem)
var logger taslog.Logger

func newCountItem() *countItem {
	return &countItem{lock:new(sync.RWMutex),innerMap:new(sync.Map)}
}

func (item *countItem)get(code uint32) uint32 {
	item.lock.RLock()
	defer item.lock.RUnlock()
	if v,ok2 := item.innerMap.Load(code);ok2{
		return v.(uint32)
	} else{
		return 0
	}
}

func (item *countItem)set(code uint32,value uint32) {
	item.lock.RLock()
	defer item.lock.RUnlock()
	item.innerMap.Store(code, value)
}

func (item *countItem)print() string{
	item.lock.Lock()
	defer item.lock.Unlock()
	var buffer bytes.Buffer
	item.innerMap.Range(func(code, value interface{}) bool {
		buffer.WriteString(fmt.Sprintf(" %d:%d",code,value))
		item.innerMap.Delete(code)
		return true
	})
	return buffer.String()
}

func printAndRefresh()  {
	for key,item := range count_map{
		content := item.print()
		logger.Infof("%s%s\n", key, content)
		//fmt.Printf("%s%s\n", key, content)
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
	if item,ok := count_map[name];ok{
		newValue := item.get(code) + 1
		item.set(code, newValue)
	} else {
		item = newCountItem()
		item.set(code, 1)
		count_map[name] = item
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
