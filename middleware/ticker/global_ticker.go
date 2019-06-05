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
	"github.com/taschain/taschain/common"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type RoutineFunc func() bool

const (
	STOPPED = int32(0)
	RUNNING = int32(1)
)

const (
	rtypePreiodic = 1
	rtypeOneTime  = 2
)

type TickerRoutine struct {
	id              string
	handler         RoutineFunc // Executive function
	interval        uint32      // Triggered heartbeat interval
	lastTicker      uint64      // Last executed heartbeat
	status          int32       // The current state : STOPPED, RUNNING
	triggerNextTick int32       // Next heartbeat
	rtype           int8
}

type GlobalTicker struct {
	beginTime time.Time
	timer     *time.Ticker
	ticker    uint64
	id        string
	routines  sync.Map // key: string, value: *TickerRoutine
}

func NewGlobalTicker(id string) *GlobalTicker {
	ticker := &GlobalTicker{
		id:        id,
		beginTime: time.Now(),
	}

	go ticker.routine()

	return ticker
}

func (gt *GlobalTicker) addRoutine(name string, tr *TickerRoutine) {
	gt.routines.Store(name, tr)
}

func (gt *GlobalTicker) getRoutine(name string) *TickerRoutine {
	if v, ok := gt.routines.Load(name); ok {
		return v.(*TickerRoutine)
	}
	return nil
}

// trigger trigger an execution
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
		return false
	}

	b := false
	if lastTicker < t && atomic.CompareAndSwapUint64(&routine.lastTicker, lastTicker, t) {
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
			if (atomic.LoadInt32(&rt.status) == RUNNING && gt.ticker-rt.lastTicker >= uint64(rt.interval)) || atomic.LoadInt32(&rt.triggerNextTick) == 1 {
				atomic.CompareAndSwapInt32(&rt.triggerNextTick, 1, 0)
				go gt.trigger(rt)
			}
			return true
		})
	}
}

func (gt *GlobalTicker) RegisterPeriodicRoutine(name string, routine RoutineFunc, interval uint32) {
	if rt := gt.getRoutine(name); rt != nil {
		return
	}
	r := &TickerRoutine{
		rtype:           rtypePreiodic,
		interval:        interval,
		handler:         routine,
		lastTicker:      gt.ticker,
		id:              name,
		status:          STOPPED,
		triggerNextTick: 0,
	}
	gt.addRoutine(name, r)
}

func (gt *GlobalTicker) RegisterOneTimeRoutine(name string, routine RoutineFunc, delay uint32) {
	if rt := gt.getRoutine(name); rt != nil {
		rt.lastTicker = gt.ticker
		return
	}

	r := &TickerRoutine{
		rtype:           rtypeOneTime,
		interval:        delay,
		handler:         routine,
		lastTicker:      gt.ticker,
		id:              name,
		status:          RUNNING,
		triggerNextTick: 0,
	}
	gt.addRoutine(name, r)
}

func (gt *GlobalTicker) RemoveRoutine(name string) {
	gt.routines.Delete(name)
}

func (gt *GlobalTicker) StartTickerRoutine(name string, triggerNextTicker bool) {
	routine := gt.getRoutine(name)
	if routine == nil {
		return
	}
	atomic.CompareAndSwapInt32(&routine.triggerNextTick, 0, 1)
	atomic.CompareAndSwapInt32(&routine.status, STOPPED, RUNNING)
}

func (gt *GlobalTicker) StartAndTriggerRoutine(name string) {
	routine := gt.getRoutine(name)
	if routine == nil {
		return
	}

	atomic.CompareAndSwapInt32(&routine.status, STOPPED, RUNNING)
}

func (gt *GlobalTicker) StopTickerRoutine(name string) {
	routine := gt.getRoutine(name)
	if routine == nil {
		return
	}

	atomic.CompareAndSwapInt32(&routine.status, RUNNING, STOPPED)
}
