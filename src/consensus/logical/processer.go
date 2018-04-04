package logical

import (
	"consensus/groupsig"
	"fmt"
	"time"

	"common"
)

//计算当前距上一个铸块完成已经过去了几个铸块时间窗口（组间）
func getBlockTimeWindow(b time.Time) int32 {
	diff := time.Since(b).Seconds() //从上个铸块完成到现在的时间（秒）
	if diff >= 0 {
		return int32(diff) / MAX_GROUP_BLOCK_TIME
	} else {
		return -1
	}
}

//计算当前距上一个铸块完成已经过去了几个出块时间窗口（组内）
func getCastTimeWindow(b time.Time) int32 {
	diff := time.Since(b).Seconds() //从上个铸块完成到现在的时间（秒）
	if diff >= 0 {
		return int32(diff) / MAX_USER_CAST_TIME
	} else {
		return -1
	}
}

//自己的出块信息
type SelfCastInfo struct {
	block_qns map[uint][]uint //当前节点已经出过的块(高度->出块QN列表)
}

func (sci *SelfCastInfo) FindQN(height uint, newQN uint) bool {
	qns, ok := sci.block_qns[height]
	if ok {
		for _, qn := range qns {
			if qn == newQN { //该newQN已存在
				return true
			}
		}
		return false
	} else {
		return false
	}
}

//如该QN已存在，则返回false
func (sci *SelfCastInfo) AddQN(height uint, newQN uint) bool {
	qns, ok := sci.block_qns[height]
	if ok {
		for _, qn := range qns {
			if qn == newQN { //该newQN已存在
				return false
			}
		}
		sci.block_qns[height] = append(sci.block_qns[height], newQN)
		return true
	} else {
		sci.block_qns[height] = []uint{newQN}
		return true
	}
	return false
}

//见证人处理器
type Processer struct {
	gc   *GroupContext   //组初始化上下文(组初始化完成上链后，不再需要)
	bc   BlockContext    //铸块上下文
	gg   GlobalGroups    //全网组静态信息
	gid  groupsig.ID     //当前节点所属组ID
	uid  groupsig.ID     //当前节点ID
	gusk groupsig.Seckey //组成员（铸块）签名私钥
	usk  groupsig.Seckey //个人私钥，和组无关（用全网组静态信息里所属组保存的个人公钥可以验签）
	sci  SelfCastInfo    //当前节点的铸块信息（包括当前节点在不同高度不同QN值所有成功和不成功的出块）
}

func (p *Processer) InitProcesser() {
	//to do ： 从链上加载和初始化成员变量
	return
}

//检查区块头是否合法
func (p Processer) isBHCastLegal(bh BlockHeader, sd SignData) (result bool) {
	//to do : 检查是否基于链上最高块的出块
	gi := p.gg.GetCastGroup(bh.PreHash) //取得合法的铸块组
	if gi.GroupID == sd.SignMember {
		//检查组签名是否正确
		result = sd.VerifySign(gi.GroupPK)
	}
	return result
}

//收到组内成员的出块消息，铸块人用组分片密钥进行了签名
func (p Processer) OnMessageCast(ccm ConsensusCastMessage) {
	if !p.bc.IsCasting() { //当前没有在组铸块中
		fmt.Printf("processer::OnMessageCast failed, group not in cast.\n")
		return
	}
	cs := GenConsensusSummary(ccm.bh, ccm.si)
	n := p.bc.UserCasted(cs)
	fmt.Printf("processer:OnMessageCast UserCasted result=%v.\n", n)
	if n == CBMR_THRESHOLD_SUCCESS {
		b := p.bc.VerifyGroupSign(cs, p.getSelfGroup().GroupPK)
		if b { //组签名验证通过
			p.SuccessNewBlock(cs) //上链和组外广播
		}
	}
	return
}

func (p Processer) OnMessageVerify(cvm ConsensusVerifyMessage) {
	cs := GenConsensusSummary(cvm.bh, cvm.si)
	n := p.bc.UserVerified(cs)
	fmt.Printf("processer::OnMessageVerify UserVerified result=%v.\n", n)
	if n == CBMR_THRESHOLD_SUCCESS {
		b := p.bc.VerifyGroupSign(cs, p.getSelfGroup().GroupPK)
		if b { //组签名验证通过
			p.SuccessNewBlock(cs) //上链和组外广播
		}
	}
	return
}

//收到铸块上链消息
func (p Processer) OnMessageBlock(cbm ConsensusBlockMessage) {
	if p.isBHCastLegal(cbm.bh, cbm.si) { //铸块头合法
		//to do : 鸠兹上链保存
		next_group, err := p.gg.SelectNextGroup(cbm.si.DataHash) //查找下一个铸块组
		if err == nil {
			if next_group == p.gid { //自身属于下一个铸块组
				p.bc.BeingCastGroup(cbm.bh.BlockHeight, cbm.bh.PreTime, cbm.si.DataHash)
				//to do : 屮逸组内广播
			}
		} else {
			panic("find next cast group failed.")
		}
	} else {
		//丢弃该块
		fmt.Printf("received invalid new block, height = %v.\n", cbm.bh.BlockHeight)
	}
}

//收到成为当前组消息
func (p Processer) OnMessageCurrent(ccm ConsensusCurrentMessage) {
	gi, err := p.gg.GetGroupByID(p.gid)
	if err == nil {
		ru, ok := gi.GetMember(ccm.si.SignMember) //检查发消息用户是否跟当前节点同组
		if ok {                                   //该用户和我是同一组
			if ccm.si.VerifySign(ru.pk) { //消息合法
				p.bc.BeingCastGroup(ccm.BlockHeight, ccm.PreTime, ccm.PreHash)
				//to do : 屮逸组内广播
			}
		}
	} else {
		panic("OnMessageCrrent failed, invalid gid.")
	}
}

//在某个区块高度的QN值成功出块，保存上链，向组外广播
//同一个高度，可能会因QN不同而多次调用该函数
//但一旦低的QN出过，就不该出高的QN。即该函数可能被多次调用，但是调用的QN值越来越小
func (p Processer) SuccessNewBlock(cs ConsensusBlockSummary) {
	//鸠兹保存上链
	//屮逸组外广播
	p.bc.CastedUpdateStatus(uint(cs.QueueNumber))
	p.bc.SignedUpdateMinQN(uint(cs.QueueNumber))
	return
}

func (p Processer) CheckCastRoutine(user_index int32, qn int64, height uint) {
	if user_index < 0 || qn < 0 {
		return
	}

	if p.getSelfGroup().GetCastor(int(user_index)) == p.uid { //轮到自己铸块
		if p.sci.AddQN(height, uint(qn)) { //在该高度该QN，自己还没铸过快
			p.castBlock(qn) //铸块
		}
	}
	return
}

///////////////////////////////////////////////////////////////////////////////
//组初始化相关消息
//to do : 之前我跟哪些节点属于同一组的信息保存在哪里？
func (p Processer) OnMessageGroupInit(grm ConsensusGroupRawMessage) {
	if p.gg.GetGroupStatus(p.gid) != SGS_INITING {
		return //当前节点所在的组不需要初始化
	}
	if p.gc == nil {
		//to do : 需要初始化
		fmt.Printf("OnMessageGroupInit failed, receive GROUPINIT msg when gc=nil.\n")
		return
	}
	if p.gc.RawMeesage(grm) { //第一次收到该消息
		pieces := p.gc.GenSharePieces() //生成秘密分享
		for _, piece := range pieces {
			if piece.IsValid() {
				//to do : 调用屮逸的发送函数
			}
		}
	}
	return
}

//收到组内成员发给我的秘密分享片段消息
func (p Processer) OnMessageSharePiece(spm ConsensusSharePieceMessage) {
	if p.gc == nil {
		fmt.Printf("OnMessageSharePiece failed, receive SHAREPIECE msg when gc=nil.\n")
		return
	}
	if !p.isSameGroup(spm.si.SignMember, false) {
		fmt.Printf("OnMessageSharePiece failed, not same group.\n")
		return
	}
	result := p.gc.PieceMessage(spm)
	if result == 1 { //已聚合出签名私钥
		pk := p.gc.GetPiecePubKey()
		if pk.IsValid() {
			//to do ：把公钥pk发送给组内所有成员
		}
	}
}

//收到组内成员发给我的组成员签名公钥消息
func (p Processer) OnMessagePubKeyPiece(ppm ConsensusPubKeyPieceMessage) {
	if p.gc == nil {
		fmt.Printf("OnMessagePubKeyPiece failed, receive PUBKEYPIECE msg when gc=nil.\n")
		return
	}
	if !p.isSameGroup(ppm.si.SignMember, false) {
		fmt.Printf("OnMessagePubKeyPiece failed, not same group.\n")
		return
	}
	result := p.gc.PiecePubKey(ppm)
	if result == 1 { //已经聚合出组公钥
		id, pk := p.gc.GetGroupInfo()
		if id.IsValid() && pk.IsValid() {
			//to do : 把已初始化的组信息广播到全网
		}
	}
}

//全网节点收到某组已初始化完成消息（在一个时间窗口内收到该组51%成员的消息相同，才确认上链）
//最终版本修改为父亲节点进行验证（51%）和上链
func (p Processer) OnMessageGroupInited(gim ConsensusGroupInitedMessage) {
	//g := p.gg.GetGroupByID(gim.gi.gis.DummyID)
	return
}

///////////////////////////////////////////////////////////////////////////////
//取得自身所在的组
func (p Processer) getSelfGroup() StaticGroupInfo {
	g, err := p.gg.GetGroupByID(p.gid)
	if err != nil {
		panic("GetSelfGroup failed.")
	}
	return g
}

//当前节点成为KING，出块
func (p Processer) castBlock(qn int64) bool {
	var bh BlockHeader
	var hash []byte
	//to do : 鸠兹生成bh和哈希
	//给鸠兹的参数：QN, nonce，castor
	var si SignData
	si.DataHash = common.BytesToHash(hash)
	si.SignMember = p.uid
	si.DataSign = groupsig.Sign(p.gusk, si.DataHash.Bytes()) //对区块头签名
	if bh.BlockHeight > 0 && si.DataSign.IsValid() {
		fmt.Printf("success cast block, height= %v, castor= %v.\n", bh.BlockHeight, bh.Castor.GetHexString())
	}
	//个人铸块完成的同时也是个人验证完成（第一个验证者）
	//更新共识上下文
	return true
}

//判断某个ID和当前节点是否同一组
//uid：远程节点ID，inited：组是否已初始化完成
func (p Processer) isSameGroup(uid groupsig.ID, inited bool) bool {
	if inited {
		return p.getSelfGroup().MemExist(uid)
	} else {
		return p.gc.MemExist(uid)
	}
}
