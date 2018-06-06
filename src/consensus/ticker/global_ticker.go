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
	triggerCh       chan struct{} //触发信号
	status          int32         //当前状态 STOPPED, RUNNING
	triggerNextTick int32         //下次心跳触发
	executing 		int32			//是否正在执行
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

func (gt *GlobalTicker) trigger(routine *TickerRoutine) bool {
	defer func() {
		//if err := recover(); err != nil {
		//	log.Printf("routine handler error! id=%v, err=%v\n", routine.id, err)
		//}
	}()
	t := gt.ticker
	log.Printf("ticker routine begin, id=%v, globalticker=%v\n", routine.id, t)
	atomic.SwapInt32(&routine.executing, 1)
	b := routine.handler()
	atomic.SwapInt32(&routine.executing, 0)
	log.Printf("ticker routine end, id=%v, result=%v, globalticker=%v\n", routine.id, b, t)
	return b
}

func (gt *GlobalTicker) routine() {
	gt.timer = time.NewTicker(1 * time.Second)
	for range gt.timer.C {
		gt.ticker++
		//log.Println("global ticker id", gt.id, " ticker", gt.ticker)
		for _, rt := range gt.routines {
			if (atomic.LoadInt32(&rt.status) == RUNNING && gt.ticker - rt.lastTicker >= uint64(rt.interval)) || atomic.LoadInt32(&rt.triggerNextTick) == 1 {
				rt.lastTicker = gt.ticker
				atomic.CompareAndSwapInt32(&rt.triggerNextTick, 1, 0)
				rt.triggerCh <- struct{}{}
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
		triggerCh:       make(chan struct{}, 5),
		status:          STOPPED,
		triggerNextTick: 0,
	}
	go func() {
		for {
			select {
			case <-r.triggerCh:
				gt.trigger(r)
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
		log.Printf("ticker routine start failed, already in running! id=%v\n", ticker.id)
	}
}


func (gt *GlobalTicker) StopTickerRoutine(name string)  {
	ticker, ok := gt.routines[name]
	if !ok {
		return
	}

	if atomic.CompareAndSwapInt32(&ticker.status, RUNNING, STOPPED) {
		log.Printf("ticker routine stopped! id=%v\n", ticker.id)
	} else {
		log.Printf("ticker routine stop failed, not in running! id=%v\n", ticker.id)
	}
}