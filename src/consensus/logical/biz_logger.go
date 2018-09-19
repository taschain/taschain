package logical

import (
	"log"
	"fmt"
	"time"
)

/*
**  Creator: pxf
**  Date: 2018/8/31 下午4:40
**  Description: 
*/

type bizLog struct {
	biz string
}

func newBizLog(biz string) *bizLog {
	return &bizLog{biz: biz}
}

func (bl *bizLog) log(format string, p ...interface{})  {
    log.Printf("%v:%v\n", bl.biz, fmt.Sprintf(format, p...))
}

type rtLog struct {
	start time.Time
	key string
}

func newRtLog(key string) *rtLog {
    return &rtLog{
    	start: time.Now(),
    	key: key,
	}
}

func (r *rtLog) log(format string, p ...interface{})  {
    log.Printf(fmt.Sprintf("%v:%v cost %v. %v", time.Now().Format(TIMESTAMP_LAYOUT), r.key, time.Since(r.start).String(), fmt.Sprintf(format, p...)))
}

func (r *rtLog) end()  {
	log.Printf(fmt.Sprintf("%v:%v cost %v. %v", time.Now().Format(TIMESTAMP_LAYOUT), r.key))
}