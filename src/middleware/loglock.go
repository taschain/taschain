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

func NewLoglock(title string) Loglock {
	module := "0"
	if common.GlobalConf != nil {
		module = common.GlobalConf.GetString("chain", "database", "0")
	}

	var prefix = `<seelog minlevel="error">
		<outputs formatid="lockConfig">
			<rollingfile type="size" filename="./logs/`
	var suffix = `.log" maxsize="1000000" maxrolls="3"/>
		</outputs>
		<formats>
			<format id="lockConfig" format="%Date/%Time [%Level] [%File:%Line] %Msg%n" />
		</formats>
	</seelog>`

	loglock := Loglock{
		lock:   sync.RWMutex{},
		logger: taslog.GetLogger(prefix + title + module + suffix),
	}
	loglock.addr = fmt.Sprintf("%p", &loglock)
	return loglock
}

func (lock *Loglock) Lock(msg string) {
	if 0==len(msg){
		lock.logger.Debugf("try to lock: %s, with msg: %s", lock.addr, msg)
	}
	lock.lock.Lock()
	if 0==len(msg){
		lock.logger.Debugf("locked: %s, with msg: %s", lock.addr, msg)
	}

}

func (lock *Loglock) RLock(msg string) {
	if 0==len(msg){
		lock.logger.Debugf("try to Rlock: %s, with msg: %s", lock.addr, msg)
	}
	lock.lock.RLock()
	if 0==len(msg){
		lock.logger.Debugf("Rlocked: %s, with msg: %s", lock.addr, msg)
	}
}

func (lock *Loglock) Unlock(msg string) {
	if 0==len(msg){
		lock.logger.Debugf("try to UnLock: %s, with msg: %s", lock.addr, msg)
	}
	lock.lock.Unlock()
	if 0==len(msg){
		lock.logger.Debugf("UnLocked: %s, with msg: %s", lock.addr, msg)
	}

}

func (lock *Loglock) RUnlock(msg string) {
	if 0==len(msg){
		lock.logger.Debugf("try to UnRLock: %s, with msg: %s", lock.addr, msg)
	}
	lock.lock.RUnlock()
	if 0==len(msg){
		lock.logger.Debugf("UnRLocked: %s, with msg: %s", lock.addr, msg)
	}

}
