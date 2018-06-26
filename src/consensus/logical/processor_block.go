package logical

import (
	"log"
	"common"
	"fmt"
	"time"
	"consensus/groupsig"
	"middleware/types"
	"core"
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

func (p *Processor) addFutureBlockMsg(msg *ConsensusBlockMessage) {
	b := msg.Block
	log.Printf("future block receive cached! h=%v, hash=%v\n", b.Header.Height, GetHashPrefix(b.Header.Hash))
	p.futureBlockLock.Lock()
	defer p.futureBlockLock.Unlock()

	if bs, ok := p.futureBlockMsg[b.Header.PreHash]; ok {
		bs = append(bs, msg)
		p.futureBlockMsg[b.Header.PreHash] = bs
	} else {
		bs := make([]*ConsensusBlockMessage, 0)
		bs = append(bs, msg)
		p.futureBlockMsg[b.Header.PreHash] = bs
	}
}

func (p *Processor) getFutureBlockMsgs(hash common.Hash) []*ConsensusBlockMessage {
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

func (p *Processor) removeFutureBlockMsgs(hash common.Hash) {
	p.futureBlockLock.Lock()
	defer p.futureBlockLock.Unlock()
	delete(p.futureBlockMsg, hash)
}

func (p *Processor) doAddOnChain(block *types.Block) (result int8) {
	begin := time.Now()
	defer func() {
		log.Printf("doAddOnChain begin at %v, cost %v\n", begin.String(), time.Since(begin).String())
	}()
	result = p.MainChain.AddBlockOnChain(block)

	bh := block.Header

	log.Printf("AddBlockOnChain header %v \n", p.blockPreview(bh))
	log.Printf("QueryTopBlock header %v \n", p.blockPreview(p.MainChain.QueryTopBlock()))
	log.Printf("proc(%v) core.AddBlockOnChain, height=%v, qn=%v, result=%v.\n", p.getPrefix(), bh.Height, bh.QueueNumber, result)
	logHalfway("doAddOnChain", bh.Height, bh.QueueNumber, p.getPrefix(), "result=%v,castor=%v", result, GetIDPrefix(*groupsig.DeserializeId(bh.Castor)))

	if result == 0 {
		p.triggerFutureVerifyMsg(block.Header.Hash)
		p.groupManager.CreateNextGroupRoutine()
	} else if result == -1 {
		p.removeFutureVerifyMsgs(block.Header.Hash)
	}

	return result

}

func (p *Processor) blockOnChain(bh *types.BlockHeader) bool {
	exist := p.MainChain.QueryBlockByHash(bh.Hash)
	if exist != nil { //已经上链
		return true
	}
	return false
}

func (p *Processor) getBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
    b := p.MainChain.QueryBlockByHash(hash)
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

func (p *Processor) addFutureVerifyMsg(msg *ConsensusBlockMessageBase) {
	b := msg.BH
	log.Printf("future verifyMsg receive cached! h=%v, hash=%v, preHash=%v\n", b.Height, GetHashPrefix(b.Hash), GetHashPrefix(b.PreHash))
	p.futureVerifyLock.Lock()
	defer p.futureVerifyLock.Unlock()

	if bs, ok := p.futureVerifyMsg[b.PreHash]; ok {
		bs = append(bs, msg)
		p.futureVerifyMsg[b.PreHash] = bs
	} else {
		bs := make([]*ConsensusBlockMessageBase, 0)
		bs = append(bs, msg)
		p.futureVerifyMsg[b.PreHash] = bs
	}
}

func (p *Processor) getFutureVerifyMsgs(hash common.Hash) []*ConsensusBlockMessageBase {
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

func (p *Processor) removeFutureVerifyMsgs(hash common.Hash) {
	p.futureVerifyLock.Lock()
	defer p.futureVerifyLock.Unlock()
	delete(p.futureVerifyMsg, hash)
}

func (p *Processor) blockPreview(bh *types.BlockHeader) string {
    return fmt.Sprintf("hash=%v, height=%v, qn=%v, curTime=%v, preHash=%v, preTime=%v", GetHashPrefix(bh.Hash), bh.Height, bh.QueueNumber, bh.CurTime, GetHashPrefix(bh.PreHash), bh.PreTime)
}

func (p *Processor) prepareForCast(gid groupsig.ID)  {
	bc := new(BlockContext)
	bc.Proc = p
	bc.Init(GroupMinerID{gid, p.GetMinerID()})
	sgi, err := p.gg.GetGroupByID(gid)
	if err != nil {
		panic("prepareForCast GetGroupByID failed.\n")
	}
	bc.pos = sgi.GetMinerPos(p.GetMinerID())
	log.Printf("prepareForCast current ID in group pos=%v.\n", bc.pos)
	//to do:只有自己属于这个组的节点才需要调用AddBlockConext
	b := p.AddBlockContext(bc)
	log.Printf("(proc:%v) prepareForCast Add BlockContext result = %v, bc_size=%v.\n", p.getPrefix(), b, len(p.bcs))

	p.Ticker.RegisterRoutine(bc.getKingCheckRoutineName(), bc.kingTickerRoutine, uint32(MAX_USER_CAST_TIME))
	p.triggerCastCheck()
}

func (p *Processor) verifyBlock(bh *types.BlockHeader) ([]common.Hash, int8) {
	lostTransHash, ret, _, _ := core.BlockChainImpl.VerifyCastingBlock(*bh)
	log.Printf("BlockChainImpl.VerifyCastingBlock result=%v.", ret)
	return lostTransHash, ret
}