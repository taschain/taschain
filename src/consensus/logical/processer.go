package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/rand"
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

func (sci *SelfCastInfo) Init() {
	sci.block_qns = make(map[uint][]uint, 0)
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
	jgs JoiningGroups //已加入未完成初始化的组(组初始化完成上链后，不再需要)。组内成员数据过程数据。

	bcs map[string]*BlockContext //铸块上下文, 单个组。to do : 处理一个processer参与多个铸块组的情况
	gg  GlobalGroups             //全网组静态信息（包括待完成组初始化的组，即还没有组ID只有DUMMY ID）

	sci SelfCastInfo //当前节点的出块信息（包括当前节点在不同高度不同QN值所有成功和不成功的出块）。组内成员动态数据。
	//////和组无关的矿工信息
	mi MinerInfo
	//////加入(成功)的组信息(矿工节点数据)
	belongGroups map[string]SecKeyInfo //当前ID参与了哪些(已上链，可铸块的)组, 组id_str->(组id, 组成员签名私钥)。组内成员静态数据。
	//////测试数据，代替屮逸的网络消息
	GroupProcs map[string]*Processer

	piece_count  int
	inited_count int
}

func GetIDPrefix(id groupsig.ID) string {
	str := id.GetHexString()
	return str[0:6]
}

func (p Processer) getPrefix() string {
	return GetIDPrefix(p.GetMinerID())
}

//私密函数，用于测试，正式版本不提供
func (p Processer) getmi() MinerInfo {
	return p.mi
}

func (p Processer) GetMinerInfo() PubKeyInfo {
	var pki PubKeyInfo
	pki.id = p.mi.GetMinerID()
	pki.pk = p.mi.GetDefaultPubKey()
	return pki
}

func (p *Processer) setProcs(gps map[string]*Processer) {
	p.GroupProcs = gps
}

//初始化矿工数据（和组无关）
func (p *Processer) Init(mi MinerInfo) bool {
	p.mi = mi
	p.gg.Init()
	p.jgs.Init()
	p.belongGroups = make(map[string]SecKeyInfo, 0)
	p.bcs = make(map[string]*BlockContext, 0)
	p.sci.Init()
	fmt.Printf("proc(%v) inited.\n", p.getPrefix())
	return true
}

//预留接口
//后续如有全局定时器，从这个函数启动
func (p *Processer) Start() bool {
	return true
}

//预留接口
func (p *Processer) Stop() {
	return
}

func (p *Processer) AddBlockConext(bc *BlockContext) bool {
	if p.GetBlockContext(bc.MinerID.gid.GetHexString()) == nil {
		p.bcs[bc.MinerID.gid.GetHexString()] = bc
		return true
	} else {
		return false //已存在
	}
}

func (p *Processer) GetBlockContext(gid string) *BlockContext {
	if v, ok := p.bcs[gid]; ok {
		return v
	} else {
		return nil
	}
}

//取得矿工ID（和组无关）
func (p Processer) GetMinerID() groupsig.ID {
	return p.mi.MinerID
}

//取得矿工参与的所有铸块组私密私钥，正式版不提供
func (p Processer) getMinerGroups() map[string]SecKeyInfo {
	return p.belongGroups
}

//增加一个组签名私钥（一个矿工可能加入多个组）
//gid : 组ID(非dummy id)
//sk：用户的组成员签名私钥
func (p *Processer) addSignKey(gid groupsig.ID, sk groupsig.Seckey) {
	fmt.Printf("begin Processer::addSignKey...\n")
	if !p.IsMinerGroup(gid) {
		p.belongGroups[gid.GetHexString()] = SecKeyInfo{gid, sk}
	} else {
		panic("Processer::AddSignKey failed.")
	}
	fmt.Printf("SUCCESS:node=%v inited group=%v, sign key=%v.\n", p.GetMinerID().GetHexString(), gid.GetHexString(), sk.GetHexString())
	return
}

//取得矿工在某个组的签名私钥
func (p Processer) GetSignKey(gid groupsig.ID) groupsig.Seckey {
	return p.belongGroups[gid.GetHexString()].sk //如该组不存在则返回空值
}

//检测某个组是否矿工的铸块组（一个矿工可以参与多个组）
func (p Processer) IsMinerGroup(gid groupsig.ID) bool {
	_, ok := p.belongGroups[gid.GetHexString()]
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

//收到成为当前组消息
func (p *Processer) OnMessageCurrent(ccm ConsensusCurrentMessage) {
	fmt.Printf("begin Processer::OnMessageCurrent...\n")
	var gid groupsig.ID
	if gid.Deserialize(ccm.GroupID) != nil {
		panic("Processer::OnMessageCurrent failed, reason=group id Deserialize.")
	}
	if !p.IsMinerGroup(gid) {
		panic("Processer::OnMessageCurrent failed, node not in group.")
	}
	gi, err := p.gg.GetGroupByID(gid)
	if err == nil {
		ru, ok := gi.GetMember(ccm.si.SignMember) //检查发消息用户是否跟当前节点同组
		if ok {                                   //该用户和我是同一组
			fmt.Printf("message sender's id=%v, pub key=%v.\n", ru.id.GetHexString(), ru.pk.GetHexString())
			if ccm.si.VerifySign(ru.pk) { //消息合法
				fmt.Printf("message verify sign OK, call BeingCastGroup...\n")
				bc := p.GetBlockContext(gid.GetHexString())
				if bc == nil {
					panic("ERROR, BlockContext = nil.\n")
				}
				b := bc.BeingCastGroup(ccm.BlockHeight, ccm.PreTime, ccm.PreHash)
				fmt.Printf("blockContext::BeingCastGroup result=%v.\n", b)
				//to do : 屮逸组内广播
			} else {
				fmt.Printf("ERROR, message verify failed.\n")
				panic("ERROR, message verify failed.\n")
			}
		} else {
			fmt.Printf("message sender not same group, ignored.\n")
		}

	} else {
		panic("OnMessageCrrent failed, invalid gid.")
	}
	fmt.Printf("end Processer::OnMessageCurrent.\n")
	return
}

//收到组内成员的出块消息，出块人（KING）用组分片密钥进行了签名
func (p *Processer) OnMessageCast(ccm ConsensusCastMessage) {
	fmt.Printf("begin Processer::OnMessageCast, group=%v...\n", ccm.GroupID.GetHexString())
	bc := p.GetBlockContext(ccm.GroupID.GetHexString())
	if bc == nil {
		fmt.Printf("local joined groups=%v.\n", len(p.bcs))
		for k, _ := range p.bcs {
			fmt.Printf("---joined group:%v.\n", k)
		}
		panic("not found this group.")
	}
	if !bc.IsCasting() { //当前没有在组铸块中
		fmt.Printf("processer::OnMessageCast failed, group not in cast.\n")
		return
	}
	cs := GenConsensusSummary(ccm.bh)
	n := bc.UserCasted(cs, ccm.si)
	fmt.Printf("processer:OnMessageCast UserCasted result=%v.\n", n)
	if n == CBMR_THRESHOLD_SUCCESS {
		b := bc.VerifyGroupSign(cs, p.getGroupPubKey(ccm.GroupID))
		fmt.Printf("bc.VerifyGroupSign result=%v.\n", b)
		if b { //组签名验证通过
			p.SuccessNewBlock(cs, ccm.GroupID) //上链和组外广播
		}
	}
	fmt.Printf("end Processer::OnMessageCast.\n")
	return
}

//收到组内成员的出块验证通过消息（组内成员消息）
func (p *Processer) OnMessageVerify(cvm ConsensusVerifyMessage) {
	bc := p.GetBlockContext(cvm.GroupID.GetHexString())
	cs := GenConsensusSummary(cvm.bh)
	n := bc.UserVerified(cs, cvm.si)
	fmt.Printf("processer::OnMessageVerify UserVerified result=%v.\n", n)
	if n == CBMR_THRESHOLD_SUCCESS {
		b := bc.VerifyGroupSign(cs, p.getGroupPubKey(cvm.GroupID))
		if b { //组签名验证通过
			p.SuccessNewBlock(cs, cvm.GroupID) //上链和组外广播
		}
	}
	return
}

//收到铸块上链消息(组外矿工节点处理)
func (p *Processer) OnMessageBlock(cbm ConsensusBlockMessage) {
	bc := p.GetBlockContext(cbm.GroupID.GetHexString())
	if p.isBHCastLegal(cbm.bh, cbm.si) { //铸块头合法
		//to do : 鸠兹上链保存
		next_group, err := p.gg.SelectNextGroup(cbm.si.DataHash) //查找下一个铸块组
		if err == nil {
			if p.IsMinerGroup(next_group) { //自身属于下一个铸块组
				bc.BeingCastGroup(cbm.bh.BlockHeight, cbm.bh.PreTime, cbm.si.DataHash)
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

//在某个区块高度的QN值成功出块，保存上链，向组外广播
//同一个高度，可能会因QN不同而多次调用该函数
//但一旦低的QN出过，就不该出高的QN。即该函数可能被多次调用，但是调用的QN值越来越小
func (p *Processer) SuccessNewBlock(cs ConsensusBlockSummary, gid groupsig.ID) {
	bc := p.GetBlockContext(gid.GetHexString())
	//to do : 鸠兹保存上链
	//to do : 屮逸组外广播
	bc.CastedUpdateStatus(uint(cs.QueueNumber))
	bc.SignedUpdateMinQN(uint(cs.QueueNumber))
	return
}

//检查是否轮到自己出块
func (p *Processer) CheckCastRoutine(gid groupsig.ID, user_index int32, qn int64, height uint) {
	fmt.Printf("begin Processer::CheckCastRoutine, gid=%v, king_index=%v, qn=%v, height=%v.\n", gid.GetHexString(), user_index, qn, height)
	if user_index < 0 || qn < 0 {
		return
	}
	sgi := p.getGroup(gid)
	pos := sgi.GetMinerPos(p.GetMinerID())
	fmt.Printf("Current KING=%v.\n", sgi.GetCastor(int(user_index)).GetHexString())
	fmt.Printf("Current node=%v, index=%v.\n", p.GetMinerID().GetHexString(), pos)
	if sgi.GetCastor(int(user_index)).GetHexString() == p.GetMinerID().GetHexString() { //轮到自己铸块
		fmt.Printf("curent node IS KING!.\n")
		if p.sci.AddQN(height, uint(qn)) { //在该高度该QN，自己还没铸过快
			p.castBlock(gid, qn) //铸块
		} else {
			fmt.Printf("In height=%v, qn=%v current node already casted.", height, qn)
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
				dest_proc, ok := p.GroupProcs[spm.Dest.GetHexString()]
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
			p.addSignKey(g.id, sk)
			fmt.Printf("SUCCESS INIT GROUP: group_id=%v, pub_key=%v.\n", g.id.GetHexString(), g.pk.GetHexString())
			//to do : 把初始化完成的组加入到gc（更新）
			//to do : 把组初始化完成消息广播到全网
			{
				var msg ConsensusGroupInitedMessage
				ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
				msg.gi.gis = gc.gis
				msg.gi.GroupID = g.id
				msg.gi.GroupPK = g.pk
				var mems []PubKeyInfo
				for _, v := range gc.mems {
					mems = append(mems, v)
				}
				msg.gi.members = mems
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
	fmt.Printf("proc(%v)bein Processer::OnMessageGroupInited, sender=%v...\n", p.getPrefix(), GetIDPrefix(gim.si.SignMember))
	var ngmd NewGroupMemberData
	ngmd.h = gim.gi.gis.GenHash()
	ngmd.gid = gim.gi.GroupID
	ngmd.gpk = gim.gi.GroupPK
	var mid GroupMinerID
	mid.gid = gim.gi.gis.DummyID
	mid.uid = gim.si.SignMember
	result := p.gg.GroupInitedMessage(mid, ngmd)
	p.inited_count++
	fmt.Printf("proc(%v)gg.GroupInitedMessage return=%v, inited_count=%v.\n", p.getPrefix(), result, p.inited_count)
	if result < 0 {
		panic("gg.GroupInitedMessage failed because of return value.")
	}
	switch result {
	case 1: //收到组内相同消息>=阈值，可上链
		b := p.gg.AddGroup(gim.gi)
		fmt.Printf("Add to Global static groups, result=%v, groups=%v.\n", b, p.gg.GetGroupSize())
		bc := new(BlockContext)
		bc.Init(GroupMinerID{gim.gi.GroupID, p.GetMinerID()})
		sgi, err := p.gg.GetGroupByID(gim.gi.GroupID)
		if err != nil {
			panic("GetGroupByID failed.\n")
		}
		bc.pos = sgi.GetMinerPos(p.GetMinerID())
		fmt.Printf("current node in group pos=%v.\n", bc.pos)
		bc.Proc = p
		b = p.AddBlockConext(bc)
		fmt.Printf("(proc:%v) Add BlockContext result = %v, bc_size=%v.\n", p.getPrefix(), b, len(p.bcs))
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

func genDummyBH(qn int, uid groupsig.ID) *core.BlockHeader {
	bh := new(core.BlockHeader)
	bh.Hash = rand.String2CommonHash("thiefox")
	bh.Height = 2
	bh.PreHash = rand.String2CommonHash("TASchain")
	bh.BlockHeight = bh.Height
	bh.QueueNumber = uint64(qn)
	bh.CurTime = time.Now()
	bh.Nonce = 123
	bh.Castor = uid.Serialize()
	//bh.Signature
	return bh
}

//当前节点成为KING，出块
func (p Processer) castBlock(gid groupsig.ID, qn int64) *core.BlockHeader {
	fmt.Printf("begin Processer::castBlock...\n")
	bh := genDummyBH(int(qn), p.GetMinerID())
	var hash common.Hash
	hash = bh.Hash //TO DO:替换成出块头的哈希
	//to do : 鸠兹生成bh和哈希
	//给鸠兹的参数：QN, nonce，castor
	var si SignData
	si.DataHash = hash
	si.SignMember = p.GetMinerID()
	b := si.GenSign(p.GetSignKey(gid)) //组成员签名私钥片段签名
	fmt.Printf("castor sign bh result = %v.\n", b)

	if bh.BlockHeight > 0 && si.DataSign.IsValid() {
		var tmp_id groupsig.ID
		if tmp_id.Deserialize(bh.Castor) != nil {
			panic("ID Deserialize failed.")
		}
		fmt.Printf("success cast block, height= %v, castor= %v.\n", bh.BlockHeight, tmp_id.GetHexString())
		//发送该出块消息
		var ccm ConsensusCastMessage
		ccm.bh = *bh
		ccm.GroupID = gid
		ccm.si = si
		for _, proc := range p.GroupProcs {
			proc.OnMessageCast(ccm)
		}

	} else {
		fmt.Printf("bh Error or sign Error, bh=%v, ds=%v.\n", bh.BlockHeight, si.DataSign.GetHexString())
		panic("bh Error or sign Error.")
	}
	//个人铸块完成的同时也是个人验证完成（第一个验证者）
	//更新共识上下文
	fmt.Printf("end Processer::castBlock.\n")
	return bh
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
