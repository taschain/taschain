package utility

import (
	"fmt"
	"sync"
	"time"
)

/*
**  Creator: pxf
**  Date: 2019-05-29 11:11
**  Description:
 */

type TimeMetric struct {
	Total int
	Cost  time.Duration
}
type TimeStatCtx struct {
	Stats map[string]*TimeMetric
	lock sync.Mutex
}

func NewTimeStatCtx() *TimeStatCtx {
	return &TimeStatCtx{Stats: make(map[string]*TimeMetric)}
}

func (ts *TimeStatCtx) AddStat(name string, dur time.Duration) {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	if v, ok := ts.Stats[name]; ok {
		v.Cost += dur
		v.Total++
	} else {
		tm := &TimeMetric{}
		tm.Total++
		tm.Cost+= dur
		ts.Stats[name] = tm
	}
}

func (ts *TimeStatCtx) Output() string {
    s := ""
	for key, v := range ts.Stats {
		s += fmt.Sprintf("%v %v %v\n", key, v.Total, v.Cost.Seconds()/float64(v.Total))
	}
	return s
}