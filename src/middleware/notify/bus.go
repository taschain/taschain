package notify

import (
	"sync"
)

var BUS *Bus
/*
	内部消息订阅服务
 */
type Bus struct {
	topics map[string]*Topic
	lock   sync.RWMutex
}

func NewBus() *Bus {
	return &Bus{
		lock:   sync.RWMutex{},
		topics: make(map[string]*Topic, 10),
	}
}

func (bus *Bus) Subscribe(id string, handler Handler) {
	bus.lock.Lock()
	defer bus.lock.Unlock()

	topic, ok := bus.topics[id]
	if !ok {
		topic = &Topic{
			Id: id,
		}
		bus.topics[id] = topic
	}

	topic.Subscribe(handler)
}

func (bus *Bus) UnSubscribe(id string, handler Handler) {
	bus.lock.RLock()
	defer bus.lock.RUnlock()

	topic, ok := bus.topics[id]
	if !ok {
		return
	}

	topic.UnSubscribe(handler)
}

func (bus *Bus) Publish(id string, message Message) {
	bus.lock.RLock()
	defer bus.lock.RUnlock()

	topic, ok := bus.topics[id]
	if !ok {
		return
	}

	topic.Handle(message)
}
