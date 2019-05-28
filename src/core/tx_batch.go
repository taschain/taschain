package core

import (
	"sync"
	"runtime"
	"middleware/types"
	"math"
)

/*
**  Creator: pxf
**  Date: 2019/5/22 上午11:37
**  Description: 
*/

type txBatchAdder struct {
	pool TransactionPool
	routineNum int
	mu sync.Mutex
}

func newTxBatchAdder(pool TransactionPool) *txBatchAdder {
	return &txBatchAdder{
		pool: 		pool,
		routineNum: runtime.NumCPU(),
	}
}

func (tv *txBatchAdder) batchAdd(txs []*types.Transaction) {
    if len(txs) == 0 {
    	return
	}
	tv.mu.Lock()
	defer tv.mu.Unlock()
	wg := sync.WaitGroup{}
	step := int(math.Ceil(float64(len(txs))/float64(tv.routineNum)))
	for begin := 0; begin < len(txs); {
		end := begin + step
		if end > len(txs) {
			end = len(txs)
		}
		copySlice := txs[begin:end]
		wg.Add(1)
		go func() {
			defer wg.Done()
			tv.pool.AsyncAddTxs(copySlice)
		}()
		begin = end
	}
	wg.Wait()
}
