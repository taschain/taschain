package rpc

import (
	"common"
	"container/list"
	"github.com/gorilla/websocket"
	"net/http"
	"storage/core/types"
	"sync"
)

const (
	EventTypeLog = 0
	EventTypeTransaction = 1
)

type EventSubscribeReq struct {
	Type int
	ContractAddress string
	EventName string
	Argument string
}

type EventSubscribe struct {
	contractAddress common.Address
	eventName       common.Hash
	argument        common.Hash
	socket          *websocket.Conn
	lock            sync.Mutex
}

type WsEventPublisher struct {
	subscribeMap map[common.Address]*list.List
	lock *sync.RWMutex
}

var EventPublisher = WsEventPublisher{subscribeMap:make(map[common.Address]*list.List), lock:&sync.RWMutex{}}

func (wsep *WsEventPublisher) addEventSubscribe(subscribe *EventSubscribe)  {
	wsep.lock.Lock()
	defer wsep.lock.Unlock()
	if l,ok := wsep.subscribeMap[subscribe.contractAddress];ok{
		l.PushBack(subscribe)
	} else {
		l := new(list.List)
		l.PushBack(subscribe)
		wsep.subscribeMap[subscribe.contractAddress] = l
	}
}

func (wsep *WsEventPublisher) PublishEvent(log *types.Log){
	wsep.lock.RLock()
	defer wsep.lock.RUnlock()
	if l,ok := wsep.subscribeMap[log.Address];ok{
		e := l.Front()
		for e != nil{
			item := e.Value.(*EventSubscribe)
			if log.Topics[0] == item.eventName{
				//logArgs := log.Topics[1:]
				//lengthLog := len(logArgs)
				//lengthSub := len(item.argument)
				//length := lengthLog
				//if lengthSub < length{
				//	length = lengthSub
				//}
				//match := true
				//for i:=0;i<length;i++{
				//	if !common.EmptyHash(item.argument[i]) && item.argument[i] != logArgs[i]{
				//		match = false
				//		break
				//	}
				//}
				match := common.EmptyHash(item.argument) || item.argument == log.Topics[1]
				if match{
					go wsep.publishEvent(log, item, e)
				}
			}
		}
	}
}

func (wsep *WsEventPublisher) publishEvent(log *types.Log, subscribe *EventSubscribe, e *list.Element){
	subscribe.lock.Lock()
	defer subscribe.lock.Unlock()
	err := subscribe.socket.WriteJSON(log)
	logger.Debugf("publishEvent %+v %+v", subscribe, log)
	if err != nil{
		logger.Error(err)
		wsep.lock.Lock()
		defer wsep.lock.Unlock()
		subscribe.socket.Close()
		wsep.subscribeMap[log.Address].Remove(e)
	}
}

var upgrader  = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func serveEventRequest(w http.ResponseWriter,r *http.Request){
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			logger.Error(err)
		}
		return
	}
	var req EventSubscribeReq
	err = ws.ReadJSON(&req)
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debugf("EventSubscribeReq %+v",req)
	//args := make([]common.Hash,len(req.Arguments))
	//for i,arg := range req.Arguments{
	//	if arg == "*"{
	//		args[i] = common.Hash{}
	//	} else {
	//		args[i] = common.BytesToHash(common.Sha256([]byte(arg)))
	//	}
	//}
	subscribe := &EventSubscribe{contractAddress:common.HexStringToAddress(req.ContractAddress), argument:common.BytesToHash(common.Sha256([]byte(req.Argument))),
		eventName:common.BytesToHash(common.Sha256([]byte(req.EventName))), socket:ws}
	EventPublisher.addEventSubscribe(subscribe)
}