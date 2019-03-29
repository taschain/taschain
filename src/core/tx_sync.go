package core

import (
	"middleware/types"
	"time"
	"sync"
	"middleware/ticker"
	"github.com/hashicorp/golang-lru"
	"common"
	"network"
	"middleware/notify"
	"bytes"
	"utility"
	"taslog"
)

/*
**  Creator: pxf
**  Date: 2019/3/20 下午5:02
**  Description: 
*/

const (
	txNofifyInterval   = 5
	txNotifyRoutine    = "ts_notify"
	txNotifyGap        = 60*time.Second
	txMaxNotifyPerTime = 100

	txReqRoutine 		= "ts_req"
	txReqInterval		= 5
)

type txSyncer struct {
	pool 		*TxPool
	chain 		*FullBlockChain
	rctNotifiy    *lru.Cache
	txSimpleIndex *lru.Cache
	ticker        *ticker.GlobalTicker
	candidateKeys *lru.Cache
	logger 		taslog.Logger

}
var TxSyncer *txSyncer

type peerTxsKeys struct {
	lock sync.RWMutex
	txKeys map[uint64]byte
}

func newPeerTxsKeys() *peerTxsKeys {
	return &peerTxsKeys{
		txKeys: make(map[uint64]byte),
	}
}

func (ptk *peerTxsKeys) addKeys(ks []uint64)  {
	ptk.lock.Lock()
	defer ptk.lock.Unlock()
	for _, k := range ks {
		ptk.txKeys[k] = 1
	}
}

func (ptk *peerTxsKeys) removeKeys(ks []uint64)  {
	ptk.lock.Lock()
	defer ptk.lock.Unlock()
	for _, k := range ks {
		delete(ptk.txKeys, k)
	}
}

func (ptk *peerTxsKeys) reset()  {
	ptk.lock.Lock()
	defer ptk.lock.Unlock()
	ptk.txKeys = make(map[uint64]byte)
}

func (ptk *peerTxsKeys) hasKey(k uint64) bool {
	ptk.lock.RLock()
	defer ptk.lock.RUnlock()
	_, ok := ptk.txKeys[k]
	return ok
}

func (ptk *peerTxsKeys) forEach(f func(k uint64) bool)  {
	ptk.lock.RLock()
	defer ptk.lock.RUnlock()
	for k, _ := range ptk.txKeys {
		if !f(k) {
			break
		}
	}
}

func initTxSyncer(pool *TxPool) {
	cache, _ := lru.New(1000)
	indexs, _ := lru.New(maxTxPoolSize)
	cands, _ := lru.New(100)
	s := &txSyncer{
		rctNotifiy:    cache,
		txSimpleIndex: indexs,
		pool:pool,
		ticker:        ticker.NewGlobalTicker("tx_syncer"),
		candidateKeys: cands,
		logger: taslog.GetLoggerByIndex(taslog.TxSyncLogConfig, common.GlobalConf.GetString("instance", "index", "")),
	}
	s.ticker.RegisterPeriodicRoutine(txNotifyRoutine, s.notifyTxs, txNofifyInterval)
	s.ticker.StartTickerRoutine(txNotifyRoutine, false)

	s.ticker.RegisterPeriodicRoutine(txReqRoutine, s.reqTxsRoutine, txReqInterval)
	s.ticker.StartTickerRoutine(txReqRoutine, false)

	notify.BUS.Subscribe(notify.TxSyncNotify, s.onTxNotify)
	notify.BUS.Subscribe(notify.TxSyncReq, s.onTxReq)
	notify.BUS.Subscribe(notify.TxSyncResponse, s.onTxResponse)
	TxSyncer = s
}

func simpleTxKey(hash common.Hash) uint64 {
	return hash.Big().Uint64()
}

func (ts *txSyncer) add(tx *types.Transaction)  {
    ts.txSimpleIndex.Add(simpleTxKey(tx.Hash), tx.Hash)
}

func (ts *txSyncer) get(k uint64) *types.Transaction {
    if v, ok := ts.txSimpleIndex.Get(k); ok {
    	txHash := v.(common.Hash)
		return BlockChainImpl.GetTransactionByHash(false, false, txHash)
	}
	return nil
}

func (ts *txSyncer) has(key uint64) bool {
	if _, ok := ts.txSimpleIndex.Get(key); ok {
		return ok
	}
	return false
}

func (ts *txSyncer) clearJob() {
	for _, k := range ts.rctNotifiy.Keys() {
		t, ok := ts.rctNotifiy.Get(k)
		if ok {
			if time.Since(t.(time.Time)).Seconds() > float64(txNotifyGap) {
				ts.rctNotifiy.Remove(k)
			}
		}
	}
	ts.pool.bonPool.forEach(func(tx *types.Transaction) bool {
		if ts.pool.bonPool.hasBonus(tx.Data) {
			bhash := common.BytesToHash(tx.Data)
			rm := ts.pool.bonPool.removeByBlockHash(bhash)
			ts.logger.Debugf("remove from bonus pool: blockHash %v, size %v", bhash.String(), rm)
		}
		return true
	})
}

func (ts *txSyncer) checkTxCanBroadcast(txHash common.Hash) bool {
	if t, ok := ts.rctNotifiy.Get(txHash); !ok || time.Since(t.(time.Time)).Seconds() > float64(txNotifyGap) {
		return true
	}
	return false
}

func (ts *txSyncer) notifyTxs() bool {
	ts.clearJob()

	txs := make([]*types.Transaction, 0)
	ts.pool.bonPool.forEach(func(tx *types.Transaction) bool {
		if ts.checkTxCanBroadcast(tx.Hash) {
			txs = append(txs, tx)
			return len(txs) < txMaxNotifyPerTime
		}
		return true
	})
	if len(txs) < txMaxNotifyPerTime {
		for _, tx := range ts.pool.received.asSlice(maxTxPoolSize) {
			if ts.checkTxCanBroadcast(tx.Hash) {
				txs = append(txs, tx)
				if len(txs) >= txMaxNotifyPerTime {
					break
				}
			}
		}
	}

	ts.sendSimpleTxKeys(txs)

	for _, tx := range txs {
		ts.rctNotifiy.Add(tx.Hash, time.Now())
	}

	return true
}

func (ts *txSyncer) sendSimpleTxKeys(txs []*types.Transaction) {
	if len(txs) > 0 {
		txKeys := make([]uint64, 0)

		for _, tx := range txs {
			txKeys = append(txKeys, simpleTxKey(tx.Hash))
		}

		bodyBuf := bytes.NewBuffer([]byte{})
		for _, k := range txKeys {
			bodyBuf.Write(utility.UInt64ToByte(k))
		}

		ts.logger.Debugf("notify transactions len:%d", len(txs))
		message := network.Message{Code: network.TxSyncNotify, Body: bodyBuf.Bytes()}

		netInstance := network.GetNetInstance()
		if netInstance != nil {
			go network.GetNetInstance().TransmitToNeighbor(message)
		}
	}
}

func (ts *txSyncer) getOrAddCandidateKeys(id string) *peerTxsKeys {
    v, _ := ts.candidateKeys.Get(id)
	if v == nil {
		v = newPeerTxsKeys()
		ts.candidateKeys.Add(id, v)
	}
	return v.(*peerTxsKeys)
}

func (ts *txSyncer) onTxNotify(msg notify.Message)  {
	nm := msg.(*notify.NotifyMessage)
	reader := bytes.NewReader(nm.Body)

	keys := make([]uint64, 0)
	buf := make([]byte, 8)
	for {
		n, _ := reader.Read(buf)
		if n != 8 {
			break
		}
		keys = append(keys, utility.ByteToUInt64(buf))
	}

	candidateKeys := ts.getOrAddCandidateKeys(nm.Source)

	accepts := make([]uint64, 0)

	for _, k := range keys {
		if !ts.has(k) {
			accepts = append(accepts, k)
		}
	}
	candidateKeys.addKeys(accepts)
	ts.logger.Debugf("Rcv txs notify from %v, size %v, accept %v, totalOfSource %v", nm.Source, len(keys), len(accepts), len(candidateKeys.txKeys))

}

func (ts *txSyncer) reqTxsRoutine() bool {
	ts.logger.Debugf("req txs routine, candidate size %v", ts.candidateKeys.Len())
	reqMap := make(map[uint64]byte)
	//去重
	for _, v := range ts.candidateKeys.Keys() {
		ptk := ts.getOrAddCandidateKeys(v.(string))
		if ptk == nil {
			continue
		}
		rms := make([]uint64, 0)
		ptk.forEach(func(k uint64) bool {
			if _, exist := reqMap[k]; exist {
				rms = append(rms, k)
			} else {
				reqMap[k] = 1
			}
			return true
		})
		ptk.removeKeys(rms)
	}
	//请求
	for _, v := range ts.candidateKeys.Keys() {
		ptk := ts.getOrAddCandidateKeys(v.(string))
		if ptk == nil {
			continue
		}
		rqs := make([]uint64, 0)
		ptk.forEach(func(k uint64) bool {
			if !ts.has(k) {
				rqs = append(rqs, k)
			}
			return true
		})
		ptk.reset()
		if len(rqs) > 0 {
			go ts.requestTxs(v.(string), rqs)
		}
	}
	return true
}

func (ts *txSyncer) requestTxs(id string, keys []uint64)  {
	ts.logger.Debugf("request txs from %v, size %v", id, len(keys))

	bodyBuf := bytes.NewBuffer([]byte{})
	for _, k := range keys {
		bodyBuf.Write(utility.UInt64ToByte(k))
	}

	message := network.Message{Code: network.TxSyncReq, Body: bodyBuf.Bytes()}

	network.GetNetInstance().Send(id, message)
}

func (ts *txSyncer) onTxReq(msg notify.Message)  {
	nm := msg.(*notify.NotifyMessage)
	reader := bytes.NewReader(nm.Body)
	keys := make([]uint64, 0)
	buf := make([]byte, 8)
	for {
		n, _ := reader.Read(buf)
		if n != 8 {
			break
		}
		keys = append(keys, utility.ByteToUInt64(buf))
	}

	ts.logger.Debugf("Rcv tx req from %v, size %v", nm.Source, len(keys))

	txs := make([]*types.Transaction, 0)
	for _, k := range keys {
		tx := ts.get(k)
		if tx != nil {
			txs = append(txs, tx)
		}
	}
	if len(txs) == 0 {
		return
	}
	body, e := types.MarshalTransactions(txs)
	if e != nil {
		ts.logger.Errorf("Discard MarshalTransactions because of marshal error:%s!", e.Error())
		return
	}
	ts.logger.Debugf("send transactions to %v size %v", len(txs), nm.Source)
	message := network.Message{Code: network.TxSyncResponse, Body: body}
	network.GetNetInstance().Send(nm.Source, message)
}

func (ts *txSyncer) onTxResponse(msg notify.Message)  {
	nm := msg.(*notify.NotifyMessage)
	txs, e := types.UnMarshalTransactions(nm.Body)
	if e != nil {
		ts.logger.Errorf("Unmarshal got transactions error:%s", e.Error())
		return
	}

	ts.logger.Debugf("Rcv txs from %v, size %v", nm.Source, len(txs))
	ts.pool.AddTransactions(txs, txSync)
}