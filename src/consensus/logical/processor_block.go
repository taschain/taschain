package logical

import (
	"core"
	"log"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/5/16 下午7:44
**  Description: 
*/

func findBlock(bs []*ConsensusBlockMessage, hash common.Hash) bool {
	for _, b := range bs {
		if b.Block.Header.Hash == hash {
			return true
		}
	}
	return false
}

func (p *Processer) addFutureBlockMsg(msg *ConsensusBlockMessage) {
	b := msg.Block
	log.Printf("future block receive cached! h=%v, hash=%v\n", b.Header.Height, b.Header.Hash)
	p.futureLock.Lock()
	defer p.futureLock.Unlock()

	if bs, ok := p.futureBlockMsg[b.Header.PreHash]; ok {
		if !findBlock(bs, b.Header.Hash) {
			bs = append(bs, msg)
			p.futureBlockMsg[b.Header.PreHash] = bs
		}
	} else {
		bs := make([]*ConsensusBlockMessage, 0)
		bs = append(bs, msg)
		p.futureBlockMsg[b.Header.PreHash] = bs
	}
}

func (p *Processer) getFutureBlockMsgs(hash common.Hash) []*ConsensusBlockMessage {
	p.futureLock.Lock()
	defer p.futureLock.Unlock()
	return p.futureBlockMsg[hash]
}

func (p *Processer) removeFutureBlockMsgs(hash common.Hash) {
	p.futureLock.Lock()
	defer p.futureLock.Unlock()
	delete(p.futureBlockMsg, hash)
}

func (p *Processer) doAddOnChain(block *core.Block) (result int8) {
	result = p.MainChain.AddBlockOnChain(block)
	log.Printf("AddBlockOnChain header %v \n", block.Header)
	log.Printf("QueryTopBlock header %v \n", p.MainChain.QueryTopBlock())
	log.Printf("proc(%v) core.AddBlockOnChain, height=%v, qn=%v, result=%v.\n", p.getPrefix(), block.Header.Height, block.Header.QueueNumber, result)
	return result

}
