package logical

import (
	"log"
	"fmt"
	"time"
	"consensus/groupsig"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/8/31 下午4:40
**  Description: 
*/

type bLog interface {
	log(format string, params ...interface{})
}

//业务标准输出日志
type bizLog struct {
	biz string
}

func newBizLog(biz string) *bizLog {
	return &bizLog{biz: biz}
}

func (bl *bizLog) log(format string, p ...interface{})  {
    log.Printf("%v,%v:%v\n", time.Now().Format(TIMESTAMP_LAYOUT),bl.biz, fmt.Sprintf(format, p...))
}

//接口rt日志
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

const TIMESTAMP_LAYOUT = "2006-01-02/15:04:05.000"

func (r *rtLog) log(format string, p ...interface{})  {
	if time.Since(r.start).Nanoseconds() > 5000000 {
		log.Printf(fmt.Sprintf("%v:%v cost %v. %v", time.Now().Format(TIMESTAMP_LAYOUT), r.key, time.Since(r.start).String(), fmt.Sprintf(format, p...)))
	}
}

func (r *rtLog) end()  {
	log.Printf(fmt.Sprintf("%v:%v cost ", time.Now().Format(TIMESTAMP_LAYOUT), r.key))
}

//消息追踪日志，记录到文件
type msgTraceLog struct {
	mtype string	//消息类型
	key   string	//关键字
	sender string	//消息发送者
}

func newMsgTraceLog(mtype string, key string, sender string) *msgTraceLog {
	return &msgTraceLog{
		mtype:mtype,
		key:key,
		sender:sender,
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

func (mtl *msgTraceLog) log(format string, params ... interface{})  {
	_doLog(mtl.mtype, mtl.key, mtl.sender, format, params...)
}

func (mtl *msgTraceLog) logStart(format string, params ... interface{})  {
	_doLog(mtl.mtype + "-begin", mtl.key, mtl.sender, format, params...)
}

func (mtl *msgTraceLog) logEnd(format string, params ... interface{})  {
	_doLog(mtl.mtype + "-end", mtl.key, mtl.sender, format, params...)
}

