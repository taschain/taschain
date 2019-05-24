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

package middleware

import (
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/taslog"
	"sync"
	"time"
)

type Loglock struct {
	lock  sync.RWMutex
	addr  string
	begin time.Time
}

var lockLogger taslog.Logger

const costLimit = 10 * time.Microsecond
const durationLimit = time.Millisecond

func NewLoglock(title string) Loglock {
	loglock := Loglock{
		lock: sync.RWMutex{},
	}
	loglock.addr = fmt.Sprintf("%p", &loglock)
	if lockLogger == nil {
		lockLogger = taslog.GetLoggerByIndex(taslog.LockLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	}
	return loglock
}

func (lock *Loglock) Lock(msg string) {
	if 0 != len(msg) {
		lockLogger.Debugf("try to lock: %s, with msg: %s", lock.addr, msg)
	}
	begin := time.Now()
	lock.lock.Lock()
	lock.begin = time.Now()
	cost := time.Since(begin)

	if 0 != len(msg) && cost > costLimit {
		lockLogger.Debugf("locked: %s, with msg: %s wait: %v", lock.addr, msg, cost)
	}

}

func (lock *Loglock) RLock(msg string) {
	if 0 != len(msg) {
		lockLogger.Debugf("try to Rlock: %s, with msg: %s", lock.addr, msg)
	}
	begin := time.Now()
	lock.lock.RLock()
	cost := time.Since(begin)
	if 0 != len(msg) && cost > costLimit {
		lockLogger.Debugf("Rlocked: %s, with msg: %s wait: %v", lock.addr, msg, cost)
	}
}

func (lock *Loglock) Unlock(msg string) {
	if 0 != len(msg) {
		lockLogger.Debugf("try to UnLock: %s, with msg: %s", lock.addr, msg)
	}
	begin := time.Now()
	lock.lock.Unlock()
	duration := time.Since(lock.begin)
	cost := time.Since(begin)
	if 0 != len(msg) && (cost > costLimit || duration > durationLimit) {
		lockLogger.Debugf("UnLocked: %s, with msg: %s duration:%v wait: %v", lock.addr, msg, duration, cost)
	}

}

func (lock *Loglock) RUnlock(msg string) {
	if 0 != len(msg) {
		lockLogger.Debugf("try to UnRLock: %s, with msg: %s", lock.addr, msg)
	}
	begin := time.Now()
	lock.lock.RUnlock()
	cost := time.Since(begin)
	if 0 != len(msg) && cost > costLimit {
		lockLogger.Debugf("UnRLocked: %s, with msg: %s wait: %v", lock.addr, msg, cost)
	}

}
