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

//func (p *Processor) addFutureBlockMsg(msg *model.ConsensusBlockMessage) {
//	b := msg.Block
//	log.Printf("future block receive cached! h=%v, hash=%v\n", b.Header.Height, b.Header.Hash.ShortS())
//
//	p.futureBlockMsgs.addMessage(b.Header.PreHash, msg)
//}
//
//func (p *Processor) getFutureBlockMsgs(hash common.Hash) []*model.ConsensusBlockMessage {
//	if vs := p.futureBlockMsgs.getMessages(hash); vs != nil {
//		ret := make([]*model.ConsensusBlockMessage, len(vs))
//		for idx, m := range vs {
//			ret[idx] = m.(*model.ConsensusBlockMessage)
//		}
//		return ret
//	}
//	return nil
//}
//
//func (p *Processor) removeFutureBlockMsgs(hash common.Hash) {
//	p.futureBlockMsgs.remove(hash)
//}

func (p *Processor) doAddOnChain(block *types.Block) (result int8) {
	//begin := time.Now()
	//defer func() {
	//	log.Printf("doAddOnChain begin at %v, cost %v\n", begin.String(), time.Since(begin).String())
	//}()
	bh := block.Header

	rlog := newRtLog("doAddOnChain")
	//blog.log("start, height=%v, hash=%v", bh.Height, bh.Hash.ShortS())
	result = p.MainChain.AddBlockOnChain("", block)

	//log.Printf("AddBlockOnChain header %v \n", p.blockPreview(bh))
	//log.Printf("QueryTopBlock header %v \n", p.blockPreview(p.MainChain.QueryTopBlock()))
	rlog.log("height=%v, hash=%v, result=%v.", p.getPrefix(), bh.Height, bh.Hash.ShortS(), result)
	castor := groupsig.DeserializeId(bh.Castor)
	tlog := newHashTraceLog("doAddOnChain", bh.Hash, castor)
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
	stdLogger.Debugf("future verifyMsg receive cached! h=%v, hash=%v, preHash=%v\n", b.Height, b.Hash.ShortS(), b.PreHash.ShortS())

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
	p.NetServer.BuildGroupNet(sgi.GroupID.GetHexString(), sgi.GetMembers())

	bc := NewBlockContext(p, sgi)

	bc.pos = sgi.GetMinerPos(p.GetMinerID())
	stdLogger.Debugf("prepareForCast current ID in group pos=%v.\n", bc.pos)
	//to do:只有自己属于这个组的节点才需要调用AddBlockConext
	b := p.AddBlockContext(bc)
	stdLogger.Infof("(proc:%v) prepareForCast Add BlockContext result = %v, bc_size=%v.\n", p.getPrefix(), b, p.blockContexts.blockContextSize())

	//bc.registerTicker()
	//p.triggerCastCheck()
}

func (p *Processor) verifyBlock(bh *types.BlockHeader) ([]common.Hash, int8) {
	lostTransHash, ret := core.BlockChainImpl.VerifyBlock(*bh)
	stdLogger.Infof("BlockChainImpl.VerifyCastingBlock result=%v.", ret)
	return lostTransHash, ret
}

func (p *Processor) getNearestBlockByHeight(h uint64) *types.Block {
	for {
		bh := p.MainChain.QueryBlockByHeight(h)
		if bh != nil {
			b := p.MainChain.QueryBlockByHash(bh.Hash)
			if b != nil {
				return b
			} else {
				//bh2 := p.MainChain.QueryBlockByHeight(h)
				//stdLogger.Debugf("get bh not nil, but block is nil! hash1=%v, hash2=%v, height=%v", bh.Hash.ShortS(), bh2.Hash.ShortS(), bh.Height)
				//if bh2.Hash == bh.Hash {
				//	panic("chain queryBlockByHash nil!")
				//} else {
				//	continue
				//}
			}
		}
		if h == 0 {
			panic("cannot find block of height 0")
		}
		h--
	}
}

func (p *Processor) getNearestVerifyHashByHeight(h uint64) (realHeight uint64, vhash common.Hash) {
	for {
		hash, err := p.MainChain.GetCheckValue(h)
		if err == nil {
			return h, hash
		}
		if h == 0 {
			panic("cannot find verifyHash of height 0")
		}
		h--
	}
}

func (p *Processor) VerifyBlock(bh *types.BlockHeader, preBH *types.BlockHeader) (ok bool, err error) {
	tlog := newMsgTraceLog("VerifyBlock", bh.Hash.ShortS(), "")
	defer func() {
		tlog.log("preHash=%v, height=%v, result=%v %v", bh.PreHash.ShortS(), bh.Height, ok, err)
		newBizLog("VerifyBlock").log("hash=%v, preHash=%v, height=%v, result=%v %v", bh.Hash.ShortS(), bh.PreHash.ShortS(), bh.Height, ok, err)
	}()
	if bh.Hash != bh.GenHash() {
		err = fmt.Errorf("block hash error")
		return
	}
	if preBH.Hash != bh.PreHash {
		err = fmt.Errorf("preHash error")
		return
	}

	if ok2, group, err2 := p.isCastLegal(bh, preBH); !ok2 {
		err = err2
		return
	} else {
		gpk := group.GroupPK
		sig := groupsig.DeserializeSign(bh.Signature)
		b := groupsig.VerifySig(gpk, bh.Hash.Bytes(), *sig)
		if !b {
			err = fmt.Errorf("signature verify fail")
			return
		}
		rsig := groupsig.DeserializeSign(bh.Random)
		b = groupsig.VerifySig(gpk, preBH.Random, *rsig)
		if !b {
			err = fmt.Errorf("random verify fail")
			return
		}
	}
	ok = true
	return
}

func (p *Processor) VerifyBlockHeader(bh *types.BlockHeader) (ok bool, err error) {
	if bh.Hash != bh.GenHash() {
		err = fmt.Errorf("block hash error")
		return
	}

	gid := groupsig.DeserializeId(bh.GroupId)
	gpk := p.getGroupPubKey(gid)
	sig := groupsig.DeserializeSign(bh.Signature)
	b := groupsig.VerifySig(gpk, bh.Hash.Bytes(), *sig)
	if !b {
		err = fmt.Errorf("signature verify fail")
		return
	}
	ok = true
	return
}
