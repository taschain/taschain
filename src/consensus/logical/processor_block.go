package logical

import (
	"log"
	"common"
	"fmt"
	"time"
	"consensus/groupsig"
	"middleware/types"
	"core"
	"sync"
)

/*
**  Creator: pxf
**  Date: 2018/5/16 下午7:44
**  Description: 
*/
type FutureMessageHolder struct {
	messages sync.Map
}

func NewFutureMessageHolder() *FutureMessageHolder {
	return &FutureMessageHolder{
		messages: sync.Map{},
	}
}
func (holder *FutureMessageHolder) addMessage(hash common.Hash, msg interface{}) {
	if vs, ok := holder.messages.Load(hash); ok {
		vsSlice := vs.([]interface{})
		vsSlice = append(vsSlice, msg)
		holder.messages.Store(hash, vsSlice)
	} else {
		slice := make([]interface{}, 0)
		slice = append(slice, msg)
		holder.messages.Store(hash, slice)
	}
}

func (holder *FutureMessageHolder) getMessages(hash common.Hash) []interface{} {
    if vs, ok := holder.messages.Load(hash); ok {
    	return vs.([]interface{})
	}
	return nil
}

func (holder *FutureMessageHolder) remove(hash common.Hash) {
	holder.messages.Delete(hash)
}

func (p *Processor) addFutureBlockMsg(msg *ConsensusBlockMessage) {
	b := msg.Block
	log.Printf("future block receive cached! h=%v, hash=%v\n", b.Header.Height, GetHashPrefix(b.Header.Hash))

	p.futureBlockMsgs.addMessage(b.Header.PreHash, msg)
}

func (p *Processor) getFutureBlockMsgs(hash common.Hash) []*ConsensusBlockMessage {
	if vs := p.futureBlockMsgs.getMessages(hash); vs != nil {
		ret := make([]*ConsensusBlockMessage, len(vs))
		for idx, m := range vs {
			ret[idx] = m.(*ConsensusBlockMessage)
		}
		return ret
	}
	return nil
}

func (p *Processor) removeFutureBlockMsgs(hash common.Hash) {
	p.futureBlockMsgs.remove(hash)
}

func (p *Processor) doAddOnChain(block *types.Block) (result int8) {
	begin := time.Now()
	defer func() {
		log.Printf("doAddOnChain begin at %v, cost %v\n", begin.String(), time.Since(begin).String())
	}()
	result = p.MainChain.AddBlockOnChain(block)

	bh := block.Header

	//log.Printf("AddBlockOnChain header %v \n", p.blockPreview(bh))
	//log.Printf("QueryTopBlock header %v \n", p.blockPreview(p.MainChain.QueryTopBlock()))
	log.Printf("proc(%v) core.AddBlockOnChain, height=%v, qn=%v, result=%v.\n", p.getPrefix(), bh.Height, bh.QueueNumber, result)
	logHalfway("doAddOnChain", bh.Height, bh.QueueNumber, p.getPrefix(), "result=%v,castor=%v", result, GetIDPrefix(*groupsig.DeserializeId(bh.Castor)))

	if result == 0 {
		p.triggerFutureVerifyMsg(block.Header.Hash)
		p.groupManager.CreateNextGroupRoutine()
		p.cleanVerifyContext(bh.Height)
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

func (p *Processor) addFutureVerifyMsg(msg *ConsensusBlockMessageBase) {
	b := msg.BH
	log.Printf("future verifyMsg receive cached! h=%v, hash=%v, preHash=%v\n", b.Height, GetHashPrefix(b.Hash), GetHashPrefix(b.PreHash))

	p.futureVerifyMsgs.addMessage(b.PreHash, msg)
}

func (p *Processor) getFutureVerifyMsgs(hash common.Hash) []*ConsensusBlockMessageBase {
	if vs := p.futureVerifyMsgs.getMessages(hash); vs != nil {
		ret := make([]*ConsensusBlockMessageBase, len(vs))
		for idx, m := range vs {
			ret[idx] = m.(*ConsensusBlockMessageBase)
		}
		return ret
	}
	return nil
}

func (p *Processor) removeFutureVerifyMsgs(hash common.Hash) {
	p.futureVerifyMsgs.remove(hash)
}

func (p *Processor) blockPreview(bh *types.BlockHeader) string {
    return fmt.Sprintf("hash=%v, height=%v, qn=%v, curTime=%v, preHash=%v, preTime=%v", GetHashPrefix(bh.Hash), bh.Height, bh.QueueNumber, bh.CurTime, GetHashPrefix(bh.PreHash), bh.PreTime)
}

func (p *Processor) prepareForCast(sgi *StaticGroupInfo)  {
	bc := NewBlockContext(p, sgi)

	bc.pos = sgi.GetMinerPos(p.GetMinerID())
	log.Printf("prepareForCast current ID in group pos=%v.\n", bc.pos)
	//to do:只有自己属于这个组的节点才需要调用AddBlockConext
	b := p.AddBlockContext(bc)
	log.Printf("(proc:%v) prepareForCast Add BlockContext result = %v, bc_size=%v.\n", p.getPrefix(), b, p.blockContexts.contextSize())

	bc.registerTicker()
	p.triggerCastCheck()
}

func (p *Processor) verifyBlock(bh *types.BlockHeader) ([]common.Hash, int8) {
	lostTransHash, ret, _, _ := core.BlockChainImpl.VerifyCastingBlock(*bh)
	log.Printf("BlockChainImpl.VerifyCastingBlock result=%v.", ret)
	return lostTransHash, ret
}