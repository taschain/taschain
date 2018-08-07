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
	//begin := time.Now()
	//defer func() {
	//	log.Printf("checkSelfCastRoutine: begin at %v, cost %v", begin, time.Since(begin).String())
	//}()
	if !p.Ready() {
		return false
	}

	if p.belongGroups.groupSize() == 0 || p.blockContexts.contextSize() == 0 {
		log.Printf("current node don't belong to anygroup!!")
		return false
	}

	if p.MainChain.IsAdujsting() {
		log.Printf("checkSelfCastRoutine: isAdjusting, return...\n")
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

	log.Printf("checkSelfCastRoutine: topHeight=%v, topHash=%v, topCurTime=%v, castHeight=%v, expireTime=%v\n", top.Height, GetHashPrefix(top.Hash), top.CurTime, castHeight, expireTime)

	casting := false
	p.blockContexts.forEach(func(_bc *BlockContext) bool {
		if _bc.alreadyInCasting(castHeight, top.Hash) {
			log.Printf("checkSelfCastRoutine: already in cast height, castInfo=%v", _bc.castingInfo())
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

	log.Printf("NEXT CAST GROUP is %v\n", GetIDPrefix(*selectGroup))

	//自己属于下一个铸块组
	if p.IsMinerGroup(*selectGroup) {
		bc := p.GetBlockContext(*selectGroup)
		if bc == nil {
			log.Printf("[ERROR]checkSelfCastRoutine: get nil blockcontext!, gid=%v", GetIDPrefix(*selectGroup))
			return false
		}

		log.Printf("MYGOD! BECOME NEXT CAST GROUP! uid=%v, gid=%v\n", GetIDPrefix(p.GetMinerID()), GetIDPrefix(*selectGroup))
		bc.StartCast(castHeight, expireTime, top)

		return true
	} else { //自己不是下一个铸块组, 但是当前在铸块
		p.blockContexts.forEach(func(_bc *BlockContext) bool {
			log.Printf("reset casting blockcontext: castingInfo=%v", _bc.castingInfo())
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
	data := preBH.Signature

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
func (p *Processor) SuccessNewBlock(bh *types.BlockHeader, vctx *VerifyContext, gid groupsig.ID) {
	//begin := time.Now()
	//defer func() {
	//	log.Printf("SuccessNewBlock begin at %v, cost %v\n", begin.String(), time.Since(begin).String())
	//}()

	if bh == nil {
		panic("SuccessNewBlock arg failed.")
	}

	if p.blockOnChain(bh) { //已经上链
		log.Printf("SuccessNewBlock core.GenerateBlock is nil! block alreayd onchain!")
		vctx.CastedUpdateStatus(int64(bh.QueueNumber))
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
	vctx.CastedUpdateStatus(int64(bh.QueueNumber))

	var cbm model.ConsensusBlockMessage
	cbm.Block = *block
	cbm.GroupID = gid
	ski := model.NewSecKeyInfo(p.GetMinerID(), p.mi.GetDefaultSecKey())
	cbm.GenSign(ski, &cbm)
	if !PROC_TEST_MODE {
		logHalfway("SuccessNewBlock", bh.Height, bh.QueueNumber, p.getPrefix(), "SuccessNewBlock, hash %v, 耗时%v秒", GetHashPrefix(bh.Hash), time.Since(bh.CurTime).Seconds())
		p.NetServer.BroadcastNewBlock(&cbm)
		p.triggerCastCheck()
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
	block := p.MainChain.CastingBlock(uint64(height), uint64(nonce), uint64(qn), p.GetMinerID().Serialize(), gid.Serialize())
	if block == nil {
		log.Printf("MainChain::CastingBlock failed, height=%v, qn=%v, gid=%v, mid=%v.\n", height, qn, GetIDPrefix(gid), GetIDPrefix(p.GetMinerID()))
		//panic("MainChain::CastingBlock failed, jiuci return nil.\n")
		logHalfway("CASTBLOCK", height, uint64(qn), p.getPrefix(), "铸块失败, block为空")
		return nil
	}

	bh := block.Header

	log.Printf("AAAAAA castBlock bh %v, top bh %v\n", p.blockPreview(bh), p.blockPreview(p.MainChain.QueryTopBlock()))

	var si model.SignData
	si.DataHash = bh.Hash
	si.SignMember = p.GetMinerID()

	if bh.Height > 0 && si.DataSign.IsValid() && bh.Height == height && bh.PreHash == vctx.prevHash {
		//发送该出块消息
		var ccm model.ConsensusCastMessage
		ccm.BH = *bh
		//ccm.GroupID = gid
		ccm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), p.getSignKey(gid)), &ccm)

		logHalfway("CASTBLOCK", height, uint64(qn), p.getPrefix(), "铸块成功, SendVerifiedCast, hash %v, 时间间隔 %v", GetHashPrefix(bh.Hash), bh.CurTime.Sub(bh.PreTime).Seconds())
		if !PROC_TEST_MODE {
			p.NetServer.SendCastVerify(&ccm)
		} else {
			for _, proc := range p.GroupProcs {
				proc.OnMessageCast(&ccm)
			}
		}
	} else {
		log.Printf("bh/prehash Error or sign Error, bh=%v, ds=%v, real height=%v. bc.prehash=%v, bh.prehash=%v\n", height, GetSignPrefix(si.DataSign), bh.Height, vctx.prevHash, bh.PreHash)
		//panic("bh Error or sign Error.")
		return nil
	}
	//个人铸块完成的同时也是个人验证完成（第一个验证者）
	//更新共识上下文
	return bh
}