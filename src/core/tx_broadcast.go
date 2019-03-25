package core

import (
	"middleware/types"
	"time"
	"sync"
	"middleware/ticker"
	"github.com/hashicorp/golang-lru"
	"common"
)

/*
**  Creator: pxf
**  Date: 2019/3/20 下午5:02
**  Description: 
*/

const (
	broadcastTimerInterval = 5
	tickerCheckBroadcast = "check_broadcast"
	broadcastInterval = 60*time.Second
	maxBroadcastPerTime = 50
)

type txBroadcastAgent struct {
	recentBroadcast *lru.Cache
	pool  *TxPool
	ticker 	*ticker.GlobalTicker
	lock sync.RWMutex
}
var agent *txBroadcastAgent

func initBroadcastAget(pool *TxPool) {
	cache, _ := lru.New(1000)
	agent = &txBroadcastAgent{
		recentBroadcast: cache,
		pool: pool,
		ticker: ticker.NewGlobalTicker("tx_broadcast_agent"),
	}
	agent.ticker.RegisterPeriodicRoutine(tickerCheckBroadcast, agent.broadcast, broadcastTimerInterval)
	agent.ticker.StartTickerRoutine(tickerCheckBroadcast, false)
}


func (ag *txBroadcastAgent) clearRecentBroadcast() {
	for _, k := range ag.recentBroadcast.Keys() {
		t, ok := ag.recentBroadcast.Get(k)
		if ok {
			if time.Since(t.(time.Time)).Seconds() > float64(broadcastInterval) {
				ag.recentBroadcast.Remove(k)
			}
		}
	}
}

func (ag *txBroadcastAgent) checkTxCanBroadcast(txHash common.Hash) bool {
	if t, ok := ag.recentBroadcast.Get(txHash); !ok || time.Since(t.(time.Time)).Seconds() > float64(broadcastInterval) {
		return true
	}
	return false
}

func (ag *txBroadcastAgent) broadcast() bool {
	ag.clearRecentBroadcast()

	txs := make([]*types.Transaction, 0)
	ag.pool.bonPool.forEach(func(tx *types.Transaction) bool {
		if ag.checkTxCanBroadcast(tx.Hash) {
			txs = append(txs, tx)
			return len(txs) < maxBroadcastPerTime
		}
		return true
	})
	if len(txs) < maxBroadcastPerTime {
		for _, tx := range ag.pool.received.asSlice(rcvTxPoolSize) {
			if ag.checkTxCanBroadcast(tx.Hash) {
				txs = append(txs, tx)
				if len(txs) >= maxBroadcastPerTime {
					break
				}
			}
		}
	}

	broadcastTransactions(txs)

	for _, tx := range txs {
		ag.recentBroadcast.Add(tx.Hash, time.Now())
	}

	return true
}