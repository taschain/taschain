package logical

import (
	"log"
	"common"
	"fmt"
	"consensus/groupsig"
	"middleware/types"
	"core"
	"sync"
	"consensus/model"
	"bytes"
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

func (p *Processor) addFutureBlockMsg(msg *model.ConsensusBlockMessage) {
	b := msg.Block
	log.Printf("future block receive cached! h=%v, hash=%v\n", b.Header.Height, GetHashPrefix(b.Header.Hash))

	p.futureBlockMsgs.addMessage(b.Header.PreHash, msg)
}

func (p *Processor) getFutureBlockMsgs(hash common.Hash) []*model.ConsensusBlockMessage {
	if vs := p.futureBlockMsgs.getMessages(hash); vs != nil {
		ret := make([]*model.ConsensusBlockMessage, len(vs))
		for idx, m := range vs {
			ret[idx] = m.(*model.ConsensusBlockMessage)
		}
		return ret
	}
	return nil
}

func (p *Processor) removeFutureBlockMsgs(hash common.Hash) {
	p.futureBlockMsgs.remove(hash)
}

func (p *Processor) doAddOnChain(block *types.Block) (result int8) {
	//begin := time.Now()
	//defer func() {
	//	log.Printf("doAddOnChain begin at %v, cost %v\n", begin.String(), time.Since(begin).String())
	//}()
	result = p.MainChain.AddBlockOnChain(block)

	bh := block.Header

	//log.Printf("AddBlockOnChain header %v \n", p.blockPreview(bh))
	//log.Printf("QueryTopBlock header %v \n", p.blockPreview(p.MainChain.QueryTopBlock()))
	log.Printf("proc(%v) core.AddBlockOnChain, height=%v, level=%v, result=%v.\n", p.getPrefix(), bh.Height, bh.Level, result)
	logHalfway("doAddOnChain", bh.Height, p.getPrefix(), "result=%v,castor=%v", result, GetIDPrefix(*groupsig.DeserializeId(bh.Castor)))

	if result == 0 {
		p.updateLatestBlock(bh)
		p.triggerFutureVerifyMsg(block.Header.Hash)
		p.groupManager.CreateNextGroupRoutine()
		p.cleanVerifyContext(bh.Height)
		p.startPowComputation(bh)
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

func (p *Processor) addFutureVerifyMsg(msg *model.ConsensusBlockMessageBase) {
	b := msg.BH
	log.Printf("future verifyMsg receive cached! h=%v, hash=%v, preHash=%v\n", b.Height, GetHashPrefix(b.Hash), GetHashPrefix(b.PreHash))

	p.futureVerifyMsgs.addMessage(b.PreHash, msg)
}

func (p *Processor) getFutureVerifyMsgs(hash common.Hash) []*model.ConsensusBlockMessageBase {
	if vs := p.futureVerifyMsgs.getMessages(hash); vs != nil {
		ret := make([]*model.ConsensusBlockMessageBase, len(vs))
		for idx, m := range vs {
			ret[idx] = m.(*model.ConsensusBlockMessageBase)
		}
		return ret
	}
	return nil
}

func (p *Processor) removeFutureVerifyMsgs(hash common.Hash) {
	p.futureVerifyMsgs.remove(hash)
}

func (p *Processor) blockPreview(bh *types.BlockHeader) string {
    return fmt.Sprintf("hash=%v, height=%v, level=%v, curTime=%v, preHash=%v, preTime=%v", GetHashPrefix(bh.Hash), bh.Height, bh.Level, bh.CurTime, GetHashPrefix(bh.PreHash), bh.PreTime)
}

func (p *Processor) prepareForCast(sgi *StaticGroupInfo)  {
	bc := NewBlockContext(p, sgi)

	bc.pos = sgi.MemberIndex(p.GetMinerID())
	log.Printf("prepareForCast current ID in group pos=%v.\n", bc.pos)
	b := p.AddBlockContext(bc)
	log.Printf("(proc:%v) prepareForCast Add BlockContext result = %v, bc_size=%v.\n", p.getPrefix(), b, p.blockContexts.contextSize())

	//bc.registerTicker()
	p.triggerCastCheck()
}

func (p *Processor) verifyBlock(bh *types.BlockHeader) ([]common.Hash, int8) {
	lostTransHash, ret, _, _ := core.BlockChainImpl.VerifyCastingBlock(*bh)
	log.Printf("BlockChainImpl.VerifyCastingBlock result=%v.", ret)
	return lostTransHash, ret
}

type latestBlockCache struct {
	blocks sync.Map
}

func (lbc *latestBlockCache) update(bh *types.BlockHeader)  {
	gid := groupsig.DeserializeId(bh.GroupId)
	key := gid.GetHexString()
	if v, load := lbc.blocks.LoadOrStore(key, bh); load {
		existBh := v.(*types.BlockHeader)
		if existBh.Height < bh.Height {	//若已有的高度小于当前高度，则更新
			lbc.blocks.Store(key, bh)
		}
	}
}

func (lbc *latestBlockCache) get(gid groupsig.ID) *types.BlockHeader {
    if v, ok := lbc.blocks.Load(gid.GetHexString()); ok {
    	return v.(*types.BlockHeader)
	} else {
		return nil
	}
}

func (p *Processor) updateLatestBlock(bh *types.BlockHeader)  {
    p.latestBlocks.update(bh)
}

func (p *Processor) getLatestBlock(gid groupsig.ID) *types.BlockHeader {
	if bh := p.latestBlocks.get(gid); bh == nil {
		group := p.getGroup(gid)
		tmpBH := p.MainChain.QueryTopBlock()
		if group.CastQualified(tmpBH.Height) {
			beginHeight := group.BeginHeight
			gidBytes := gid.Serialize()
			for tmpBH != nil && tmpBH.Height >= beginHeight {
				if bytes.Equal(gidBytes, tmpBH.GroupId) {
					p.updateLatestBlock(tmpBH)
					return tmpBH
				}
				tmpBH = p.MainChain.QueryBlockByHash(tmpBH.PreHash)
			}
			return nil
		} else {
			return nil
		}
	} else {
		return bh
	}
}