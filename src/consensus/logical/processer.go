package logical

import (
	"common"
	"consensus/groupsig"
	"core"
	"fmt"
	"time"
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
	jgs JoiningGroups //已加入未完成初始化的组(组初始化完成上链后，不再需要)
	bc  BlockContext  //铸块上下文
	gg  GlobalGroups  //全网组静态信息

	sci SelfCastInfo //当前节点的铸块信息（包括当前节点在不同高度不同QN值所有成功和不成功的出块）
	//////和组无关的矿工信息
	mi MinerInfo
	//////加入(成功)的组信息(矿工节点数据)
	BelongGroups map[groupsig.ID]groupsig.Seckey //当前ID参与了哪些(已上链，可铸块的)组, id->组成员签名私钥
	//////测试数据，代替屮逸的网络消息
	GroupProcs map[groupsig.ID]*Processer

	piece_count  int
	inited_count int
}

func (p *Processer) setProcs(gps map[groupsig.ID]*Processer) {
	p.GroupProcs = gps
}

//初始化矿工数据（和组无关）
func (p *Processer) Init(mi MinerInfo) bool {
	p.mi = mi
	p.jgs.Init()
	p.gg.ngg.Init()
	return true
}

//取得矿工ID（和组无关）
func (p Processer) GetMinerID() groupsig.ID {
	return p.mi.MinerID
}

//增加一个组签名私钥（一个矿工可能加入多个组）
//gid : 组ID(非dummy id)
//sk：用户的组成员签名私钥
func (p *Processer) addSignKey(gid groupsig.ID, sk groupsig.Seckey) {
	fmt.Printf("begin Processer::addSignKey...\n")
	if !p.IsMinerGroup(gid) {
		if p.BelongGroups == nil {
			p.BelongGroups = make(map[groupsig.ID]groupsig.Seckey, 1)
		}
		p.BelongGroups[gid] = sk
	} else {
		panic("Processer::AddSignKey failed.")
	}
	fmt.Printf("SUCCESS:node=%v inited group=%v, sign key=%v.\n", p.GetMinerID().GetHexString(), gid.GetHexString(), sk.GetHexString())
	return
}

//取得矿工在某个组的签名私钥
func (p Processer) GetSignKey(gid groupsig.ID) groupsig.Seckey {
	return p.BelongGroups[gid] //如该组不存在则返回空值
}

//检测某个组是否矿工的铸块组（一个矿工可以参与多个组）
func (p Processer) IsMinerGroup(gid groupsig.ID) bool {
	_, ok := p.BelongGroups[gid]
	return ok
}

//检查区块头是否合法
func (p Processer) isBHCastLegal(bh core.BlockHeader, sd SignData) (result bool) {
	//to do : 检查是否基于链上最高块的出块
	gi := p.gg.GetCastGroup(bh.PreHash) //取得合法的铸块组
	if gi.GroupID == sd.SignMember {
		//检查组签名是否正确
		result = sd.VerifySign(gi.GroupPK)
	}
	return result
}

//收到组内成员的出块消息，铸块人用组分片密钥进行了签名
func (p *Processer) OnMessageCast(ccm ConsensusCastMessage) {
	if !p.bc.IsCasting() { //当前没有在组铸块中
		fmt.Printf("processer::OnMessageCast failed, group not in cast.\n")
		return
	}
	cs := GenConsensusSummary(ccm.bh)
	n := p.bc.UserCasted(cs, ccm.si)
	fmt.Printf("processer:OnMessageCast UserCasted result=%v.\n", n)
	if n == CBMR_THRESHOLD_SUCCESS {
		b := p.bc.VerifyGroupSign(cs, p.getGroupPubKey(ccm.GroupID))
		if b { //组签名验证通过
			p.SuccessNewBlock(cs) //上链和组外广播
		}
	}
	return
}

func (p *Processer) OnMessageVerify(cvm ConsensusVerifyMessage) {
	cs := GenConsensusSummary(cvm.bh)
	n := p.bc.UserVerified(cs, cvm.si)
	fmt.Printf("processer::OnMessageVerify UserVerified result=%v.\n", n)
	if n == CBMR_THRESHOLD_SUCCESS {
		b := p.bc.VerifyGroupSign(cs, p.getGroupPubKey(cvm.GroupID))
		if b { //组签名验证通过
			p.SuccessNewBlock(cs) //上链和组外广播
		}
	}
	return
}

//收到铸块上链消息
func (p *Processer) OnMessageBlock(cbm ConsensusBlockMessage) {
	if p.isBHCastLegal(cbm.bh, cbm.si) { //铸块头合法
		//to do : 鸠兹上链保存
		next_group, err := p.gg.SelectNextGroup(cbm.si.DataHash) //查找下一个铸块组
		if err == nil {
			if p.IsMinerGroup(next_group) { //自身属于下一个铸块组
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
func (p *Processer) OnMessageCurrent(ccm ConsensusCurrentMessage) {
	if !p.IsMinerGroup(ccm.GroupID) {
		panic("Processer::OnMessageCurrent failed, node not in group.")
	}
	gi, err := p.gg.GetGroupByID(ccm.GroupID)
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
func (p *Processer) SuccessNewBlock(cs ConsensusBlockSummary) {
	//to do : 鸠兹保存上链
	//to do : 屮逸组外广播
	p.bc.CastedUpdateStatus(uint(cs.QueueNumber))
	p.bc.SignedUpdateMinQN(uint(cs.QueueNumber))
	return
}

//检查是否轮到自己出块
func (p *Processer) CheckCastRoutine(gid groupsig.ID, user_index int32, qn int64, height uint) {
	if user_index < 0 || qn < 0 {
		return
	}

	if p.getGroup(gid).GetCastor(int(user_index)) == p.GetMinerID() { //轮到自己铸块
		if p.sci.AddQN(height, uint(qn)) { //在该高度该QN，自己还没铸过快
			p.castBlock(gid, qn) //铸块
		}
	}
	return
}

///////////////////////////////////////////////////////////////////////////////
//组初始化相关消息
func (p *Processer) OnMessageGroupInit(grm ConsensusGroupRawMessage) {
	fmt.Printf("begin Processer::OnMessageGroupInit, procs=%v...\n", len(p.GroupProcs))
	//to do : 从链上检查消息发起人（父亲组成员）是否有权限发该消息（鸠兹）

	gc := p.jgs.ConfirmGroupFromRaw(grm, p.mi)
	if gc == nil {
		panic("Processer::OnMessageGroupInit failed, CreateGroupContextWithMessage return nil.")
	}
	gs := gc.GetGroupStatus()
	fmt.Printf("joining group(%v) status=%v.\n", grm.gi.DummyID.GetHexString(), gs)
	if gs == GIS_RAW {
		fmt.Printf("begin GenSharePieces...\n")
		shares := gc.GenSharePieces() //生成秘密分享
		fmt.Printf("node(%v) end GenSharePieces, piece size=%v.\n", p.GetMinerID().GetHexString(), len(shares))
		var spm ConsensusSharePieceMessage
		spm.GISHash = grm.gi.GenHash()
		spm.DummyID = grm.gi.DummyID
		ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
		spm.si.SignMember = p.GetMinerID()
		for id, piece := range shares {
			if id.IsValid() && piece.IsValid() {
				spm.Dest = id
				spm.share = piece
				sb := spm.GenSign(ski)
				fmt.Printf("spm.GenSign result=%v.\n", sb)
				fmt.Printf("piece to ID(%v), share=%v, pub=%v.\n", spm.Dest.GetHexString(), spm.share.share.GetHexString(), spm.share.pub.GetHexString())
				//to do : 调用屮逸的发送函数
				dest_proc, ok := p.GroupProcs[spm.Dest]
				if ok {
					dest_proc.OnMessageSharePiece(spm)
				} else {
					panic("ERROR, dest proc not found!\n")
				}
			} else {
				panic("GenSharePieces data not IsValid.\n")
			}
		}
		fmt.Printf("end GenSharePieces.\n")
	} else {
		fmt.Printf("group(%v) status=%v, ignore init message.\n", grm.gi.DummyID.GetHexString(), gs)
	}
	fmt.Printf("end Processer::OnMessageGroupInit.\n")
	return
}

//收到组内成员发给我的秘密分享片段消息
func (p *Processer) OnMessageSharePiece(spm ConsensusSharePieceMessage) {
	gc := p.jgs.ConfirmGroupFromPiece(spm, p.mi)
	if gc == nil {
		panic("OnMessageSharePiece failed, receive SHAREPIECE msg but gc=nil.\n")
		return
	}
	result := gc.PieceMessage(spm)
	fmt.Printf("node(%v)begin Processer::OnMessageSharePiece, piecc_count=%v, gc result=%v.\n", p.GetMinerID().GetHexString(), p.piece_count, result)
	p.piece_count++
	if result < 0 {
		panic("OnMessageSharePiece failed, gc.PieceMessage result=%v.\n")
	}
	if result == 1 { //已聚合出签名私钥
		g, sk := gc.GetGroupInfo()
		if g.IsValid() && sk.IsValid() {
			p.addSignKey(g.Id, sk)
			fmt.Printf("SUCCESS INIT GROUP: group_id=%v, pub_key=%v.\n", g.Id.GetHexString(), g.pk.GetHexString())
			//to do : 把初始化完成的组加入到gc（更新）
			//to do : 把组初始化完成消息广播到全网
			{
				var msg ConsensusGroupInitedMessage
				ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
				msg.Gi.gis = gc.gis
				msg.Gi.GroupID = g.Id
				msg.Gi.GroupPK = g.pk
				var pki PubKeyInfo
				var mems []PubKeyInfo
				for _, id := range gc.ids {
					pki.Id = id
					mems = append(mems, pki)
				}
				msg.Gi.Members = mems
				msg.GenSign(ski)
				for _, proc := range p.GroupProcs {
					proc.OnMessageGroupInited(msg)
				}

			}

		} else {
			panic("Processer::OnMessageSharePiece failed, aggr key error.")
		}
	}
	fmt.Printf("end Processer::OnMessageSharePiece.\n")
	return
}

//全网节点收到某组已初始化完成消息（在一个时间窗口内收到该组51%成员的消息相同，才确认上链）
//最终版本修改为父亲节点进行验证（51%）和上链
//全网节点处理函数->to do : 调整为父亲组节点处理函数
func (p *Processer) OnMessageGroupInited(gim ConsensusGroupInitedMessage) {
	fmt.Printf("node(%v)bein Processer::OnMessageGroupInited...\n", p.mi.GetMinerID().GetHexString())
	var ngmd NewGroupMemberData
	ngmd.h = gim.Gi.gis.GenHash()
	ngmd.gid = gim.Gi.GroupID
	ngmd.gpk = gim.Gi.GroupPK
	var mid GroupMinerID
	mid.gid = gim.Gi.gis.DummyID
	mid.uid = gim.si.SignMember
	result := p.gg.GroupInitedMessage(mid, ngmd)
	p.inited_count++
	fmt.Printf("gg.GroupInitedMessage return=%v, inited_count=%v.\n", result, p.inited_count)
	if result < 0 {
		panic("gg.GroupInitedMessage failed because of return value.")
	}
	switch result {
	case 1: //收到组内相同消息>阈值，可上链
		//to do : 上链已初始化的组
		//to do ：从待初始化组中删除
		//to do : 是否全网广播该组的生成？广播的意义？
	case -1: //该组初始化异常，且无法恢复
		//to do : 从待初始化组中删除
	case 0:
		//继续等待下一包数据
	}
	fmt.Printf("end Processer::OnMessageGroupInited.\n")
	return
}

///////////////////////////////////////////////////////////////////////////////
//取得自己参与的某个铸块组的公钥片段（聚合一个组所有成员的公钥片段，可以生成组公钥）
func (p Processer) GetMinerPubKeyPieceForGroup(gid groupsig.ID) groupsig.Pubkey {
	var pub_piece groupsig.Pubkey
	gc := p.jgs.GetGroup(gid)
	node := gc.GetNode()
	if node != nil {
		pub_piece = node.GetSeedPubKey()
	}
	return pub_piece
}

//取得自己参与的某个铸块组的私钥片段（聚合一个组所有成员的私钥片段，可以生成组私钥）
//用于测试目的，正式版对外不提供。
func (p Processer) getMinerSecKeyPieceForGroup(gid groupsig.ID) groupsig.Seckey {
	var sec_piece groupsig.Seckey
	gc := p.jgs.GetGroup(gid)
	node := gc.GetNode()
	if node != nil {
		sec_piece = node.getSeedSecKey()
	}
	return sec_piece
}

//取得特定的组
func (p Processer) getGroup(gid groupsig.ID) StaticGroupInfo {
	g, err := p.gg.GetGroupByID(gid)
	if err != nil {
		panic("GetSelfGroup failed.")
	}
	return g
}

//取得一个铸块组的公钥(processer初始化时从链上加载)
func (p Processer) getGroupPubKey(gid groupsig.ID) groupsig.Pubkey {
	g, err := p.gg.GetGroupByID(gid)
	if err != nil {
		panic("GetSelfGroup failed.")
	}
	return g.GetPubKey()
}

//当前节点成为KING，出块
func (p Processer) castBlock(gid groupsig.ID, qn int64) bool {
	var bh core.BlockHeader
	var hash []byte
	//to do : 鸠兹生成bh和哈希
	//给鸠兹的参数：QN, nonce，castor
	var si SignData
	si.DataHash = common.BytesToHash(hash)
	si.SignMember = p.GetMinerID()
	si.DataSign = groupsig.Sign(p.GetSignKey(gid), si.DataHash.Bytes()) //对区块头签名
	if bh.BlockHeight > 0 && si.DataSign.IsValid() {
		var tmp_id groupsig.ID
		if tmp_id.Deserialize(bh.Castor) != nil {
			panic("ID Deserialize failed.")
		}
		fmt.Printf("success cast block, height= %v, castor= %v.\n", bh.BlockHeight, tmp_id.GetHexString())
	}
	//个人铸块完成的同时也是个人验证完成（第一个验证者）
	//更新共识上下文
	return true
}

//判断某个ID和当前节点是否同一组
//uid：远程节点ID，inited：组是否已初始化完成
func (p Processer) isSameGroup(gid groupsig.ID, uid groupsig.ID, inited bool) bool {
	if inited {
		return p.getGroup(gid).MemExist(uid) && p.getGroup(gid).MemExist(p.GetMinerID())
	} else {
		//return p.gc.MemExist(uid)
		//to do : 增加判断
		return false
	}
}
