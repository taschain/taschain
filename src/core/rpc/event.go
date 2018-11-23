package rpc

import (
	"common"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/golang-lru"
	"gopkg.in/fatih/set.v0"
	"log"
	"net/http"
	"storage/core/types"
	"sync"
	t "middleware/types"
)

const (
	LogWatch = 0
	TransactionWatch = 1
	LogWatchResult = 2
	TransactionWatchResult = 3
	LogEvent = 4
	TransactionEvent = 5
)

type Hash2 [common.HashLength * 2]byte

func combineToHash2(low , high common.Hash) *Hash2 {
	var hash2 Hash2
	copy(hash2[:common.HashLength], low[:])
	copy(hash2[common.HashLength:], high[:])
	return &hash2
}

type EventSubscribeReq struct {
	Type int
	ContractAddress string
	EventName string
	Argument string
}

type EventSubscribe struct {
	eventName       common.Hash
	argument        common.Hash
}

type WsHolder struct {
	subscribeLogMap map[common.Address]set.Interface
	socket          *websocket.Conn
	lock            *sync.Mutex
	closed			bool
}

type PushResponse struct {
	Type int
	Code int
	Hash common.Hash
	Value interface{}
	Error *t.TransactionError
}

type WsEventPublisher struct {
	subscribeLogMap map[common.Address]set.Interface
	transactionPushCache *lru.Cache
	lock            *sync.RWMutex
}

func (wh *WsHolder) addEventSubscribe(contract common.Address, eventName common.Hash,argument common.Hash)  {
	wh.lock.Lock()
	defer wh.lock.Unlock()
	subscribe := EventSubscribe{eventName:eventName, argument:argument}
	if s,ok := wh.subscribeLogMap[contract];ok{
		s.Add(subscribe)
	} else{
		s = set.New(set.NonThreadSafe)
		s.Add(subscribe)
		wh.subscribeLogMap[contract] = s
		EventPublisher.addEventSubscribe(contract, wh)
	}
	log.Printf("addEventSubscribe contract:%s event:%s",contract.GetHexString(),eventName.Hex())
}

func (wh *WsHolder) close()  {
	wh.lock.Lock()
	defer wh.lock.Unlock()
	if !wh.closed {
		EventPublisher.leave(wh)
		wh.closed = true
		log.Printf("WsHolder close")
	}
}

var EventPublisher WsEventPublisher

func init()  {
	transactionPushCache,_ := lru.New(500)
	EventPublisher = WsEventPublisher{subscribeLogMap: make(map[common.Address]set.Interface), lock:&sync.RWMutex{}, transactionPushCache:transactionPushCache}
}

func (wsep *WsEventPublisher) addEventSubscribe(contract common.Address, wh *WsHolder)  {
	wsep.lock.Lock()
	defer wsep.lock.Unlock()
	if s,ok := wsep.subscribeLogMap[contract];ok{
		s.Add(wh)
	} else {
		s = set.New(set.ThreadSafe)
		s.Add(wh)
		wsep.subscribeLogMap[contract] = s
	}
}

func (wsep *WsEventPublisher) addTransactionSubscribe(transaction common.Hash, wh *WsHolder){
	wsep.transactionPushCache.Add(transaction, wh)
	log.Printf("WsEventPublisher addTransactionSubscribe %s", transaction.Hex())
}


func (wsep *WsEventPublisher) leave(wh *WsHolder){
	wsep.lock.RLock()
	defer wsep.lock.RUnlock()
	for key := range wh.subscribeLogMap{
		if s,ok := wsep.subscribeLogMap[key];ok{
			s.Remove(wh)
		}
	}
}

func (wsep *WsEventPublisher) PublishTransaction(receipt *types.Receipt, err *t.TransactionError){
	log.Printf("WsEventPublisher PublishTransaction %v %v", receipt, err)
	if c,ok := wsep.transactionPushCache.Get(receipt.TxHash);ok{
		holder := c.(*WsHolder)
		resp := &PushResponse{Value:receipt, Type:TransactionEvent, Error:err}
		holder.push(resp)
		wsep.transactionPushCache.Remove(receipt.TxHash)
	}
	if receipt.Logs != nil {
		for _, log := range receipt.Logs {
			wsep.PublishEvent(log)
		}
	}
}

func (wsep *WsEventPublisher) PublishEvent(tlog *types.Log){
	log.Printf("WsEventPublisher PublishEvent %s %v", tlog.Address.GetHexString(),tlog.Topics)
	wsep.lock.RLock()
	defer wsep.lock.RUnlock()

	if s,ok := wsep.subscribeLogMap[tlog.Address];ok{
		s.Each(func(i interface{}) bool {
			holder := i.(*WsHolder)
			log.Printf("PublishEvent WsHolder %+v",holder)
			if holder.closed{
				return true
			}
			if ss,ok2 := holder.subscribeLogMap[tlog.Address];ok2{
				ss.Each(func(j interface{}) bool {
					subscribe := j.(EventSubscribe)
					log.Printf("PublishEvent EventSubscribe %+v",subscribe)
					if tlog.Topics[0] == subscribe.eventName{
						if common.EmptyHash(subscribe.argument) || subscribe.argument == tlog.Topics[1]{
							resp := &PushResponse{Value:tlog,Type:LogEvent}
							holder.push(resp)
						}
					}
					return true
				})
			}
			return true
		})

	}
}

func (wh *WsHolder) push(resp *PushResponse){
	wh.lock.Lock()
	defer wh.lock.Unlock()
	err := wh.socket.WriteJSON(resp)
	log.Printf("push %+v", resp)
	if err != nil{
		go wh.close()
	}
}

var upgrader  = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func loop(holder *WsHolder)  {
	for{
		if holder.closed{
			return
		}
		var req EventSubscribeReq
		err := holder.socket.ReadJSON(&req)
		if err != nil {
			log.Printf("WsEventPublisher ReadJSON %v",err)
			go holder.close()
			return
		}
		log.Printf("WsEventPublisher receive %+v", req)
		switch req.Type{
		case LogWatch:
			var arg common.Hash
			if req.Argument != ""{
				arg = common.BytesToHash(common.Sha256([]byte(req.Argument)))
			}
			eventName := common.BytesToHash(common.Sha256([]byte(req.EventName)))
			address := common.HexStringToAddress(req.ContractAddress)
			holder.addEventSubscribe(address, eventName, arg)
		case TransactionWatch:
			var arg common.Hash
			if req.Argument != ""{
				arg = common.HexToHash(req.Argument)
			}
			if !common.EmptyHash(arg){
				EventPublisher.addTransactionSubscribe(arg, holder)
			}
		}
	}
}

func serveEventRequest(w http.ResponseWriter,r *http.Request){
	ws, err := upgrader.Upgrade(w, r, nil)
	log.Printf("WsEventPublisher serveEventRequest connect")
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Printf("WsEventPublisher HandshakeError %v",err)
		}
		return
	}
	holder := &WsHolder{socket:ws,subscribeLogMap:make(map[common.Address]set.Interface),lock:&sync.Mutex{}}
	go loop(holder)
}