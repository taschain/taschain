package logical

import (
	"log"
	"fmt"
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