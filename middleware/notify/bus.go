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
	"sync"
)

var BUS *Bus

// Bus is internal message subscription service
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
			ID: id,
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
