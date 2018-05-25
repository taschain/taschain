package logical

import (
	"core"
	"log"
	"sync"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/5/16 下午7:44
**  Description: 
*/

var futureBlocks = make(map[common.Hash]*core.Block)
var lock = sync.Mutex{}

func addFutureBlock(b *core.Block) {
	log.Printf("future block receive cached! h=%v, hash=%v\n", b.Header.Height, b.Header.Hash)
	lock.Lock()
	defer lock.Unlock()
	futureBlocks[b.Header.PreHash] = b
}

func (p *Processer) doAddOnChain(block *core.Block) (result int8) {
	result = p.MainChain.AddBlockOnChain(block)
	log.Printf("AddBlockOnChain header %v \n", block.Header)
	log.Printf("QueryTopBlock header %v \n", p.MainChain.QueryTopBlock())
	log.Printf("proc(%v) core.AddBlockOnChain, height=%v, qn=%v, result=%v.\n", p.getPrefix(), block.Header.Height, block.Header.QueueNumber, result)
	return result

}

func (p *Processer) AddOnChain(block *core.Block) (result int8, futrue bool) {
	pre := p.MainChain.QueryBlockByHash(block.Header.PreHash)
	if pre == nil {
		addFutureBlock(block)
		return int8(0), true
	}

	topHash := p.MainChain.QueryTopBlock().Hash

	 result = p.doAddOnChain(block)
	 preHash := p.MainChain.QueryTopBlock().Hash

	 lock.Lock()
	 defer lock.Unlock()

	 del := make([]common.Hash, 0)
	 for f, ok := futureBlocks[preHash]; ok; {
		 r := p.doAddOnChain(f)
		 log.Printf("add cached block on chain, height = %v, hash = %v, result=%v\n", f.Header.Height, f.Header.Hash, r)
		 del = append(del, preHash)
		 preHash = p.MainChain.QueryTopBlock().Hash
	 }

	for _, d := range del {
		delete(futureBlocks, d)
	}

	afterOnChainTopHash := p.MainChain.QueryTopBlock().Hash
	if topHash != afterOnChainTopHash {	//链最高块发生变化后, 触发下个铸块者检查
		p.triggerCastCheck()
	}
	return result, false
}