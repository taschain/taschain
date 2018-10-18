//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/model"
	"core"
	"fmt"
	"log"
	"middleware/types"
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

func (p *Processor) addFutureBlockMsg(msg *model.ConsensusBlockMessage) {
	b := msg.Block
	log.Printf("future block receive cached! h=%v, hash=%v\n", b.Header.Height, b.Header.Hash.ShortS())

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
	bh := block.Header

	blog := newBizLog("doAddOnChain")
	blog.log("start, height=%v, hash=%v", bh.Height, bh.Hash.ShortS())
	result = p.MainChain.AddBlockOnChain(block)

	//log.Printf("AddBlockOnChain header %v \n", p.blockPreview(bh))
	//log.Printf("QueryTopBlock header %v \n", p.blockPreview(p.MainChain.QueryTopBlock()))
	blog.log("proc(%v) core.AddBlockOnChain, height=%v, hash=%v, result=%v.", p.getPrefix(), bh.Height, bh.Hash.ShortS(), result)
	castor := groupsig.DeserializeId(bh.Castor)
	tlog := newBlockTraceLog("doAddOnChain", bh.Hash, castor)
	tlog.log("result=%v,castor=%v", result, castor.ShortS())

	if result == -1 {
		p.removeFutureVerifyMsgs(block.Header.Hash)
		p.futureRewardReqs.remove(block.Header.Hash)
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
    b := p.MainChain.QueryBlockHeaderByHash(hash)
	return b
}

func (p *Processor) addFutureVerifyMsg(msg *model.ConsensusBlockMessageBase) {
	b := msg.BH
	log.Printf("future verifyMsg receive cached! h=%v, hash=%v, preHash=%v\n", b.Height, b.Hash.ShortS(), b.PreHash.ShortS())

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
	return fmt.Sprintf("hash=%v, height=%v, curTime=%v, preHash=%v, preTime=%v", bh.Hash.ShortS(), bh.Height, bh.CurTime, bh.PreHash.ShortS(), bh.PreTime)
}

func (p *Processor) prepareForCast(sgi *StaticGroupInfo) {
	//组建组网络
	mems := make([]groupsig.ID, 0)
	for _, mem := range sgi.Members {
		mems = append(mems, mem.ID)
	}
	p.NetServer.BuildGroupNet(sgi.GroupID, mems)

	bc := NewBlockContext(p, sgi)

	bc.pos = sgi.GetMinerPos(p.GetMinerID())
	log.Printf("prepareForCast current ID in group pos=%v.\n", bc.pos)
	//to do:只有自己属于这个组的节点才需要调用AddBlockConext
	b := p.AddBlockContext(bc)
	log.Printf("(proc:%v) prepareForCast Add BlockContext result = %v, bc_size=%v.\n", p.getPrefix(), b, p.blockContexts.contextSize())

	//bc.registerTicker()
	//p.triggerCastCheck()
}

func (p *Processor) verifyBlock(bh *types.BlockHeader) ([]common.Hash, int8) {
	lostTransHash, ret, _, _ := core.BlockChainImpl.VerifyBlock(*bh)
	log.Printf("BlockChainImpl.VerifyCastingBlock result=%v.", ret)
	return lostTransHash, ret
}
