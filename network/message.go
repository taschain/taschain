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

package network

import (
	"hash/fnv"
	"sync"
	"time"
)

const BizMessageIdLength = 32

type BizMessageId = [BizMessageIdLength]byte

//MessageManager 消息管理
type MessageManager struct {
	messages      map[uint64]time.Time
	bizMessages   map[BizMessageId]time.Time
	index         uint32
	id            NodeID
	forwardNodeId uint32
	mutex         sync.Mutex
}

func decodeMessageInfo(info uint32) (chaidId uint16, protocolVersion uint16) {

	chaidId = uint16(info >> 16)
	protocolVersion = uint16(info)

	return chaidId, protocolVersion
}

func encodeMessageInfo(chaidId uint16, protocolVersion uint16) uint32 {

	return uint32(chaidId)<<16 | uint32(protocolVersion)
}

func newMessageManager(id NodeID) *MessageManager {

	mm := &MessageManager{
		messages:    make(map[uint64]time.Time),
		bizMessages: make(map[BizMessageId]time.Time),
	}
	mm.id = id
	mm.index = 0
	h := fnv.New32a()
	h.Write(id[:])
	mm.forwardNodeId = uint32(h.Sum32())
	return mm
}

//生成新的消息id
func (mm *MessageManager) genMessageId() uint64 {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.index += 1
	messageId := uint64(mm.forwardNodeId)
	messageId = messageId << 32
	messageId = messageId | uint64(mm.index)
	mm.messages[messageId] = time.Now()
	return messageId
}

func (mm *MessageManager) forward(messageId uint64) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.messages[messageId] = time.Now()
}

func (mm *MessageManager) isForwarded(messageId uint64) bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	_, ok := mm.messages[messageId]
	return ok
}

func (mm *MessageManager) forwardBiz(messageId BizMessageId) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.bizMessages[messageId] = time.Now()
}

func (mm *MessageManager) isForwardedBiz(messageId BizMessageId) bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	_, ok := mm.bizMessages[messageId]
	return ok
}

func (mm *MessageManager) ByteToBizId(bid []byte) BizMessageId {
	var id [BizMessageIdLength]byte
	for i := 0; i < len(bid) && i < BizMessageIdLength; i++ {
		id[i] = bid[i]
	}
	return id
}

func (mm *MessageManager) clear() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	now := time.Now()
	MessageCacheTime := 5 * time.Minute

	for mid, t := range mm.messages {
		if now.Sub(t) > MessageCacheTime {
			delete(mm.messages, mid)
		}
	}

	for mid, t := range mm.bizMessages {
		if now.Sub(t) > MessageCacheTime {
			delete(mm.bizMessages, mid)
		}
	}

	return
}
