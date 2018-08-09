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
	"log"
	"sync/atomic"
)

type RoutineFunc func() bool

const (
	STOPPED = int32(0)
	RUNNING = int32(1)
)

type TickerRoutine struct {
	id              string
	handler         RoutineFunc   //执行函数
	interval        uint32        //触发的心跳间隔
	lastTicker      uint64        //最后一次执行的心跳
	triggerCh       chan int32 //触发信号
	status          int32         //当前状态 STOPPED, RUNNING
	triggerNextTick int32         //下次心跳触发
}

type GlobalTicker struct {
	beginTime time.Time
	timer 	*time.Ticker
	ticker  uint64
	id 		string
	routines map[string]*TickerRoutine
}

func NewGlobalTicker(id string) *GlobalTicker {
	ticker := &GlobalTicker{
		id: id,
		beginTime: time.Now(),
		routines: make(map[string]*TickerRoutine),
	}

	go ticker.routine()

	return ticker
}

/**
* @Description: 触发一次执行
* @Param: chanVal, 1 表示定时器定时触发, 若本次ticker已执行过,则忽略; 2表示需要立即执行, 若本次ticker已执行过, 则延迟到下一次ticker执行
* @return: bool
*/
func (gt *GlobalTicker) trigger(routine *TickerRoutine, chanVal int32) bool {
	defer func() {
		//if err := recover(); err != nil {
		//	log.Printf("routine handler error! id=%v, err=%v\n", routine.id, err)
		//}
	}()
	t := gt.ticker
	lastTicker := atomic.LoadUint64(&routine.lastTicker)

	if atomic.LoadInt32(&routine.status) != RUNNING {
		log.Println("ticker routine already stopped!, trigger return")
		return false
	}

	b := false
	if lastTicker < t && atomic.CompareAndSwapUint64(&routine.lastTicker, lastTicker, t) {
		log.Printf("ticker routine begin, id=%v, globalticker=%v\n", routine.id, t)
		b = routine.handler()
	} else {
		if chanVal == 2 {
			atomic.CompareAndSwapInt32(&routine.triggerNextTick, 0, 1)
			log.Printf("ticker routine executed this ticker, will trigger next ticker! id=%v, globalticker=%v, lastTicker=%v, status=%v\n", routine.id, t, routine.lastTicker, routine.status)
		} else {
			log.Printf("ticker routine already executed this ticker! id=%v, globalticker=%v, lastTicker=%v, status=%v\n", routine.id, t, routine.lastTicker, routine.status)
		}
	}
	return b
}

func (gt *GlobalTicker) routine() {
	gt.timer = time.NewTicker(1 * time.Second)
	for range gt.timer.C {
		gt.ticker++
		for _, rt := range gt.routines {
			if (atomic.LoadInt32(&rt.status) == RUNNING && gt.ticker - rt.lastTicker >= uint64(rt.interval)) || atomic.LoadInt32(&rt.triggerNextTick) == 1 {
				//rt.lastTicker = gt.ticker
				atomic.CompareAndSwapInt32(&rt.triggerNextTick, 1, 0)
				rt.triggerCh <- 1
			}
		}
	}
}

func (gt *GlobalTicker) RegisterRoutine(name string, routine RoutineFunc, interval uint32)  {

	log.Printf("RegisterRoutine, id=%v, interval=%v\n", name, interval)
	if _, ok := gt.routines[name]; ok {
		log.Printf("RegisterRoutine, id=%v already exist!\n", name)
		return
	}
	r := &TickerRoutine{
		interval:        interval,
		handler:         routine,
		lastTicker:      0,
		id:              name,
		triggerCh:       make(chan int32, 5),
		status:          STOPPED,
		triggerNextTick: 0,
	}
	go func() {
		for {
			select {
			case val := <-r.triggerCh:
				gt.trigger(r, val)
			}
		}
	}()

	gt.routines[name] = r

	for k, _ := range gt.routines {
		log.Printf("global tickers %v", k)
	}
}


func (gt *GlobalTicker) StartTickerRoutine(name string, triggerNextTicker bool)  {
	ticker, ok := gt.routines[name]
	if !ok {
		return
	}
	if triggerNextTicker && atomic.CompareAndSwapInt32(&ticker.triggerNextTick, 0, 1) {
		log.Printf("ticker routine will trigger next ticker! id=%v\n", ticker.id)
	}
	if atomic.CompareAndSwapInt32(&ticker.status, STOPPED, RUNNING) {
		log.Printf("ticker routine started! id=%v\n", ticker.id)
	} else {
		//log.Printf("ticker routine start failed, already in running! id=%v\n", ticker.id)
	}
}

func (gt *GlobalTicker) StartAndTriggerRoutine(name string)  {
	ticker, ok := gt.routines[name]
	if !ok {
		return
	}

	if atomic.CompareAndSwapInt32(&ticker.status, STOPPED, RUNNING) {
		log.Printf("StartAndTriggerRoutine:ticker routine started! id=%v\n", ticker.id)
	} else {
		//log.Printf("StartAndTriggerRoutine:ticker routine start failed, already in running! id=%v\n", ticker.id)
	}
	go func() {
		ticker.triggerCh <- 2
	}()

}

func (gt *GlobalTicker) StopTickerRoutine(name string)  {
	ticker, ok := gt.routines[name]
	if !ok {
		return
	}

	if atomic.CompareAndSwapInt32(&ticker.status, RUNNING, STOPPED) {
		log.Printf("ticker routine stopped! id=%v\n", ticker.id)
	} else {
		//log.Printf("ticker routine stop failed, not in running! id=%v\n", ticker.id)
	}
}