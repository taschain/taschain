package taslog

import (
	"fmt"
	"strconv"
	"time"
)

var SlowLogger Logger

func InitSlowLogger(index int) {
	SlowLogger = GetLoggerByIndex(SlowLogConfig, strconv.FormatInt(int64(index), 10))
}

type BLog interface {
	Log(format string, params ...interface{})
}

type StageLogTime struct {
	stage string
	begin time.Time
	end   time.Time
}

type SlowLog struct {
	lts       []*StageLogTime
	begin     time.Time
	key       string
	threshold float64
}

func NewSlowLog(key string, thresholdSecs float64) *SlowLog {
	return &SlowLog{
		lts:       make([]*StageLogTime, 0),
		begin:     time.Now(),
		key:       key,
		threshold: thresholdSecs,
	}
}

func (log *SlowLog) AddStage(key string) {
	st := &StageLogTime{
		begin: time.Now(),
		stage: key,
	}
	log.lts = append(log.lts, st)
}

func (log *SlowLog) EndStage() {
	if len(log.lts) > 0 {
		st := log.lts[len(log.lts)-1]
		st.end = time.Now()
	}
}

func (log *SlowLog) Log(format string, params ...interface{}) {
	c := time.Since(log.begin)
	if c.Seconds() < log.threshold {
		return
	}
	s := fmt.Sprintf(format, params...)
	detail := ""
	for _, lt := range log.lts {
		if lt.end.Nanosecond() == 0 {
			continue
		}
		detail = fmt.Sprintf("%v,%v(%v)", detail, lt.stage, lt.end.Sub(lt.begin).String())
	}
	s = fmt.Sprintf("%v:%v,cost %v, detail %v", log.key, s, c.String(), detail)
	if SlowLogger != nil {
		SlowLogger.Warnf(s)
	} else {
		println(s)
	}
}
