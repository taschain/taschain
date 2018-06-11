package logical

import (
	"core"
	"log"
	"common"
	"fmt"
	"time"
	"consensus/groupsig"
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
	p.futureBlockLock.Lock()
	defer p.futureBlockLock.Unlock()

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
	p.futureBlockLock.RLock()
	defer p.futureBlockLock.RUnlock()

	if t, ok := p.futureBlockMsg[hash]; ok {
		ret := make([]*ConsensusBlockMessage, len(t))
		copy(ret, t)
		return t
	} else {
		return nil
	}
}

func (p *Processer) removeFutureBlockMsgs(hash common.Hash) {
	p.futureBlockLock.Lock()
	defer p.futureBlockLock.Unlock()
	delete(p.futureBlockMsg, hash)
}

func (p *Processer) doAddOnChain(block *core.Block) (result int8) {
	begin := time.Now()
	defer func() {
		log.Printf("doAddOnChain begin at %v, cost %v\n", begin.String(), time.Since(begin).String())
	}()
	result = p.MainChain.AddBlockOnChain(block)

	bh := block.Header

	log.Printf("AddBlockOnChain header %v \n", p.blockPreview(bh))
	log.Printf("QueryTopBlock header %v \n", p.blockPreview(p.MainChain.QueryTopBlock()))
	log.Printf("proc(%v) core.AddBlockOnChain, height=%v, qn=%v, result=%v.\n", p.getPrefix(), bh.Height, bh.QueueNumber, result)
	logHalfway("doAddOnChain", bh.Height, bh.QueueNumber, p.getPrefix(), "result=%v,castor=%v", result, GetIDPrefix(*groupsig.NewIdFromBytes(bh.Castor)))

	if result == 0 {
		p.triggerFutureVerifyMsg(block.Header.Hash)
	} else if result == -1 {
		p.removeFutureVerifyMsgs(block.Header.Hash)
	}

	return result

}

func (p *Processer) blockOnChain(bh *core.BlockHeader) bool {
	exist := p.MainChain.QueryBlockByHash(bh.Hash)
	if exist != nil {	//已经上链
		return true
	}
	return false
}

func (p *Processer) getBlockHeaderByHash(hash common.Hash) *core.BlockHeader {
    b := p.MainChain.QueryBlockByHash(hash)
	if b == nil {
		//p.futureBlockLock.RLock()
		//defer p.futureBlockLock.RUnlock()
		//END:
		//for _, bm := range p.futureBlockMsg {
		//	for _, msg := range bm {
		//		if msg.Block.Header.Hash == hash {
		//			b = msg.Block.Header
		//			log.Printf("getBlockHeaderByHash: got from future blockMsg! hash=%v, height=%v\n", b.Hash, b.Height)
		//			break END
		//		}
		//	}
		//}
	}
	return b
}

func findVerifyMsg(bs []*ConsensusBlockMessageBase, hash common.Hash) bool {
	for _, b := range bs {
		if b.BH.Hash == hash {
			return true
		}
	}
	return false
}

func (p *Processer) addFutureVerifyMsg(msg *ConsensusBlockMessageBase) {
	b := msg.BH
	log.Printf("future verifyMsg receive cached! h=%v, hash=%v, preHash=%v\n", b.Height, b.Hash, b.PreHash)
	p.futureVerifyLock.Lock()
	defer p.futureVerifyLock.Unlock()

	if bs, ok := p.futureVerifyMsg[b.PreHash]; ok {
		if !findVerifyMsg(bs, b.Hash) {
			bs = append(bs, msg)
			p.futureVerifyMsg[b.PreHash] = bs
		}
	} else {
		bs := make([]*ConsensusBlockMessageBase, 0)
		bs = append(bs, msg)
		p.futureVerifyMsg[b.PreHash] = bs
	}
}

func (p *Processer) getFutureVerifyMsgs(hash common.Hash) []*ConsensusBlockMessageBase {
	p.futureVerifyLock.RLock()
	defer p.futureVerifyLock.RUnlock()

	if t, ok := p.futureVerifyMsg[hash]; ok {
		ret := make([]*ConsensusBlockMessageBase, len(t))
		copy(ret, t)
		return t
	} else {
		return nil
	}
}

func (p *Processer) removeFutureVerifyMsgs(hash common.Hash) {
	p.futureVerifyLock.Lock()
	defer p.futureVerifyLock.Unlock()
	delete(p.futureVerifyMsg, hash)
}

func (p *Processer) blockPreview(bh *core.BlockHeader) string {
    return fmt.Sprintf("hash=%v, height=%v, qn=%v, curTime=%v, preHash=%v, preTime=%v", GetHashPrefix(bh.Hash), bh.Height, bh.QueueNumber, bh.CurTime, GetHashPrefix(bh.PreHash), bh.PreTime)
}