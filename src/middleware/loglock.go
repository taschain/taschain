package middleware

import (
	"sync"
	"log"
	"fmt"
)

type Loglock struct {
	lock sync.RWMutex
	addr string
}

func NewLoglock() Loglock {
	loglock := Loglock{}
	loglock.addr = fmt.Sprintf("%p", &loglock)
	return loglock
}

func (lock *Loglock) Lock(msg string) {
	log.Printf("try to lock: %s, with msg: %s", lock.addr, msg)
	lock.lock.Lock()
	log.Printf("locked: %s, with msg: %s", lock.addr, msg)

}

func (lock *Loglock) RLock(msg string) {
	log.Printf("try to Rlock: %s, with msg: %s", lock.addr, msg)
	lock.lock.RLock()
	log.Printf("Rlocked: %s, with msg: %s", lock.addr, msg)
}

func (lock *Loglock) Unlock(msg string) {
	log.Printf("try to UnLock: %s, with msg: %s", lock.addr, msg)
	lock.lock.Unlock()
	log.Printf("UnLocked: %s, with msg: %s", lock.addr, msg)

}

func (lock *Loglock) RUnlock(msg string) {
	log.Printf("try to UnRLock: %s, with msg: %s", lock.addr, msg)
	lock.lock.RUnlock()
	log.Printf("UnRLocked: %s, with msg: %s", lock.addr, msg)

}
