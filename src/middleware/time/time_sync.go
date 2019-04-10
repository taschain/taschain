package time

import (
	"time"
	"middleware/ticker"
	"github.com/beevik/ntp"
	"math/rand"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2019/4/10 上午11:15
**  Description: 
*/

var natServer = []string{"ntp.aliyun.com","ntp1.aliyun.com", "ntp2.aliyun.com","ntp3.aliyun.com","ntp4.aliyun.com","ntp4.aliyun.com","ntp5.aliyun.com","ntp6.aliyun.com","ntp7.aliyun.com"}

type TimeSync struct {
	currentOffset time.Duration
	ticker *ticker.GlobalTicker
}



type TimeService interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	NowAfter(t time.Time) bool
}

var TSInstance TimeService

func InitTimeSync() {
	ts :=&TimeSync{
		currentOffset: 0,
		ticker: ticker.NewGlobalTicker("time_sync"),
	}

	ts.ticker.RegisterPeriodicRoutine("time_sync", ts.syncRoutine, 60)
	ts.ticker.StartTickerRoutine("time_sync", false)
	ts.syncRoutine()
	TSInstance = ts
}

func (ts *TimeSync) syncRoutine() bool {
	r := rand.Intn(len(natServer))
	rsp, err := ntp.Query(natServer[r])
	if err != nil {
		fmt.Printf("time sync from %v err: %v\n", natServer[r], err)
		return false
	}
	ts.currentOffset = rsp.ClockOffset
	fmt.Printf("time offset from %v is %v\n", natServer[r], ts.currentOffset.String())
	return true
}

func (ts *TimeSync) Now() time.Time {
	return time.Now().Add(ts.currentOffset)
}

func (ts *TimeSync) Since(t time.Time) time.Duration {
	return ts.Now().Sub(t)
}

func (ts *TimeSync) NowAfter(t time.Time) bool {
    return ts.Now().After(t)
}