package ticker

import (
	"time"
	"log"
)

type TickerRoutine func() bool

type GlobalTicker struct {

	intervalSec int	//间隔
	beginTime time.Time
	ticker 	*time.Ticker
	id 		string
	routines map[string]TickerRoutine
}

func NewGlobalTicker(id string, intervalSec int) *GlobalTicker {
	ticker := &GlobalTicker{
		id: id,
		intervalSec: intervalSec,
		beginTime: time.Now(),
		routines: make(map[string]TickerRoutine),
	}

	go ticker.routine()

	return ticker
}

func (gt *GlobalTicker) routine() {
	gt.ticker = time.NewTicker(time.Duration(gt.intervalSec) * time.Second)
	for range gt.ticker.C {
		log.Println("global ticker begin: id=", gt.id)
		for name, fun := range gt.routines {
			log.Println("ticker routine begin: name=", name)
			fun()
			log.Println("ticker routine end: name=", name)
		}
		log.Println("global ticker end: id=", gt.id)
	}
}

func (gt *GlobalTicker) RegisterRoutine(name string, routine TickerRoutine, triggerImmediately bool)  {
	if triggerImmediately {
		routine()
	}

	gt.routines[name] = routine
}

func (gt *GlobalTicker) RemoveRoutine(name string)  {
    delete(gt.routines, name)
}