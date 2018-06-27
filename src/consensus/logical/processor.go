package logical

import (
	"consensus/groupsig"

	"core"
	"time"
	"log"
	"consensus/ticker"
	"middleware/types"
	"vm/ethdb"
	"core/datasource"
)

var PROC_TEST_MODE bool

//见证人处理器
type Processor struct {
	joiningGroups		*JoiningGroups //已加入未完成初始化的组(组初始化完成上链后，不再需要)。组内成员数据过程数据。

	blockContexts		*CastBlockContexts //组ID->组铸块上下文
	globalGroups         *GlobalGroups             //全网组静态信息（包括待完成组初始化的组，即还没有组ID只有DUMMY ID的组）

	groupManager 		*GroupManager

	//////和组无关的矿工信息
	mi *MinerInfo
	//////加入(成功)的组信息(矿工节点数据)
	belongGroups *BelongGroups //当前ID参与了哪些(已上链，可铸块的)组, 组id_str->组内私密数据（组外不可见或加速缓存）
	//////测试数据，代替屮逸的网络消息
	GroupProcs map[string]*Processor
	Ticker 			*ticker.GlobalTicker		//全局定时器, 组初始化完成后启动

	futureBlockMsgs 	*FutureMessageHolder //存储缺少父块的块
	futureVerifyMsgs 	*FutureMessageHolder //存储缺失前一块的验证消息

	storage 	ethdb.Database
	ready 		bool //是否已初始化完成

	//////链接口
	MainChain  core.BlockChainI
	GroupChain *core.GroupChain

}


func (p Processor) getPrefix() string {
	return GetIDPrefix(p.GetMinerID())
}

//私密函数，用于测试，正式版本不提供
func (p Processor) getMinerInfo() *MinerInfo {
	return p.mi
}

func (p Processor) GetMinerInfo() PubKeyInfo {
	return PubKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultPubKey()}
}

func (p *Processor) setProcs(gps map[string]*Processor) {
	p.GroupProcs = gps
}


//初始化矿工数据（和组无关）
func (p *Processor) Init(mi MinerInfo) bool {
	p.ready = false
	p.futureBlockMsgs = NewFutureMessageHolder()
	p.futureVerifyMsgs = NewFutureMessageHolder()
	p.MainChain = core.BlockChainImpl
	p.GroupChain = core.GroupChainImpl
	p.mi = &mi
	p.globalGroups = NewGlobalGroups()
	p.joiningGroups = NewJoiningGroups()
	p.belongGroups = NewBelongGroups()
	p.blockContexts = NewCastBlockContexts()
	p.groupManager = NewGroupManager(p)

	db, err := datasource.NewDatabase(STORE_PREFIX)
	if err != nil {
		log.Printf("NewDatabase error %v\n", err)
		return false
	}
	p.storage = db
	//p.sci.Init()

	p.Ticker = ticker.NewGlobalTicker(p.getPrefix())

	log.Printf("proc(%v) inited 2.\n", p.getPrefix())
	consensusLogger.Infof("ProcessorId:%v", p.getPrefix())
	return true
}


//取得矿工ID（和组无关）
func (p Processor) GetMinerID() groupsig.ID {
	return p.mi.MinerID
}

//验证块的组签名是否正确
func (p *Processor) verifyGroupSign(b *types.Block, sd SignData) bool {
	bh := b.Header
	var gid groupsig.ID
	if gid.Deserialize(bh.GroupId) != nil {
		panic("verifyGroupSign: group id Deserialize failed.")
	}

	groupInfo := p.getGroup(gid)
	if !groupInfo.GroupID.IsValid() {
		log.Printf("verifyGroupSign: get group is nil!, gid=" + GetIDPrefix(gid))
		return false
	}

	//检查组签名是否正确
	var gSign groupsig.Signature
	if gSign.Deserialize(bh.Signature) != nil {
		panic("verifyGroupSign sign Deserialize failed.")
	}
	result := groupsig.VerifySig(groupInfo.GroupPK, bh.Hash.Bytes(), gSign)
	if !result {
		log.Printf("[ERROR]verifyGroupSign::verify group sign failed, gpk=%v, hash=%v, sign=%v. gid=%v.\n",
			GetPubKeyPrefix(groupInfo.GroupPK), GetHashPrefix(bh.Hash), GetSignPrefix(gSign), GetIDPrefix(gid))
	}
	//to do ：对铸块的矿工（组内最终铸块者，非KING）签名做验证
	return result
}

//检查铸块组是否合法
func (p *Processor) isCastGroupLegal(bh *types.BlockHeader, preHeader *types.BlockHeader) (result bool) {
	//to do : 检查是否基于链上最高块的出块
	defer func() {
		log.Printf("isCastGroupLeagal result=%v.\n", result)
	}()
	var gid groupsig.ID
	if gid.Deserialize(bh.GroupId) != nil {
		panic("isCastGroupLeagal, group id Deserialize failed.")
	}

	selectGroupId := p.calcCastGroup(preHeader, bh.Height)
	if selectGroupId == nil {
		return false
	}

	groupInfo, err := p.globalGroups.GetGroupByID(*selectGroupId) //取得合法的铸块组
	if err != nil {
		log.Printf("isCastGroupLeagal: getGroupById error, id=%v, err=%v", GetIDPrefix(*selectGroupId), err)
		return false
	}

	if groupInfo.GroupID.IsEqual(gid) {
		return true
	} else {
		log.Printf("isCastGroupLeagal failed, expect group=%v, real cast group=%v.\n", GetIDPrefix(groupInfo.GroupID), GetIDPrefix(gid))
		//panic("isBHCastLegal failed, not expect group")  非法铸块组 直接跳过就行了吧?
	}

	return result
}

//生成创世组成员信息
func (p *Processor) BeginGenesisGroupMember() PubKeyInfo {
	gis := p.GenGenesisGroupSummary()
	temp_mi := p.getMinerInfo()
	temp_mgs := NewMinerGroupSecret(temp_mi.GenSecretForGroup(gis.GenHash()))
	gsk_piece := temp_mgs.GenSecKey()
	gpk_piece := *groupsig.NewPubkeyFromSeckey(gsk_piece)
	pki := PubKeyInfo{p.GetMinerID(), gpk_piece}
	log.Printf("\nBegin Genesis Group Member, ID=%v, gpk_piece=%v.\n", GetIDPrefix(pki.GetID()), GetPubKeyPrefix(pki.PK))
	return pki
}

func (p *Processor) GenGenesisGroupSummary() ConsensusGroupInitSummary {
	var gis ConsensusGroupInitSummary
	//gis.ParentID = P.GetMinerID()
	gis.DummyID = *groupsig.NewIDFromString("Trust Among Strangers")
	gis.Authority = 777
	gn := "TAS genesis group"
	if len(gn) <= 64 {
		copy(gis.Name[:], gn[:])
	} else {
		copy(gis.Name[:], gn[:64])
	}
	//gis.BeginTime = time.Date(2018, time.May, 4, 18, 00, 00, 00, time.Local)
	unix_time := time.Now().Unix()
	unix_time = unix_time - 100
	gis.BeginTime = time.Unix(unix_time, 0)
	gis.Extends = "room 1003, BLWJXXJS6KYHX"
	gis.Members = uint64(GROUP_MAX_MEMBERS)
	return gis
}

//创建一个新建组。由（且有创建组权限的）父亲组节点发起。
//miners：待成组的矿工信息。ID，（和组无关的）矿工公钥。
//gn：组名。
func (p *Processor) CreateDummyGroup(miners []PubKeyInfo, parentId *groupsig.ID, gn string) int {
	if len(miners) != GROUP_MAX_MEMBERS {
		log.Printf("create group error, group max members=%v, real=%v.\n", GROUP_MAX_MEMBERS, len(miners))
		return -1
	}
	var gis ConsensusGroupInitSummary
	//gis.ParentID = p.GetMinerID()

	//todo future bug
	if parentId == nil {
		parentId = groupsig.DeserializeId([]byte("genesis group dummy"))
	}

	gis.ParentID = *parentId
	gis.DummyID = *groupsig.NewIDFromString(gn)

	if p.GroupChain.GetGroupById(gis.DummyID.Serialize()) != nil {
		log.Printf("CreateDummyGroup ingored, dummyId already onchain! dummyId=%v\n", GetIDPrefix(gis.DummyID))
		return 0
	}

	log.Printf("create group, group name=%v, group dummy id=%v.\n", gn, GetIDPrefix(gis.DummyID))
	gis.Authority = 777
	if len(gn) <= 64 {
		copy(gis.Name[:], gn[:])
	} else {
		copy(gis.Name[:], gn[:64])
	}
	gis.BeginTime = time.Now()
	if !gis.ParentID.IsValid() || !gis.DummyID.IsValid() {
		panic("create group init summary failed")
	}
	gis.Members = uint64(GROUP_MAX_MEMBERS)
	gis.Extends = "Dummy"
	var grm ConsensusGroupRawMessage
	grm.MEMS = make([]PubKeyInfo, len(miners))
	copy(grm.MEMS[:], miners[:])
	gis.MemberHash = genMemberHash(grm.MEMS)
	grm.GI = gis
	grm.SI = GenSignData(grm.GI.GenHash(), p.GetMinerID(), p.getMinerInfo().GetDefaultSecKey())
	log.Printf("proc(%v) Create New Group, send network msg to members...\n", p.getPrefix())
	log.Printf("call network service SendGroupInitMessage...\n")

	SendGroupInitMessage(grm)
	return 0
}

//检测是否激活成为当前铸块组，成功激活返回有效的bc，激活失败返回nil
func (p *Processor) verifyCastSign(cgs *CastGroupSummary, si *SignData) bool {

	if !p.IsMinerGroup(cgs.GroupID) { //检测当前节点是否在该铸块组
		log.Printf("beingCastGroup failed, node not in this group.\n")
		return false
	}
	_, err := p.globalGroups.GetGroupByID(cgs.GroupID)
	if err != nil {
		panic("globalGroups.GetGroupByID failed.")
	}
	gmi := GroupMinerID{cgs.GroupID, si.GetID()}
	signPk := p.GetMemberSignPubKey(gmi) //取得消息发送方的组内签名公钥

	if signPk.IsValid() { //该用户和我是同一组
		//log.Printf("message sender's signPk=%v.\n", GetPubKeyPrefix(signPk))
		//log.Printf("verifyCast::si info: id=%v, data hash=%v, sign=%v.\n",
		//	GetIDPrefix(si.GetID()), GetHashPrefix(si.DataHash), GetSignPrefix(si.DataSign))
		if si.VerifySign(signPk) { //消息合法
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

func (p *Processor) getMinerPos(gid groupsig.ID, uid groupsig.ID) int32 {
	sgi := p.getGroup(gid)
	return int32(sgi.GetMinerPos(uid))
}

//检查是否轮到自己出块
func (p *Processor) kingCheckAndCast(bc *BlockContext, vctx *VerifyContext, kingIndex int32, qn int64) {
	//p.castLock.Lock()
	//defer p.castLock.Unlock()
	gid := bc.MinerID.gid
	height := vctx.castHeight

	log.Printf("prov(%v) begin kingCheckAndCast, gid=%v, kingIndex=%v, qn=%v, height=%v.\n", p.getPrefix(), GetIDPrefix(gid), kingIndex, qn, height)
	if kingIndex < 0 || qn < 0 {
		return
	}

	sgi := p.getGroup(gid)

	log.Printf("time=%v, Current KING=%v.\n", time.Now().Format(time.Stamp), GetIDPrefix(sgi.GetCastor(int(kingIndex))))
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

///////////////////////////////////////////////////////////////////////////////
//取得自己参与的某个铸块组的公钥片段（聚合一个组所有成员的公钥片段，可以生成组公钥）
func (p Processor) GetMinerPubKeyPieceForGroup(gid groupsig.ID) groupsig.Pubkey {
	var pub_piece groupsig.Pubkey
	gc := p.joiningGroups.GetGroup(gid)
	node := gc.GetNode()
	if node != nil {
		pub_piece = node.GetSeedPubKey()
	}
	return pub_piece
}

//取得自己参与的某个铸块组的私钥片段（聚合一个组所有成员的私钥片段，可以生成组私钥）
//用于测试目的，正式版对外不提供。
func (p Processor) getMinerSecKeyPieceForGroup(gid groupsig.ID) groupsig.Seckey {
	var secPiece groupsig.Seckey
	gc := p.joiningGroups.GetGroup(gid)
	node := gc.GetNode()
	if node != nil {
		secPiece = node.getSeedSecKey()
	}
	return secPiece
}

//取得特定的组
func (p Processor) getGroup(gid groupsig.ID) *StaticGroupInfo {
	g, err := p.globalGroups.GetGroupByID(gid)
	if err != nil {
		panic("GetSelfGroup failed.")
	}
	return g
}

//取得一个铸块组的公钥(processer初始化时从链上加载)
func (p Processor) getGroupPubKey(gid groupsig.ID) groupsig.Pubkey {
	g, err := p.globalGroups.GetGroupByID(gid)
	if err != nil {
		panic("GetSelfGroup failed.")
	}
	return g.GetPubKey()
}

func outputBlockHeaderAndSign(prefix string, bh *types.BlockHeader, si *SignData) {
	//bbyte, _ := bh.CurTime.MarshalBinary()
	//jbyte, _ := bh.CurTime.MarshalJSON()
	//textbyte, _ := bh.CurTime.MarshalText()
	//log.Printf("%v, bh.curTime %v, byte=%v, jsonByte=%v, textByte=%v, nano=%v, utc=%v, local=%v, location=%v\n", prefix, bh.CurTime, bbyte, jbyte, textbyte, bh.CurTime.UnixNano(), bh.CurTime.UTC(), bh.CurTime.Local(), bh.CurTime.Location().String())

	//var castor groupsig.ID
	//castor.Deserialize(bh.Castor)
	//txs := ""
	//if bh.Transactions != nil {
	//	for _, tx := range bh.Transactions {
	//		txs += GetHashPrefix(tx) + ","
	//	}
	//}
	//txs = "[" + txs + "]"
	//log.Printf("%v, BLOCKINFO: height= %v, castor=%v, hash=%v, txs=%v, txtree=%v, statetree=%v, receipttree=%v\n", prefix, bh.Height, GetIDPrefix(castor), GetHashPrefix(bh.Hash), txs, GetHashPrefix(bh.TxTree), GetHashPrefix(bh.StateTree), GetHashPrefix(bh.ReceiptTree))
	//
	//if si != nil {
	//	log.Printf("%v, SIDATA: datahash=%v, sign=%v, signer=%v\n", prefix, GetHashPrefix(si.DataHash), si.DataSign.GetHexString(), GetIDPrefix(si.SignMember))
	//}
}


//判断某个ID和当前节点是否同一组
//uid：远程节点ID，inited：组是否已初始化完成
func (p Processor) isSameGroup(gid groupsig.ID, uid groupsig.ID, inited bool) bool {
	if inited {
		return p.getGroup(gid).MemExist(uid) && p.getGroup(gid).MemExist(p.GetMinerID())
	} else {
		//return p.gc.MemExist(uid)
		//to do : 增加判断
		return false
	}
}

func (p *Processor) ExistInDummyGroup(dummyId groupsig.ID) bool {
	initingGroup := p.globalGroups.GetInitingGroup(dummyId)
	if initingGroup == nil {
		return false
	}
    return initingGroup.MemberExist(p.GetMinerID())
}
