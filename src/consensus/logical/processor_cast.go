package logical

import (
	"log"
	"time"
	"middleware/types"
	"consensus/groupsig"
	"common"
	"sync"
	"consensus/model"
	"consensus/base"
	"consensus/net"
	"middleware/statistics"
)

/*
**  Creator: pxf
**  Date: 2018/6/27 上午10:39
**  Description: 
*/

type CastBlockContexts struct {
	contexts sync.Map	//string -> *BlockContext
}

func NewCastBlockContexts() *CastBlockContexts {
	return &CastBlockContexts{
		contexts: sync.Map{},
	}
}

func (bctx *CastBlockContexts) addBlockContext(bc *BlockContext) (add bool) {
    _, load := bctx.contexts.LoadOrStore(bc.MinerID.Gid.GetHexString(), bc)
    return !load
}

func (bctx *CastBlockContexts) getBlockContext(gid groupsig.ID) *BlockContext {
	if v, ok := bctx.contexts.Load(gid.GetHexString()); ok {
		return v.(*BlockContext)
	}
	return nil
}

func (bctx *CastBlockContexts) contextSize() int32 {
	size := int32(0)
	bctx.contexts.Range(func(key, value interface{}) bool {
		size ++
		return true
	})
	return size
}

func (bctx *CastBlockContexts) removeContexts(gids []groupsig.ID)  {
	for _, id := range gids {
		log.Println("removeContexts ", GetIDPrefix(id))
		bc := bctx.getBlockContext(id)
		if bc != nil {
			bc.removeTicker()
			bctx.contexts.Delete(id.GetHexString())
		}
	}
}

func (bctx *CastBlockContexts) forEach(f func(bc *BlockContext) bool) {
    bctx.contexts.Range(func(key, value interface{}) bool {
    	v := value.(*BlockContext)
		return f(v)
	})
}


//增加一个铸块上下文（一个组有一个铸块上下文）
func (p *Processor) AddBlockContext(bc *BlockContext) bool {
	var add = p.blockContexts.addBlockContext(bc)
	log.Printf("AddBlockContext, gid=%v, result=%v\n.", GetIDPrefix(bc.MinerID.Gid), add)
	return add
}

//取得一个铸块上下文
//gid:组ID hex 字符串
func (p *Processor) GetBlockContext(gid groupsig.ID) *BlockContext {
	return p.blockContexts.getBlockContext(gid)
}

func (p *Processor) getReleaseRoutineName() string {
	return "release_routine_" + p.getPrefix()
}

//预留接口
//后续如有全局定时器，从这个函数启动
func (p *Processor) Start() bool {
	p.Ticker.RegisterRoutine(p.getCastCheckRoutineName(), p.checkSelfCastRoutine, 4)
	p.Ticker.RegisterRoutine(p.getReleaseRoutineName(), p.releaseRoutine, 2)
	p.Ticker.StartTickerRoutine(p.getReleaseRoutineName(), false)
	p.prepareMiner()
	p.ready = true
	return true
}

//预留接口
func (p *Processor) Stop() {
	return
}

//立即触发一次检查自己是否下个铸块组
func (p *Processor) triggerCastCheck()  {
	//p.Ticker.StartTickerRoutine(p.getCastCheckRoutineName(), true)
	p.Ticker.StartAndTriggerRoutine(p.getCastCheckRoutineName())
}

//检查是否当前组铸块
func (p *Processor) checkSelfCastRoutine() bool {
	if !p.Ready() {
		return false
	}

	blog := newBizLog("checkSelfCastRoutine")

	if p.belongGroups.groupSize() == 0 || p.blockContexts.contextSize() == 0 {
		blog.log("current node don't belong to anygroup!!")
		return false
	}

	if p.MainChain.IsAdujsting() {
		blog.log("isAdjusting, return...")
		p.triggerCastCheck()
		return false
	}

	top := p.MainChain.QueryTopBlock()

	var (
		expireTime time.Time
		castHeight uint64
		deltaHeight uint64
	)
	d := time.Since(top.CurTime)
	if d < 0 {
		return false
	}

	deltaHeight = uint64(d.Seconds()) / uint64(model.Param.MaxGroupCastTime) + 1
	expireTime = GetCastExpireTime(top.CurTime, deltaHeight)

	if top.Height > 0 {
		castHeight = top.Height + deltaHeight
	} else {
		castHeight = uint64(1)
	}

	blog.log("checkSelfCastRoutine: topHeight=%v, topHash=%v, topCurTime=%v, castHeight=%v, expireTime=%v", top.Height, GetHashPrefix(top.Hash), top.CurTime, castHeight, expireTime)

	casting := false
	p.blockContexts.forEach(func(_bc *BlockContext) bool {
		if _bc.alreadyInCasting(castHeight, top.Hash) {
			blog.log("checkSelfCastRoutine: already in cast height, castInfo=%v", _bc.castingInfo())
			casting = true
			return false
		}
		return true
	})
	if casting {
		return true
	}

	selectGroup := p.calcCastGroup(top, castHeight)
	if selectGroup == nil {
		return false
	}

	blog.log("NEXT CAST GROUP is %v", GetIDPrefix(*selectGroup))

	//自己属于下一个铸块组
	if p.IsMinerGroup(*selectGroup) {
		bc := p.GetBlockContext(*selectGroup)
		if bc == nil {
			blog.log("[ERROR]checkSelfCastRoutine: get nil blockcontext!, gid=%v", GetIDPrefix(*selectGroup))
			return false
		}

		blog.log("BECOME NEXT CAST GROUP! castHeight=%v, uid=%v, gid=%v, vctxcnt=%v, castCnt=%v, rHeights=%v", castHeight, GetIDPrefix(p.GetMinerID()), GetIDPrefix(*selectGroup), len(bc.verifyContexts), bc.castedCount, bc.recentCastedHeight)
		//for _, vt := range bc.verifyContexts {
		//	s := ""
		//	slot := ""
		//	for _, sl := range vt.slots {
		//		slot += fmt.Sprintf("(qn %v, piece %v, status %v)", sl.QueueNumber, sl.gSignGenerator.WitnessSize(), sl.slotStatus)
		//	}
		//	s += fmt.Sprintf("h:%v, hash:%v, st:%v, slot:%v", vt.castHeight, GetHashPrefix(vt.prevBH.Hash), vt.consensusStatus, slot)
		//	log.Printf(s)
		//}
		if !bc.StartCast(castHeight, expireTime, top) {
			blog.log("startCast fail, castHeight=%v, expire=%v,topBH=%v", castHeight, expireTime, bc.Proc.blockPreview(top))
		}

		return true
	} else { //自己不是下一个铸块组, 但是当前在铸块
		p.blockContexts.forEach(func(_bc *BlockContext) bool {
			blog.log("reset casting blockcontext: castingInfo=%v", _bc.castingInfo())
			_bc.Reset()
			return true
		})
	}

	return false
}

func (p *Processor) getCastCheckRoutineName() string {
	return "self_cast_check_" + p.getPrefix()
}

func (p *Processor) calcCastGroup(preBH *types.BlockHeader, height uint64) *groupsig.ID {
	var hash common.Hash
	data := preBH.Random

	deltaHeight := height - preBH.Height
	for ; deltaHeight > 0; deltaHeight -- {
		hash = base.Data2CommonHash(data)
		data = hash.Bytes()
	}

	selectGroup, err := p.globalGroups.SelectNextGroup(hash, height)
	if err != nil {
		log.Println("calcCastGroup err:", err)
		return nil
	}
	return &selectGroup
}


//在某个区块高度的QN值成功出块，保存上链，向组外广播
//同一个高度，可能会因QN不同而多次调用该函数
//但一旦低的QN出过，就不该出高的QN。即该函数可能被多次调用，但是调用的QN值越来越小
func (p *Processor) SuccessNewBlock(bh *types.BlockHeader, vctx *VerifyContext, slot *SlotContext, gid groupsig.ID) {

	if bh == nil {
		panic("SuccessNewBlock arg failed.")
	}

	if p.blockOnChain(bh) { //已经上链
		log.Printf("SuccessNewBlock core.GenerateBlock is nil! block alreayd onchain!")
		return
	}

	block := p.MainChain.GenerateBlock(*bh)

	if block == nil {
		log.Printf("SuccessNewBlock core.GenerateBlock is nil! won't broadcast block!")
		return
	}

	r := p.doAddOnChain(block)

	if r != 0 && r != 1 { //分叉调整或 上链失败都不走下面的逻辑
		return
	}

	cbm := &model.ConsensusBlockMessage{
		Block: *block,
	}

	nextId := p.calcCastGroup(bh, bh.Height+1)
	group := p.getGroup(*nextId)
	mems := make([]groupsig.ID, len(group.Members))
	for idx, mem := range group.Members {
		mems[idx] = mem.ID
	}
	next := &net.NextGroup{
		Gid: *nextId,
		MemIds: mems,
	}
	if slot.StatusTransform(SS_VERIFIED, SS_SUCCESS) {
		logHalfway("SuccessNewBlock", bh.Height, bh.QueueNumber, p.getPrefix(), "SuccessNewBlock, hash %v, 耗时%v秒", GetHashPrefix(bh.Hash), time.Since(bh.CurTime).Seconds())
		p.NetServer.BroadcastNewBlock(cbm, next)
	}

	return
}


//检查是否轮到自己出块
func (p *Processor) kingCheckAndCast(bc *BlockContext, vctx *VerifyContext, kingIndex int32, qn int64) {
	//p.castLock.Lock()
	//defer p.castLock.Unlock()
	gid := bc.MinerID.Gid
	height := vctx.castHeight

	//log.Printf("prov(%v) begin kingCheckAndCast, gid=%v, kingIndex=%v, qn=%v, height=%v.\n", p.getPrefix(), GetIDPrefix(gid), kingIndex, qn, height)
	if kingIndex < 0 || qn < 0 {
		return
	}

	sgi := p.getGroup(gid)

	log.Printf("time=%v, Current kingIndex=%v, KING=%v, qn=%v.\n", time.Now().Format(time.Stamp), kingIndex, GetIDPrefix(sgi.GetCastor(int(kingIndex))), qn)
	if sgi.GetCastor(int(kingIndex)).GetHexString() == p.GetMinerID().GetHexString() { //轮到自己铸块
		log.Printf("curent node IS KING!\n")
		if !vctx.isQNCasted(qn) { //在该高度该QN，自己还没铸过快
			head := p.castBlock(bc, vctx, qn) //铸块
			if head != nil {
				vctx.addCastedQN(qn)
			}
		} else {
			log.Printf("In height=%v, qn=%v current node already casted.\n", height, qn)
		}
	}
	return
}

//当前节点成为KING，出块
func (p Processor) castBlock(bc *BlockContext, vctx *VerifyContext, qn int64) *types.BlockHeader {

	height := vctx.castHeight

	log.Printf("begin Processor::castBlock, height=%v, qn=%v...\n", height, qn)
	//var hash common.Hash
	//hash = bh.Hash //TO DO:替换成出块头的哈希
	//to do : change nonce
	nonce := time.Now().Unix()
	gid := bc.MinerID.Gid

	logStart("CASTBLOCK", height, uint64(qn), p.getPrefix(), "开始铸块")

	//调用鸠兹的铸块处理
	block := p.MainChain.CastBlock(uint64(height), uint64(nonce), uint64(qn), p.GetMinerID().Serialize(), gid.Serialize())
	if block == nil {
		log.Printf("MainChain::CastingBlock failed, height=%v, qn=%v, gid=%v, mid=%v.\n", height, qn, GetIDPrefix(gid), GetIDPrefix(p.GetMinerID()))
		//panic("MainChain::CastingBlock failed, jiuci return nil.\n")
		logHalfway("CASTBLOCK", height, uint64(qn), p.getPrefix(), "铸块失败, block为空")
		return nil
	} else {
		//statistics.AddLog(block.Header.Hash.String(), statistics.KingCasting, time.Now().UnixNano(),string(block.Header.Castor),p.GetMinerID().String())
	}

	bh := block.Header

	log.Printf("AAAAAA castBlock bh %v, top bh %v\n", p.blockPreview(bh), p.blockPreview(p.MainChain.QueryTopBlock()))
	if bh.Height > 0 && bh.Height == height && bh.PreHash == vctx.prevBH.Hash {
		skey := p.getSignKey(gid)
		//发送该出块消息
		var ccm model.ConsensusCastMessage
		ccm.BH = *bh
		//ccm.GroupID = gid
		ccm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), skey), &ccm)
		ccm.GenRandomSign(skey, vctx.prevBH.Random)
		logHalfway("CASTBLOCK", height, uint64(qn), p.getPrefix(), "铸块成功, SendVerifiedCast, hash %v, 时间间隔 %v", GetHashPrefix(bh.Hash), bh.CurTime.Sub(bh.PreTime).Seconds())

		p.NetServer.SendCastVerify(&ccm)

		var groupId groupsig.ID
		groupId.Deserialize(ccm.BH.GroupId)
		statistics.AddBlockLog(statistics.SendCast,ccm.BH.Height,ccm.BH.QueueNumber,-1,-1,
			time.Now().UnixNano(),GetIDPrefix(p.GetMinerID()),GetIDPrefix(groupId),common.InstanceIndex,ccm.BH.CurTime.UnixNano())
	} else {
		log.Printf("bh/prehash Error or sign Error, bh=%v, real height=%v. bc.prehash=%v, bh.prehash=%v\n", height, bh.Height, vctx.prevBH.Hash, bh.PreHash)
		//panic("bh Error or sign Error.")
		return nil
	}

	return bh
}