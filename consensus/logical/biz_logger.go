package logical

import (
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
	"time"
)

// bizLog is Business standard output log
type bizLog struct {
	biz string
}

func newBizLog(biz string) *bizLog {
	return &bizLog{biz: biz}
}

func (bl *bizLog) log(format string, p ...interface{}) {
	stdLogger.Infof("%v:%v", bl.biz, fmt.Sprintf(format, p...))
}

func (bl *bizLog) debug(format string, p ...interface{}) {
	stdLogger.Debugf("%v:%v", bl.biz, fmt.Sprintf(format, p...))
}

type rtLog struct {
	start time.Time
	key   string
}

func newRtLog(key string) *rtLog {
	return &rtLog{
		start: time.Now(),
		key:   key,
	}
}

const TimestampLayout = "2006-01-02/15:04:05.000"

func (r *rtLog) log(format string, p ...interface{}) {
	cost := time.Since(r.start)
	stdLogger.Debugf(fmt.Sprintf("%v:begin at %v, cost %v. %v", r.key, r.start.Format(TimestampLayout), cost.String(), fmt.Sprintf(format, p...)))
}

func (r *rtLog) end() {
	stdLogger.Debugf(fmt.Sprintf("%v:%v cost ", time.Now().Format(TimestampLayout), r.key))
}

// msgTraceLog encapsulate message tracking log, recorded to file
type msgTraceLog struct {
	mtype  string // Message type
	key    string // Keyword
	sender string // Message sender
}

func newMsgTraceLog(mtype string, key string, sender string) *msgTraceLog {
	return &msgTraceLog{
		mtype:  mtype,
		key:    key,
		sender: sender,
	}
}

func newHashTraceLog(mtype string, hash common.Hash, sid groupsig.ID) *msgTraceLog {
	return newMsgTraceLog(mtype, hash.ShortS(), sid.ShortS())
}

func _doLog(t string, k string, sender string, format string, params ...interface{}) {
	var s string
	if params == nil || len(params) == 0 {
		s = format
	} else {
		s = fmt.Sprintf(format, params...)
	}
	consensusLogger.Infof("%v,#%v#,%v,%v", t, k, sender, s)
}

func (mtl *msgTraceLog) log(format string, params ...interface{}) {
	_doLog(mtl.mtype, mtl.key, mtl.sender, format, params...)
}

func (mtl *msgTraceLog) logStart(format string, params ...interface{}) {
	_doLog(mtl.mtype+"-begin", mtl.key, mtl.sender, format, params...)
}

func (mtl *msgTraceLog) logEnd(format string, params ...interface{}) {
	_doLog(mtl.mtype+"-end", mtl.key, mtl.sender, format, params...)
}
