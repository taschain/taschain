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
	"sync"
	"fmt"
	"taslog"
	"common"
	"time"
)

type Loglock struct {
	lock   sync.RWMutex
	addr   string
	logger taslog.Logger
	begin	time.Time
}

func NewLoglock(title string) Loglock {
	loglock := Loglock{
		lock:   sync.RWMutex{},
		logger: taslog.GetLoggerByName(title + common.GlobalConf.GetString("instance", "index", "")),
	}
	loglock.addr = fmt.Sprintf("%p", &loglock)
	return loglock
}

func (lock *Loglock) Lock(msg string) {
	if 0 != len(msg) {
		lock.logger.Debugf("try to lock: %s, with msg: %s", lock.addr, msg)
	}
	lock.lock.Lock()
	lock.begin = time.Now()
	if 0 != len(msg) {
		lock.logger.Debugf("locked: %s, with msg: %s", lock.addr, msg)
	}

}

func (lock *Loglock) RLock(msg string) {
	if 0 != len(msg) {
		lock.logger.Debugf("try to Rlock: %s, with msg: %s", lock.addr, msg)
	}
	lock.lock.RLock()
	if 0 != len(msg) {
		lock.logger.Debugf("Rlocked: %s, with msg: %s", lock.addr, msg)
	}
}

func (lock *Loglock) Unlock(msg string) {
	if 0 != len(msg) {
		lock.logger.Debugf("try to UnLock: %s, with msg: %s", lock.addr, msg)
	}
	lock.lock.Unlock()
	duration := time.Since(lock.begin)
	if 0 != len(msg) {
		lock.logger.Debugf("UnLocked: %s, with msg: %s duration:%v", lock.addr, msg, duration)
	}

}

func (lock *Loglock) RUnlock(msg string) {
	if 0 != len(msg) {
		lock.logger.Debugf("try to UnRLock: %s, with msg: %s", lock.addr, msg)
	}
	lock.lock.RUnlock()
	if 0 != len(msg) {
		lock.logger.Debugf("UnRLocked: %s, with msg: %s", lock.addr, msg)
	}

}
