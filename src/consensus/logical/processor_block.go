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
	"fmt"
	"middleware/types"
	"sync"
	"bytes"
	"time"
	"monitor"
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

func (holder *FutureMessageHolder) forEach(f func(key common.Hash, arr []interface{}) bool) {
	holder.messages.Range(func(key, value interface{}) bool {
		arr := value.([]interface{})
		return f(key.(common.Hash), arr)
	})
}

func (holder *FutureMessageHolder) size() int {
	cnt := 0
	holder.forEach(func(key common.Hash, value []interface{}) bool {
		cnt += len(value)
		return true
	})
	return cnt
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

	bh := block.Header

	traceLog := monitor.NewPerformTraceLogger("DoAddOnChain", bh.Hash, bh.Height)
	defer func() {
		traceLog.Log("result:%v", result)
	}()

	rlog := newRtLog("doAddOnChain")
	result = int8(p.MainChain.AddBlockOnChain("", block))

	rlog.log("height=%v, hash=%v, result=%v.", bh.Height, bh.Hash.ShortS(), result)
	castor := groupsig.DeserializeId(bh.Castor)
	tlog := newHashTraceLog("doAddOnChain", bh.Hash, castor)
	tlog.log("result=%v,castor=%v", result, castor.ShortS())

	if result == -1 {
		p.removeFutureVerifyMsgs(block.Header.Hash)
		p.futureRewardReqs.remove(block.Header.Hash)
	}

	return result

}

func (p *Processor) blockOnChain(h common.Hash) bool {
	return p.MainChain.HasBlock(h)
}

func (p *Processor) getBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	begin := time.Now()
	defer func() {
		if time.Since(begin).Seconds() > 0.5 {
			slowLogger.Warnf("slowQueryBlockHeaderByHash: cost %v, hash=%v", time.Since(begin).String(), hash.ShortS())
		}
	}()
	b := p.MainChain.QueryBlockHeaderByHash(hash)
	return b
}

func (p *Processor) addFutureVerifyMsg(msg *model.ConsensusCastMessage) {
	b := msg.BH
	stdLogger.Debugf("future verifyMsg receive cached! h=%v, hash=%v, preHash=%v\n", b.Height, b.Hash.ShortS(), b.PreHash.ShortS())

	p.futureVerifyMsgs.addMessage(b.PreHash, msg)
}

func (p *Processor) getFutureVerifyMsgs(hash common.Hash) []*model.ConsensusCastMessage {
	if vs := p.futureVerifyMsgs.getMessages(hash); vs != nil {
		ret := make([]*model.ConsensusCastMessage, len(vs))
		for idx, m := range vs {
			ret[idx] = m.(*model.ConsensusCastMessage)
		}
		return ret
	}
	return nil
}

func (p *Processor) removeFutureVerifyMsgs(hash common.Hash) {
	p.futureVerifyMsgs.remove(hash)
}

func (p *Processor) blockPreview(bh *types.BlockHeader) string {
	return fmt.Sprintf("hash=%v, height=%v, curTime=%v, preHash=%v, preTime=%v", bh.Hash.ShortS(), bh.Height, bh.CurTime, bh.PreHash.ShortS(), bh.CurTime.Add(-int64(bh.Elapsed)))
}

func (p *Processor) prepareForCast(sgi *StaticGroupInfo) {
	//组建组网络
	p.NetServer.BuildGroupNet(sgi.GroupID.GetHexString(), sgi.GetMembers())
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

func (p *Processor) VerifyGroup(g *types.Group) (ok bool, err error) {
	if len(g.Signature) == 0 {
		return false, fmt.Errorf("sign is empty")
	}
	//top := p.MainChain.Height()
	//if top > g.Header.WorkHeight {
	//	err = fmt.Errorf("group too late for work, workheight %v, top %v", g.Header.WorkHeight, top)
	//	return
	//}
	mems := make([]groupsig.ID, len(g.Members))
	for idx, mem := range g.Members {
		mems[idx] = groupsig.DeserializeId(mem)
	}
	gInfo := &model.ConsensusGroupInitInfo{
		GI: model.ConsensusGroupInitSummary{
			Signature: *groupsig.DeserializeSign(g.Signature),
			GHeader: 	g.Header,
		},
		Mems: mems,
	}

	//检验头和签名
	if _, ok, err := p.groupManager.checkGroupInfo(gInfo); ok {
		gpk := groupsig.DeserializePubkeyBytes(g.PubKey)
		gid := groupsig.NewIDFromPubkey(gpk).Serialize()
		if !bytes.Equal(gid, g.Id) {
			return false, fmt.Errorf("gid error, expect %v, receive %v", gid, g.Id)
		}
	} else {
		return false, err
	}
	ok = true
	return
}
