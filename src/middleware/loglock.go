package middleware

import (
	"sync"
	"fmt"
	"taslog"
	"common"
)

type Loglock struct {
	lock   sync.RWMutex
	addr   string
	logger taslog.Logger
}

func NewLoglock() Loglock {
	module := "chainlock"
	if common.GlobalConf != nil {
		module = common.GlobalConf.GetString("chain", "database", "chainlock")
	}

	var prefix = `<seelog minlevel="debug">
		<outputs formatid="lockConfig">
			<rollingfile type="size" filename="./logs/chainlock-`
	var suffix = `.log" maxsize="100000" maxrolls="3"/>
		</outputs>
		<formats>
			<format id="lockConfig" format="%Date/%Time [%Level]  [%File:%Line] %Msg%n" />
		</formats>
	</seelog>`

	loglock := Loglock{
		lock:   sync.RWMutex{},
		logger: taslog.GetLogger(prefix + module + suffix),
	}
	loglock.addr = fmt.Sprintf("%p", &loglock)
	return loglock
}

func (lock *Loglock) Lock(msg string) {
	lock.logger.Debugf("try to lock: %s, with msg: %s", lock.addr, msg)
	lock.lock.Lock()
	lock.logger.Debugf("locked: %s, with msg: %s", lock.addr, msg)

}

func (lock *Loglock) RLock(msg string) {
	lock.logger.Debugf("try to Rlock: %s, with msg: %s", lock.addr, msg)
	lock.lock.RLock()
	lock.logger.Debugf("Rlocked: %s, with msg: %s", lock.addr, msg)
}

func (lock *Loglock) Unlock(msg string) {
	lock.logger.Debugf("try to UnLock: %s, with msg: %s", lock.addr, msg)
	lock.lock.Unlock()
	lock.logger.Debugf("UnLocked: %s, with msg: %s", lock.addr, msg)

}

func (lock *Loglock) RUnlock(msg string) {
	lock.logger.Debugf("try to UnRLock: %s, with msg: %s", lock.addr, msg)
	lock.lock.RUnlock()
	lock.logger.Debugf("UnRLocked: %s, with msg: %s", lock.addr, msg)

}
