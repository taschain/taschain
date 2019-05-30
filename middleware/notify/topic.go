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

package notify

import (
	"reflect"
	"sync"
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
