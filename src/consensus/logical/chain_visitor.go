package logical

import (
	"core"
	"fmt"
	"sync"
)

/*
**  Creator: pxf
**  Date: 2018/5/16 下午7:44
**  Description: 
*/

var futureBlocks = make(map[uint64][]*core.Block)
var lock = sync.Mutex{}

func addFutureBlock(b *core.Block) {
	fmt.Printf("future block receive cached! h=%v\n", b.Header.Height)
	lock.Lock()
	defer lock.Unlock()
	h := b.Header.Height
	if bs, ok := futureBlocks[h]; ok {
		bs = append(bs, b)
	} else {
		bs = make([]*core.Block, 0)
		bs = append(bs, b)
		futureBlocks[h] = bs
	}
}

func (p *Processer) doAddOnChain(block *core.Block) (result int8) {
	result = p.MainChain.AddBlockOnChain(block)
	fmt.Printf("AddBlockOnChain header %v \n", block.Header)
	fmt.Printf("QueryTopBlock header %v \n", p.MainChain.QueryTopBlock())
	fmt.Printf("proc(%v) core.AddBlockOnChain, height=%v, qn=%v, result=%v.\n", p.getPrefix(), block.Header.Height, block.Header.QueueNumber, result)
	return result

}

func (p *Processer) AddOnChain(block *core.Block) (result int8, futrue bool) {
	pre := p.MainChain.QueryBlockByHeight(block.Header.Height - 1)
	if pre == nil {
		addFutureBlock(block)
		return int8(0), true
	}
	 result = p.doAddOnChain(block)
	 currentHeight := p.MainChain.QueryTopBlock().Height

	 lock.Lock()
	 defer lock.Unlock()

	 del := make([]uint64, 0)
	for h, bs := range futureBlocks {
		if h == currentHeight+1 {
			fmt.Printf("add cached block on chain, h = %v, size = %v\n", h, len(bs))
			for _, b := range bs {
				p.doAddOnChain(b)
			}
			del = append(del, h)
		}
		currentHeight = p.MainChain.QueryTopBlock().Height
	}

	for _, d := range del {
		delete(futureBlocks, d)
	}
	return result, false
}