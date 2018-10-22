package core

import (
	"storage/tasdb"
	"common"
	"math/big"
	"github.com/vmihailenco/msgpack"
	"sync"
	"time"
	"network"
	"core/datasource"
)

const timerDuration  = time.Second * 2

var TraceChainImpl TraceChain

type TraceChain struct {
	tracedb			tasdb.Database
	//topdb			tasdb.Database
	missdb          tasdb.Database
	lock            sync.RWMutex
	timer           *time.Timer
	TraceResponseCh chan *TraceHeader
}

type TraceHeader struct {
	Hash 	common.Hash
	PreHash common.Hash
	Value   *big.Int
	TotalQn uint64
	Height  uint64
}

func initTraceChain() error {
	trace, err := datasource.NewDatabase("trace")
	if err != nil {
		return err
	}
	miss, err := datasource.NewDatabase("tracemiss")
	if err != nil {
		return err
	}
	TraceChainImpl = TraceChain{tracedb:trace,missdb:miss,timer:time.NewTimer(timerDuration), TraceResponseCh:make(chan *TraceHeader)}
	go TraceChainImpl.loop()
	return nil
}

func (tc *TraceChain) AddTrace(header *TraceHeader) error{
	if exist,_ := tc.tracedb.Has(header.Hash.Bytes());exist{
		return ErrExisted
	}
	tc.lock.Lock()
	defer tc.lock.Unlock()
	return tc.addTrace(header)
}

func (tc *TraceChain) addTrace(header *TraceHeader) error {
	data,err := msgpack.Marshal(header)
	if err != nil{
		return err
	}

	err = tc.tracedb.Put(header.Hash.Bytes(), data)
	if err != nil{
		return err
	}

	if exist,_ := tc.tracedb.Has(header.PreHash.Bytes());exist{
		tc.missdb.Put(header.PreHash.Bytes(),[]byte{})
	}

	return nil
}

func (tc *TraceChain) GetTraceHeaderRawByHash(hash []byte) []byte{
	data,_ := tc.tracedb.Get(hash)
	return data
}

func (tc *TraceChain) GetTraceHeaderByHash(hash []byte) *TraceHeader{
	data,_ := tc.tracedb.Get(hash)
	if data == nil{
		return nil
	}
	var header TraceHeader
	if err := msgpack.Unmarshal(data,&header);err != nil{
		panic(err)
	}
	return &header
}

func (tc *TraceChain) FindCommonAncestor(localHash []byte, remoteHash []byte) (bool,*TraceHeader,error) {
	tc.lock.RLock()
	defer tc.lock.RUnlock()

	nextLocal := tc.GetTraceHeaderByHash(localHash)
	nextRemote := tc.GetTraceHeaderByHash(remoteHash)
	if nextLocal == nil || nextRemote == nil{
		panic("FindCommonAncestor not found last")
	}
	var flag = 0
	var local,remote *TraceHeader
	for{
		if nextLocal.PreHash == nextRemote.PreHash {
			replace := nextRemote.Value.Cmp(nextLocal.Value) > 0
			return replace, tc.GetTraceHeaderByHash(nextLocal.PreHash.Bytes()),nil
		}
		switch flag {
		case 0:
			local = tc.GetTraceHeaderByHash(nextLocal.PreHash.Bytes())
			remote = tc.GetTraceHeaderByHash(nextRemote.PreHash.Bytes())
		case 1:
			local = tc.GetTraceHeaderByHash(nextLocal.PreHash.Bytes())
		case 2:
			remote = tc.GetTraceHeaderByHash(nextRemote.PreHash.Bytes())
		}

		if local == nil || remote == nil{
			return false, nil, ErrMissingTrace
		}
		if local.Height > remote.Height {
			nextLocal = local
			flag = 1
		} else if local.Height < remote.Height {
			nextRemote = remote
			flag = 2
		} else {
			nextLocal = local
			nextRemote = remote
			flag = 0
		}
	}
}

func (tc *TraceChain) loop(){
	for  {
		select {
			case <-tc.timer.C:
			iter := tc.missdb.NewIterator()
			for iter.Next() {
				hash := iter.Key()
				message := network.Message{Code: network.RequestTraceMsg, Body:hash}
				network.GetNetInstance().TransmitToNeighbor(message)
			}
			tc.timer.Reset(timerDuration)
			case header := <- tc.TraceResponseCh:
				tc.AddTrace(header)
		}
	}
}