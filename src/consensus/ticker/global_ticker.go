package ticker

import (
	"time"
	"log"
	"sync"
)

type RoutineFunc func() bool

type TickerRoutine struct {
	id string
	handler RoutineFunc
	interval uint32
	lastTicker uint64
	triggerCh chan struct{}
	stopCh 	chan struct{}
}

type GlobalTicker struct {
	beginTime time.Time
	timer 	*time.Ticker
	ticker  uint64
	id 		string
	routines map[string]*TickerRoutine
	mu sync.Mutex
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

func (gt *GlobalTicker) getRoutines() map[string]*TickerRoutine {
	gt.mu.Lock()
	defer gt.mu.Unlock()

    tmp := make(map[string]*TickerRoutine)
	for id, value := range gt.routines {
		tmp[id] = value
	}
	return tmp
}

func (gt *GlobalTicker) trigger(routine *TickerRoutine) bool {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("routine handler error! id=%v, err=%v\n", routine.id, err)
		}
	}()
	log.Printf("ticker routine begin, id=%v\n", routine.id)
	b := routine.handler()
	log.Printf("ticker routine end, id=%v, result=%v\n", routine.id, b)
	return b
}

func (gt *GlobalTicker) routine() {
	gt.timer = time.NewTicker(1 * time.Second)
	for range gt.timer.C {
		gt.ticker++
		log.Println("global ticker id", gt.id, " ticker", gt.ticker)

		for _, rt := range gt.getRoutines() {
			if gt.ticker - rt.lastTicker >= uint64(rt.interval) {
				rt.lastTicker = gt.ticker
				rt.triggerCh <- struct{}{}
			}
		}
	}
}

func (gt *GlobalTicker) RegisterRoutine(name string, routine RoutineFunc, interval uint32, triggerImmediately bool)  {
	gt.RemoveRoutine(name)

	r := &TickerRoutine{
		interval: interval,
		handler: routine,
		lastTicker: 0,
		id: name,
		triggerCh: make(chan struct{}),
		stopCh: make(chan struct{}),
	}
	go func() {
		for {
			select {
			case <-r.triggerCh:
				gt.trigger(r)
			case <-r.stopCh:
				log.Printf("ticker routine stopped! id=%v\n", r.id)
				return
			}
		}
	}()

	if triggerImmediately {
		r.triggerCh <- struct{}{}
	}
	gt.mu.Lock()
	defer gt.mu.Unlock()

	gt.routines[name] = r


}

func (gt *GlobalTicker) RemoveRoutine(name string)  {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	r ,ok := gt.routines[name]
	if !ok {
		return
	}

	r.stopCh <- struct{}{}
    delete(gt.routines, name)
}