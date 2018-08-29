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

var count_map = new(sync.Map)
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
	count_map.Range(func(name, item interface{}) bool {
		citem := item.(*countItem)
		content := citem.print()
		logger.Infof("%s%s\n", name, content)
		//fmt.Printf("%s%s\n", name, content)
		return true
	})
}

func AddCount(name string, code uint32)  {
	if item,ok := count_map.Load(name);ok{
		citem := item.(*countItem)
		newValue := citem.get(code) + 1
		citem.set(code, newValue)
	} else {
		citem := newCountItem()
		citem.set(code, 1)
		count_map.Store(name, citem)
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
