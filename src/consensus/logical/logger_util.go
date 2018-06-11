package logical

import (
	"fmt"
	"time"
)

/*
**  Creator: pxf
**  Date: 2018/6/8 上午9:52
**  Description: 
*/
const TIMESTAMP_LAYOUT = "2006-01-02/15:04:05.000"

func logStart(mtype string, height uint64, qn uint64, sender string, format string, params ...interface{}) {
	var s string
	if params == nil || len(params) == 0 {
		s = format
	} else {
		s = fmt.Sprintf(format, params...)
	}
	consensusLogger.Infof("%v,%v-begin,#%v-%v#,%v,%v", time.Now().Format(TIMESTAMP_LAYOUT), mtype, height, qn, sender, s)
}

func logEnd(mtype string, height uint64, qn uint64, sender string) {
	consensusLogger.Infof("%v,%v-end,#%v-%v#,%v,%v", time.Now().Format(TIMESTAMP_LAYOUT), mtype, height, qn, sender, "")
}


func logHalfway(mtype string, height uint64, qn uint64, sender string, format string, params ...interface{}) {
	var s string
	if params == nil || len(params) == 0 {
		s = format
	} else {
		s = fmt.Sprintf(format, params...)
	}
	consensusLogger.Infof("%v,%v-half,#%v-%v#,%v,%v", time.Now().Format(TIMESTAMP_LAYOUT), mtype, height, qn, sender, s)
}