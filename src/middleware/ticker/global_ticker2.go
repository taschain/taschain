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

package ticker

import (
	"time"
	"sync/atomic"
	"sync"
	"runtime/debug"
	"common"
)

type RoutineFunc func() bool

const (
	STOPPED = int32(0)
	RUNNING = int32(1)
)

const (
	rtypePreiodic = 1
	rtypeOneTime = 2
)

type TickerRoutine struct {
	id              string
	handler         RoutineFunc   //执行函数
	interval        uint32        //触发的心跳间隔
	lastTicker      uint64        //最后一次执行的心跳
	//triggerCh       chan int32 		//触发信号
	status          int32         //当前状态 STOPPED, RUNNING
	triggerNextTick int32         //下次心跳触发
	rtype 			int8
}

type GlobalTicker struct {
	beginTime time.Time
	timer 	*time.Ticker
	ticker  uint64
	id 		string
	routines sync.Map	//string -> *TickerRoutine
	//routines map[string]*TickerRoutine
}

func NewGlobalTicker(id string) *GlobalTicker {
	ticker := &GlobalTicker{
		id: id,
		beginTime: time.Now(),
		//routines: make(map[string]*TickerRoutine),
	}

	go ticker.routine()

	return ticker
}

func (gt *GlobalTicker) addRoutine(name string, tr *TickerRoutine)  {
    gt.routines.Store(name, tr)
}

func (gt *GlobalTicker) getRoutine(name string) *TickerRoutine {
	if v, ok := gt.routines.Load(name); ok {
		return v.(*TickerRoutine)
	}
	return nil
}

/**
* @Description: 触发一次执行
* @Param: chanVal, 1 表示定时器定时触发, 若本次ticker已执行过,则忽略; 2表示需要立即执行, 若本次ticker已执行过, 则延迟到下一次ticker执行
* @return: bool
*/
func (gt *GlobalTicker) trigger(routine *TickerRoutine) bool {
	defer func() {
		if r := recover(); r != nil {
			common.DefaultLogger.Errorf("error：%v\n", r)
			s := debug.Stack()
			common.DefaultLogger.Errorf(string(s))
		}
	}()

	t := gt.ticker
	lastTicker := atomic.LoadUint64(&routine.lastTicker)

	if atomic.LoadInt32(&routine.status) != RUNNING {
		//stdLo("ticker routine already stopped!, trigger return")
		return false
	}

	b := false
	if lastTicker < t && atomic.CompareAndSwapUint64(&routine.lastTicker, lastTicker, t) {
		//log.Printf("ticker routine begin, id=%v, globalticker=%v\n", routine.id, t)
		b = routine.handler()
	}
	return b
}

func (gt *GlobalTicker) routine() {
	gt.timer = time.NewTicker(1 * time.Second)
	for range gt.timer.C {
		gt.ticker++
		gt.routines.Range(func(key, value interface{}) bool {
			rt := value.(*TickerRoutine)
			if (atomic.LoadInt32(&rt.status) == RUNNING && gt.ticker - rt.lastTicker >= uint64(rt.interval)) || atomic.LoadInt32(&rt.triggerNextTick) == 1 {
				//rt.lastTicker = gt.ticker
				atomic.CompareAndSwapInt32(&rt.triggerNextTick, 1, 0)
				go gt.trigger(rt)
			}
			return true
		})
	}
}

func (gt *GlobalTicker) RegisterPeriodicRoutine(name string, routine RoutineFunc, interval uint32)  {
	//log.Printf("RegisterPeriodicRoutine, id=%v, interval=%v\n", name, interval)
	if rt := gt.getRoutine(name); rt != nil {
		//log.Printf("RegisterPeriodicRoutine, id=%v already exist!\n", name)
		return
	}
	r := &TickerRoutine{
		rtype: 			rtypePreiodic,
		interval:        interval,
		handler:         routine,
		lastTicker:      gt.ticker,
		id:              name,
		status:          STOPPED,
		triggerNextTick: 0,
	}
	gt.addRoutine(name, r)
}

func (gt *GlobalTicker) RegisterOneTimeRoutine(name string, routine RoutineFunc, delay uint32)  {
	//log.Printf("RegisterPeriodicRoutine, id=%v, interval=%v\n", name, interval)
	if rt := gt.getRoutine(name); rt != nil {
		rt.lastTicker = gt.ticker
		return
	}

	r := &TickerRoutine{
		rtype: 			rtypeOneTime,
		interval:        delay,
		handler:         routine,
		lastTicker:      gt.ticker,
		id:              name,
		status:          RUNNING,
		triggerNextTick: 0,
	}
	gt.addRoutine(name, r)
}

func (gt *GlobalTicker) RemoveRoutine(name string)  {
	gt.routines.Delete(name)
}

func (gt *GlobalTicker) StartTickerRoutine(name string, triggerNextTicker bool)  {
	routine := gt.getRoutine(name)
	if routine == nil {
		return
	}
	if triggerNextTicker && atomic.CompareAndSwapInt32(&routine.triggerNextTick, 0, 1) {
		//log.Printf("routine will trigger next routine! id=%v\n", routine.id)
	}
	if atomic.CompareAndSwapInt32(&routine.status, STOPPED, RUNNING) {
		//log.Printf("routine started! id=%v\n", routine.id)
	} else {
		//log.Printf("routine routine start failed, already in running! id=%v\n", routine.id)
	}
}

func (gt *GlobalTicker) StartAndTriggerRoutine(name string)  {
	routine := gt.getRoutine(name)
	if routine == nil {
		return
	}

	if atomic.CompareAndSwapInt32(&routine.status, STOPPED, RUNNING) {
		//log.Printf("StartAndTriggerRoutine: routine started! id=%v\n", routine.id)
	} else {
		//log.Printf("StartAndTriggerRoutine:routine routine start failed, already in running! id=%v\n", routine.id)
	}
}

func (gt *GlobalTicker) StopTickerRoutine(name string)  {
	routine := gt.getRoutine(name)
	if routine == nil {
		return
	}

	if atomic.CompareAndSwapInt32(&routine.status, RUNNING, STOPPED) {
		//log.Printf("routine stopped! id=%v\n", routine.id)
	} else {
		//log.Printf("routine routine stop failed, not in running! id=%v\n", routine.id)
	}
}