package logical

import (
			"consensus/model"
	"log"
	"time"
	"consensus/logical/pow"
	"middleware/types"
	"consensus/groupsig"
)

/*
**  Creator: pxf
**  Date: 2018/8/9 下午4:06
**  Description: 
*/

//出块完成后，启动pow计算
func (p *Processor) startPowComputation(bh *types.BlockHeader)  {
	gid := groupsig.DeserializeId(bh.GroupId)
	gminer := model.NewGroupMinerID(*gid, p.GetMinerID())
	group := p.getGroup(gminer.Gid)
	if group == nil {
		return
	}
	p.worker.Prepare(bh, gminer, len(group.Members))
	log.Printf("startPowComputation bh=%v\n", p.blockPreview(bh))
	logStart("POW_COMP", p.worker.BH.Height, p.worker.BH.QueueNumber, p.getPrefix(), "", "")
	p.worker.Start()
}


func (p *Processor) powWorkerLoop()  {
    for {
		select {
		case cmd := <- p.worker.CmdCh:
			switch cmd {
			case pow.CMD_POW_RESULT:
				p.onPowComputedDeadline()
			case pow.CMD_POW_CONFIRM:
				p.onPowConfirmDeadline()
			}
		}
	}
}

//算出pow结果，发送给其他组员
func (p *Processor) onPowComputedDeadline()  {
	if !p.worker.Success() {
		return
	}

	msg := &model.ConsensusPowResultMessage{
		BlockHash: p.worker.BH.Hash,
		Nonce:     p.worker.Nonce,
		GroupID:   p.worker.GroupMiner.Gid,
	}

	msg.GenSign(model.NewSecKeyInfo(p.GetMinerID(), p.getMinerGroupSignKey(p.worker.GroupMiner.Gid)), msg)

	logHalfway("POW_Result", p.worker.BH.Height, p.worker.BH.QueueNumber, p.getPrefix(), "nonce %v, cost %v", p.worker.Nonce, time.Since(p.worker.StartTime).String())

	log.Printf("send ConsensusPowResultMessage ...")
	p.NetServer.SendPowResultMessage(msg)
}

func (p *Processor) onPowConfirmDeadline()  {
	if !p.worker.Confirmed() {
		return
	}

	msg := &model.ConsensusPowConfirmMessage{
		GroupID: p.worker.GroupMiner.Gid,
		BlockHash: p.worker.BH.Hash,
		NonceSeq: p.worker.GetConfirmedNonceSeq(),
	}
	msg.GenSign(model.NewSecKeyInfo(p.GetMinerID(), p.getMinerGroupSignKey(p.worker.GroupMiner.Gid)), msg)
	logHalfway("POW_Confirm", p.worker.BH.Height, p.worker.BH.QueueNumber, p.getPrefix(), "nonceSeq %v", MinerNonceSeqDesc(msg.NonceSeq))

	if p.worker.CheckBroadcastStatus(pow.BROADCAST_NONE, pow.BROADCAST_CONFIRM) {
		log.Printf("send ConsensusPowConfirmMessage ...")
		p.NetServer.SendPowConfirmMessage(msg)
	}
}

func (p *Processor) sendPowFinal()  {
	msg := &model.ConsensusPowFinalMessage{
		GroupID: p.worker.GroupMiner.Gid,
		BlockHash: p.worker.BH.Hash,
		NonceSeq: p.worker.GetConfirmedNonceSeq(),
		GSign: *p.worker.GetGSign(),
	}
	msg.GenSign(model.NewSecKeyInfo(p.GetMinerID(), p.getMinerGroupSignKey(p.worker.GroupMiner.Gid)), msg)
	logHalfway("POW_final", p.worker.BH.Height, p.worker.BH.QueueNumber, p.getPrefix(), "nonceSeq %v", MinerNonceSeqDesc(msg.NonceSeq))

	if p.worker.CheckBroadcastStatus(pow.BROADCAST_CONFIRM, pow.BROADCAST_FINAL) {
		log.Printf("send ConsensusPowFinalMessage ...")
		p.NetServer.SendPowFinalMessage(msg)
	}
}

func (p *Processor) persistPowConfirmed() bool {
	if p.worker.CheckBroadcastStatus(pow.BROADCAST_CONFIRM, pow.BROADCAST_PERSIST) || p.worker.CheckBroadcastStatus(pow.BROADCAST_FINAL, pow.BROADCAST_PERSIST)  {
		ret := p.worker.PersistConfirm()
		//触发检查 当前是否到自己组铸块
		p.checkSelfCastRoutine()
		return ret
	}
	return true
}

//区块提案
func (p *Processor) powProposeBlock(bc *BlockContext, vctx *VerifyContext, qn int64) *types.BlockHeader {
	mtype := "CASTBLOCK"
	height := vctx.castHeight

	log.Printf("begin Processor::powProposeBlock, height=%v, qn=%v...\n", height, qn)

	nonce := time.Now().Unix()
	gid := bc.MinerID.Gid

	prePowResult := p.worker.LoadConfirm()
	if prePowResult == nil {	//pow预算还未结束
		logStart(mtype, height, uint64(qn), p.getPrefix(), "pow预算未结束")
		log.Printf("pow preCompute not finished\n")
		return nil
	}
	if prePowResult.GetMinerNonce(p.GetMinerID()) == nil {	//自己不能铸块
		return nil
	}

	logStart(mtype, height, uint64(qn), p.getPrefix(), "开始铸块")

	//调用鸠兹的铸块处理
	block := p.MainChain.CastingBlock(uint64(height), uint64(nonce), uint64(qn), p.GetMinerID().Serialize(), gid.Serialize())
	if block == nil {
		log.Printf("MainChain::CastingBlock failed, height=%v, qn=%v, gid=%v, mid=%v.\n", height, qn, GetIDPrefix(gid), GetIDPrefix(p.GetMinerID()))
		//panic("MainChain::CastingBlock failed, jiuci return nil.\n")
		logHalfway(mtype, height, uint64(qn), p.getPrefix(), "铸块失败, block为空")
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
		sk := p.getMinerGroupSignKey(gid)
		ccm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), sk), &ccm)
		ccm.GenRandSign(sk, vctx.prevRandSig)

		logHalfway(mtype, height, uint64(qn), p.getPrefix(), "铸块成功, SendVerifiedCast, hash %v, 时间间隔 %v", GetHashPrefix(bh.Hash), bh.CurTime.Sub(bh.PreTime).Seconds())

		p.NetServer.SendCastVerify(&ccm)

	} else {
		log.Printf("bh/prehash Error or sign Error, bh=%v, ds=%v, real height=%v. bc.prehash=%v, bh.prehash=%v\n", height, GetSignPrefix(si.DataSign), bh.Height, vctx.prevHash, bh.PreHash)
		//panic("bh Error or sign Error.")
		return nil
	}
	return bh
}