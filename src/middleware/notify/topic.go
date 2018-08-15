package notify

import (
	"sync"
	"reflect"
)

type Message interface {
	GetRaw() []byte
	GetData() interface{}
}

type DummyMessage struct {
}

func (d *DummyMessage) GetRaw() []byte {
	return []byte{}
}
func (d *DummyMessage) GetData() interface{} {
	return struct{}{}
}

type Handler func(message Message)

// 消息订阅
type Topic struct {
	Id       string
	handlers []Handler
	lock     sync.RWMutex
}

func (topic *Topic) Subscribe(h Handler) {
	topic.lock.Lock()
	defer topic.lock.Unlock()

	for _, handler := range topic.handlers {
		if reflect.ValueOf(handler) == reflect.ValueOf(h) {
			return
		}
	}
	topic.handlers = append(topic.handlers, h)
}

func (topic *Topic) UnSubscribe(h Handler) {
	topic.lock.Lock()
	defer topic.lock.Unlock()

	for i, handler := range topic.handlers {
		if reflect.ValueOf(handler) == reflect.ValueOf(h) {
			topic.handlers = append(topic.handlers[:i], topic.handlers[i+1:]...)
			return
		}
	}
}

func (topic *Topic) Handle(message Message) {
	if 0 == len(topic.handlers) {
		return
	}

	topic.lock.RLock()
	defer topic.lock.RUnlock()
	for _, h := range topic.handlers {
		go h(message)
	}
}
