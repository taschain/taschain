package logical

import (
			"consensus/model"
	"log"
	"time"
	"consensus/logical/pow"
	"middleware/types"
	"consensus/groupsig"
	"math/big"
)

/*
**  Creator: pxf
**  Date: 2018/8/9 下午4:06
**  Description: 
*/

func (p *Processor) getPowWorker(gid groupsig.ID) *pow.PowWorker {
    if bcs := p.GetBlockContext(gid); bcs != nil {
    	return bcs.worker
	}
    return nil
}

//出块完成后，启动pow计算
func (p *Processor) startPowComputation(bh *types.BlockHeader)  {
	gid := *groupsig.DeserializeId(bh.GroupId)
	if !p.IsMinerGroup(gid) {
		return
	}
	bc := p.GetBlockContext(gid)
	bc.startPowComputation(bh)
}

//算出pow结果，发送给其他组员
func (p *Processor) onPowComputedDeadline(worker *pow.PowWorker)  {
	if !worker.Success() {
		return
	}

	msg := &model.ConsensusPowResultMessage{
		BlockHash: worker.BH.Hash,
		Nonce:     worker.Nonce,
		GroupID:   worker.GroupMiner.Gid,
	}

	msg.GenSign(model.NewSecKeyInfo(p.GetMinerID(), p.getMinerGroupSignKey(worker.GroupMiner.Gid)), msg)

	logHalfway("POW_Result", worker.BH.Height, p.getPrefix(), "nonce %v, cost %v", worker.Nonce, time.Since(worker.StartTime).String())

	log.Printf("send ConsensusPowResultMessage ...")
	p.NetServer.SendPowResultMessage(msg)
}

func (p *Processor) onPowConfirmDeadline(worker *pow.PowWorker)  {
	if !worker.Confirmed() {
		return
	}

	msg := &model.ConsensusPowConfirmMessage{
		GroupID: worker.GroupMiner.Gid,
		BlockHash: worker.BH.Hash,
		NonceSeq: worker.GetConfirmedNonceSeq(),
	}
	msg.GenSign(model.NewSecKeyInfo(p.GetMinerID(), p.getMinerGroupSignKey(worker.GroupMiner.Gid)), msg)
	logHalfway("POW_Confirm", worker.BH.Height, p.getPrefix(), "nonceSeq %v", MinerNonceSeqDesc(msg.NonceSeq))

	if worker.CheckBroadcastStatus(pow.BROADCAST_NONE, pow.BROADCAST_CONFIRM) {
		log.Printf("send ConsensusPowConfirmMessage ...")
		p.NetServer.SendPowConfirmMessage(msg)
	}
}

func (p *Processor) sendPowFinal(worker *pow.PowWorker)  {
	msg := &model.ConsensusPowFinalMessage{
		GroupID: worker.GroupMiner.Gid,
		BlockHash: worker.BH.Hash,
		NonceSeq: worker.GetConfirmedNonceSeq(),
		GSign: *worker.GetGSign(),
	}
	msg.GenSign(model.NewSecKeyInfo(p.GetMinerID(), p.getMinerGroupSignKey(worker.GroupMiner.Gid)), msg)
	logHalfway("POW_final", worker.BH.Height, p.getPrefix(), "nonceSeq %v", MinerNonceSeqDesc(msg.NonceSeq))

	if worker.CheckBroadcastStatus(pow.BROADCAST_CONFIRM, pow.BROADCAST_FINAL) {
		log.Printf("send ConsensusPowFinalMessage ...")
		p.NetServer.SendPowFinalMessage(msg)
	}
}

func (p *Processor) persistPowConfirmed(worker *pow.PowWorker) bool {
	if worker.CheckBroadcastStatus(pow.BROADCAST_CONFIRM, pow.BROADCAST_PERSIST) || worker.CheckBroadcastStatus(pow.BROADCAST_FINAL, pow.BROADCAST_PERSIST)  {
		ret := worker.PersistConfirm()
		//触发检查 当前是否到自己组铸块
		p.checkSelfCastRoutine()
		return ret
	}
	return true
}

func (p *Processor) checkBlockNonces(bh *types.BlockHeader, gid groupsig.ID) bool {
	groupInfo := p.getGroup(gid)
	minerNonces := GetMinerNonceFromBlockHeader(bh, groupInfo)
	latestBlock := p.latestBlocks.get(gid)
	if latestBlock == nil {
		latestBlock = p.MainChain.QueryBlockByHeight(0)
	}

	diffculty := pow.DIFFCULTY_20_24
	totalLevel := uint32(0)
	lastValue := new(big.Int).SetInt64(0)
	for _, mn := range minerNonces {	//校验难度是否符合，计算值是否递增
		dv, ok := pow.CheckMinerNonce(diffculty, latestBlock.Hash, &mn)
		if ok && dv.Cmp(lastValue) > 0 {
			lastValue = dv
			totalLevel += diffculty.Level(dv)
		} else {
			log.Printf("miner nonce error, id=%v, nonce=%v\n", GetIDPrefix(mn.MinerID), mn.Nonce)
			return false
		}
	}
	return totalLevel != bh.Level
}
