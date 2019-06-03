//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package logical

import (
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
	"time"
)

/*
**  Creator: pxf
**  Date: 2018/8/31 下午4:40
**  Description:
 */

//业务标准输出日志
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

//接口rt日志
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

//消息追踪日志，记录到文件
type msgTraceLog struct {
	mtype  string //消息类型
	key    string //关键字
	sender string //消息发送者
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