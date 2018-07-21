package network

import (
	"time"
	"hash/fnv"
	"sync"
)


//MessageManager 消息管理
type MessageManager struct {
	messages map[uint64]time.Time
	index uint32
	id 	NodeID
	forwardNodeId uint32
	mutex sync.Mutex
}


func newMessageManager(id NodeID) *MessageManager {

	mm := &MessageManager{
		messages: make( map[uint64]time.Time),
	}
	mm.id = id
	mm.index = 0
	h := fnv.New32a()
	h.Write(id[:])
	mm.forwardNodeId = uint32(h.Sum32())
	return mm
}

func (mm *MessageManager) genMessageId() uint64 {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.index +=1
	messageId := uint64(mm.forwardNodeId)
	messageId = messageId << 32
	messageId= messageId | uint64(mm.index)
	mm.messages[messageId] = time.Now()
	return  messageId
}

func (mm *MessageManager) forward(messageId uint64)  {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.messages[messageId] = time.Now()
}

func (mm *MessageManager) isForwarded(messageId uint64) bool  {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	_,ok := mm.messages[messageId]
	return ok
}
