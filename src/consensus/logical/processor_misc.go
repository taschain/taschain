package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/model"
	"middleware/types"
	"consensus/base"
	"github.com/vmihailenco/msgpack"
	"fmt"
	"time"
	"encoding/json"
)

/*
**  Creator: pxf
**  Date: 2018/6/12 下午6:12
**  Description:
 */



//后续如有全局定时器，从这个函数启动
func (p *Processor) Start() bool {
	p.Ticker.RegisterRoutine(p.getCastCheckRoutineName(), p.checkSelfCastRoutine, 1)
	p.Ticker.RegisterRoutine(p.getReleaseRoutineName(), p.releaseRoutine, 2)
	p.Ticker.RegisterRoutine(p.getBroadcastRoutineName(), p.broadcastRoutine, 1)
	p.Ticker.StartTickerRoutine(p.getReleaseRoutineName(), false)
	p.Ticker.StartTickerRoutine(p.getBroadcastRoutineName(), false)
	p.triggerCastCheck()
	p.prepareMiner()
	p.ready = true
	return true
}

//预留接口
func (p *Processor) Stop() {
	return
}

func (p *Processor) prepareMiner() {

	topHeight := p.MainChain.QueryTopBlock().Height

	stdLogger.Infof("prepareMiner get groups from groupchain")
	iterator := p.GroupChain.NewIterator()
	groups := make([]*StaticGroupInfo, 0)
	for coreGroup := iterator.Current(); coreGroup != nil; coreGroup = iterator.MovePre(){
		stdLogger.Infof("get group from core, id=%+v", coreGroup.Header)
		if coreGroup.Id == nil || len(coreGroup.Id) == 0 {
			continue
		}
		needBreak := false
		sgi := NewSGIFromCoreGroup(coreGroup)
		if sgi.Dismissed(topHeight) {
			needBreak = true
			genesis := p.GroupChain.GetGroupByHeight(0)
			if coreGroup == nil {
				panic("get genesis group nil")
			}
			sgi = NewSGIFromCoreGroup(genesis)
		}
		groups = append(groups, sgi)
		stdLogger.Infof("load group=%v, beginHeight=%v, topHeight=%v\n", sgi.GroupID.ShortS(), sgi.getGroupHeader().WorkHeight, topHeight)
		if sgi.MemExist(p.GetMinerID()) {
			jg := p.belongGroups.getJoinedGroup(sgi.GroupID)
			if jg == nil {
				stdLogger.Infof("prepareMiner get join group fail, gid=%v\n", sgi.GroupID.ShortS())
			} else {
				p.joinGroup(jg)
			}
		}
		if needBreak {
			break
		}
	}
	for i := len(groups)-1; i >=0; i-- {
		p.acceptGroup(groups[i])
	}
	stdLogger.Infof("prepare finished")
}

func (p *Processor) Ready() bool {
	return p.ready
}

func (p *Processor) GetAvailableGroupsAt(height uint64) []*StaticGroupInfo {
	return p.globalGroups.GetAvailableGroups(height)
}

func (p *Processor) GetCastQualifiedGroups(height uint64) []*StaticGroupInfo {
	return p.globalGroups.GetCastQualifiedGroups(height)
}

func (p *Processor) Finalize() {
	if p.belongGroups != nil {
		p.belongGroups.close()
	}
}



func (p *Processor) getVrfWorker() *vrfWorker {
	if v := p.vrf.Load(); v != nil {
		return v.(*vrfWorker)
	}
	return nil
}

func (p *Processor) setVrfWorker(vrf *vrfWorker) {
	p.vrf.Store(vrf)
}

func (p *Processor) GetSelfMinerDO() *model.SelfMinerDO {
	md := p.minerReader.getProposeMiner(p.GetMinerID())
	if md != nil {
		p.mi.MinerDO = *md
	}
	return p.mi
}

func (p *Processor) canProposalAt(h uint64) bool {
	miner := p.minerReader.getProposeMiner(p.GetMinerID())
	if miner == nil {
		return false
	}
   	return miner.CanCastAt(h)
}

func (p *Processor) GetJoinedWorkGroupNums() (work, avail int) {
	h := p.MainChain.QueryTopBlock().Height
	groups := p.globalGroups.GetAvailableGroups(h)
	for _, g := range groups {
		if !g.MemExist(p.GetMinerID()) {
			continue
		}
		if g.CastQualified(h) {
			work++
		}
		avail++
	}
	return
}

func (p *Processor) CalcBlockHeaderQN(bh *types.BlockHeader) uint64 {
	pi := base.VRFProve(bh.ProveValue.Bytes())
	castor := groupsig.DeserializeId(bh.Castor)
	miner := p.minerReader.getProposeMiner(castor)
	if miner == nil {
		stdLogger.Infof("CalcBHQN getMiner nil id=%v, bh=%v", castor.ShortS(), bh.Hash.ShortS())
		return 0
	}
	totalStake := p.minerReader.getTotalStake(bh.Height)
	_, qn := vrfSatisfy(pi, miner.Stake, totalStake)
	return qn
}

func marshalBlock(b types.Block) ([]byte, error) {
	if b.Transactions != nil && len(b.Transactions) == 0 {
		b.Transactions = nil
	}
	if b.Header.Transactions != nil && len(b.Header.Transactions) == 0 {
		b.Header.Transactions = nil
	}
	return msgpack.Marshal(&b)
}

func (p *Processor) GenVerifyHash(b *types.Block, id groupsig.ID) common.Hash {
	buf, err := marshalBlock(*b)
	if err != nil {
		panic(fmt.Sprintf("marshal block error, hash=%v, err=%v", b.Header.Hash.ShortS(), err))
	}
	//header := &b.Header
	//log.Printf("GenVerifyHash aaa bufHash=%v, buf %v", base.Data2CommonHash(buf).ShortS(), buf)
	//log.Printf("GenVerifyHash aaa headerHash=%v, genHash=%v", b.Header.Hash.ShortS(), b.Header.GenHash().ShortS())

	//headBuf, _ := msgpack.Marshal(header)
	//log.Printf("GenVerifyHash aaa headerBufHash=%v, headerBuf=%v", base.Data2CommonHash(headBuf).ShortS(), headBuf)

	//log.Printf("GenVerifyHash height:%v,id:%v,%v, bbbbbuf %v", b.Header.Height,id.ShortS(), b.Transactions == nil, buf)
	//log.Printf("GenVerifyHash height:%v,id:%v,bbbbbuf ids %v", b.Header.Height,id.ShortS(),id.Serialize())
	buf = append(buf, id.Serialize()...)
	//log.Printf("GenVerifyHash height:%v,id:%v,bbbbbuf after %v", b.Header.Height,id.ShortS(),buf)
	h := base.Data2CommonHash(buf)
	//log.Printf("GenVerifyHash height:%v,id:%v,bh:%v,vh:%v", b.Header.Height,id.ShortS(),b.Header.Hash.ShortS(), h.ShortS())
	return h
}

func (p *Processor) CheckGroupHeader(gh *types.GroupHeader, pSign groupsig.Signature) (bool, error) {
	if ok, err := p.groupManager.isGroupHeaderLegal(gh); ok {
		//验证父亲组签名
		pid := groupsig.DeserializeId(gh.Parent)
		ppk := p.getGroupPubKey(pid)
		if groupsig.VerifySig(ppk, gh.Hash.Bytes(), pSign) {
			return true, nil
		} else {
			return false, fmt.Errorf("signature verify fail, pk=", ppk.GetHexString())
		}
	} else {
		return false, err
	}
}

func (p *Processor) BlockContextSummary() string {

	type slotSummary struct {
		Hash string `json:"hash"`
		GSigSize int `json:"g_sig_size"`
		RSigSize int `json:"r_sig_size"`
		TxSigSize int `json:"tx_sig_size"`
		LostTxSize int `json:"lost_tx_size"`
		Status int32 `json:"status"`
	}
	type vctxSummary struct {
		CastHeight uint64 `json:"cast_height"`
		Status int32 `json:"status"`
		Slots  []*slotSummary `json:"slots"`
		NumSlots int `json:"num_slots"`
		Expire time.Time `json:"expire"`
		ShouldRemove bool `json:"should_remove"`
	}
	type bctxSummary struct {
    	Gid string `json:"gid"`
    	NumRvh int `json:"num_rvh"`
    	NumVctx int `json:"num_vctx"`
    	Vctxs []*vctxSummary `json:"vctxs"`
	}
	type contextSummary struct {
		NumBctxs int `json:"num_bctxs"`
		Bctxs []*bctxSummary `json:"bctxs"`
		NumReserVctx int `json:"num_reser_vctx"`
		ReservVctxs []*vctxSummary `json:"reserv_vctxs"`
		NumFutureVerifyMsg int `json:"num_future_verify_msg"`
		NumFutureRewardMsg int `json:"num_future_reward_msg"`
	}
	bctxs := make([]*bctxSummary, 0)
	p.blockContexts.forEachBlockContext(func(bc *BlockContext) bool {
		vs := make([]*vctxSummary, 0)
		for _, vctx := range bc.SafeGetVerifyContexts() {
			ss := make([]*slotSummary, 0)
			for _, slot := range vctx.GetSlots() {
				s := &slotSummary{
					Hash: slot.BH.Hash.String(),
					GSigSize: slot.gSignGenerator.WitnessSize(),
					RSigSize: slot.rSignGenerator.WitnessSize(),
					LostTxSize: slot.lostTxHash.Size(),
					Status: slot.GetSlotStatus(),
				}
				if slot.rewardGSignGen != nil {
					s.TxSigSize = slot.rewardGSignGen.WitnessSize()
				}
				ss = append(ss, s)
			}
			v := &vctxSummary{
				CastHeight: vctx.castHeight,
				Status: vctx.consensusStatus,
				NumSlots: len(vctx.slots),
				Expire: vctx.expireTime,
				ShouldRemove: vctx.castRewardSignExpire() || (vctx.broadcastSlot != nil && vctx.broadcastSlot.IsRewardSent()),
				Slots:ss,
			}
			vs = append(vs, v)
		}
		b := &bctxSummary{
			Gid: bc.MinerID.Gid.GetHexString(),
			NumRvh: len(bc.recentCasted),
			NumVctx: len(vs),
			Vctxs: vs,
		}
		bctxs = append(bctxs, b)
		return true
	})
	reservVctxs := make([]*vctxSummary, 0)
	p.blockContexts.forEachReservedVctx(func(vctx *VerifyContext) bool {
		v := &vctxSummary{
			CastHeight: vctx.castHeight,
			Status: vctx.consensusStatus,
			NumSlots: len(vctx.slots),
			Expire: vctx.expireTime,
			ShouldRemove: vctx.castRewardSignExpire() || (vctx.broadcastSlot != nil && vctx.broadcastSlot.IsRewardSent()),
		}
		reservVctxs = append(reservVctxs, v)
		return true
	})
	cs := &contextSummary{
		Bctxs: bctxs,
		ReservVctxs: reservVctxs,
		NumBctxs:len(bctxs),
		NumReserVctx: len(reservVctxs),
		NumFutureVerifyMsg: p.futureVerifyMsgs.size(),
		NumFutureRewardMsg: p.futureRewardReqs.size(),
	}
	b, _ := json.MarshalIndent(cs, "", "\t")
	fmt.Printf("%v\n", string(b))
	fmt.Println("============================================================")
	return string(b)
}

func (p *Processor) GetJoinGroupInfo(gid string) *JoinedGroup {
	var id groupsig.ID
	id.SetHexString(gid)
    jg := p.belongGroups.getJoinedGroup(id)
    return jg
}