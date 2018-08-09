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
	"sync"
	//"consensus/net"
	"consensus/rand"
	"core"
	"time"
	"log"
	"consensus/ticker"
	"middleware/types"
	"storage/tasdb"
	"core/datasource"
)

var PROC_TEST_MODE bool

//见证人处理器
type Processor struct {
	jgs JoiningGroups //已加入未完成初始化的组(组初始化完成上链后，不再需要)。组内成员数据过程数据。

	bcs map[string]*BlockContext //组ID->组铸块上下文
	gg  GlobalGroups             //全网组静态信息（包括待完成组初始化的组，即还没有组ID只有DUMMY ID的组）

	//sci SelfCastInfo //当前节点的出块信息（包括当前节点在不同高度不同QN值所有成功和不成功的出块）。组内成员动态数据。
	//////和组无关的矿工信息
	mi *MinerInfo
	//////加入(成功)的组信息(矿工节点数据)
	belongGroups map[string]JoinedGroup //当前ID参与了哪些(已上链，可铸块的)组, 组id_str->组内私密数据（组外不可见或加速缓存）
	//////测试数据，代替屮逸的网络消息
	GroupProcs map[string]*Processor
	Ticker 			*ticker.GlobalTicker		//全局定时器, 组初始化完成后启动

	futureBlockMsg  map[common.Hash][]*ConsensusBlockMessage //存储缺少父块的块
	futureBlockLock sync.RWMutex

	futureVerifyMsg  map[common.Hash][]*ConsensusBlockMessageBase //存储缺失前一块的验证消息
	futureVerifyLock sync.RWMutex

	storage 	tasdb.Database
	ready 		bool //是否已初始化完成

	piece_count  int
	inited_count int
	load_data    int
	save_data    int
	//////链接口
	MainChain  core.BlockChainI
	GroupChain *core.GroupChain
	//锁
	initLock sync.Mutex   //组初始化锁
	rwLock   sync.RWMutex //读写锁
}

//取得组内成员的签名公钥
func (p Processor) GetMemberSignPubKey(gmi GroupMinerID) (pk groupsig.Pubkey) {
	if jg, ok := p.belongGroups[gmi.gid.GetHexString()]; ok {
		pk = jg.GetMemSignPK(gmi.uid)
	}
	return
}

//取得组内自身的私密私钥（正式版本不提供）
// deprecated
func (p Processor) getGroupSeedSecKey(gid groupsig.ID) (sk groupsig.Seckey) {
	if jg, ok := p.belongGroups[gid.GetHexString()]; ok {
		sk = jg.SeedKey
	}
	return
}

func GetSecKeyPrefix(sk groupsig.Seckey) string {
	str := sk.GetHexString()
	if len(str) >= 12 {
		link := str[0:6] + "-" + str[len(str)-6:]
		return link
	} else {
		return str[0:]
	}
}

func GetPubKeyPrefix(pk groupsig.Pubkey) string {
	str := pk.GetHexString()
	if len(str) >= 12 {
		link := str[0:6] + "-" + str[len(str)-6:]
		return link
	} else {
		return str[0:]
	}
}

func GetIDPrefix(id groupsig.ID) string {
	str := id.GetHexString()
	if len(str) >= 12 {
		link := str[0:6] + "-" + str[len(str)-6:]
		return link
	} else {
		return str[0:]
	}
}

func GetHashPrefix(h common.Hash) string {
	str := h.Hex()
	if len(str) >= 12 {
		link := str[0:6] + "-" + str[len(str)-6:]
		return link
	} else {
		return str[0:]
	}
}

func GetSignPrefix(sign groupsig.Signature) string {
	str := sign.GetHexString()
	if len(str) >= 12 {
		link := str[0:6] + "-" + str[len(str)-6:]
		return link
	} else {
		return str[0:]
	}
}

func GetCastExpireTime(base time.Time, deltaHeight uint64) time.Time {
	return base.Add(time.Second * time.Duration(deltaHeight * uint64(MAX_GROUP_BLOCK_TIME)))
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


//立即触发一次检查自己是否下个铸块组
func (p *Processor) triggerCastCheck()  {
    //p.Ticker.StartTickerRoutine(p.getCastCheckRoutineName(), true)
    p.Ticker.StartAndTriggerRoutine(p.getCastCheckRoutineName())
}

//检查是否当前组铸块
func (p *Processor) checkSelfCastRoutine() bool {
	begin := time.Now()
	defer func() {
		log.Printf("checkSelfCastRoutine: begin at %v, cost %v", begin, time.Since(begin).String())
	}()
	if !p.Ready() {
		return false
	}

	if len(p.belongGroups) == 0 || len(p.bcs) == 0 {
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
	)

	if top.Height > 0 {
		d := time.Since(top.CurTime)
		if d < 0 {
			return false
		}

		deltaHeight := uint64(d.Seconds()) / uint64(MAX_GROUP_BLOCK_TIME) + 1
		castHeight = top.Height + deltaHeight
		expireTime = GetCastExpireTime(top.CurTime, deltaHeight)
	} else {
		castHeight = uint64(1)
		expireTime = GetCastExpireTime(time.Now(), 1)
	}

	log.Printf("checkSelfCastRoutine: topHeight=%v, topHash=%v, topCurTime=%v, castHeight=%v, expireTime=%v\n", top.Height, GetHashPrefix(top.Hash), top.CurTime, castHeight, expireTime)

	casting := false
	for _, _bc := range p.bcs {
		if _bc.alreadyInCasting(castHeight, top.Hash) {
			log.Printf("checkSelfCastRoutine: already in cast height, castInfo=%v", _bc.castingInfo())
			casting = true
			break
		}
	}
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
		bc := p.GetBlockContext(selectGroup.GetHexString())
		if bc == nil {
			log.Printf("[ERROR]checkSelfCastRoutine: get nil blockcontext!, gid=%v", GetIDPrefix(*selectGroup))
			return false
		}

		log.Printf("MYGOD! BECOME NEXT CAST GROUP! uid=%v, gid=%v\n", GetIDPrefix(p.GetMinerID()), GetIDPrefix(*selectGroup))
		bc.StartCast(castHeight, top.CurTime, top.Hash, expireTime)

		return true
	} else { //自己不是下一个铸块组, 但是当前在铸块
		for _, _bc := range p.bcs {
			log.Printf("reset casting blockcontext: castingInfo=%v", _bc.castingInfo())
			_bc.Reset()
		}
	}

	return false
}

func (p *Processor) getCastCheckRoutineName() string {
    return "self_cast_check_" + p.getPrefix()
}

//初始化矿工数据（和组无关）
func (p *Processor) Init(mi MinerInfo) bool {
	p.ready = false
	p.futureBlockMsg = make(map[common.Hash][]*ConsensusBlockMessage)
	p.futureVerifyMsg = make(map[common.Hash][]*ConsensusBlockMessageBase)
	p.MainChain = core.BlockChainImpl
	p.GroupChain = core.GroupChainImpl
	p.mi = &mi
	p.gg.Init()
	p.jgs.Init()
	p.belongGroups = make(map[string]JoinedGroup, 0)
	p.bcs = make(map[string]*BlockContext, 0)

	db, err := datasource.NewDatabase(STORE_PREFIX)
	if err != nil {
		log.Printf("NewDatabase error %v\n", err)
		return false
	}
	p.storage = db
	//p.sci.Init()

	cc := common.GlobalConf.GetSectionManager("consensus")
	p.load_data = cc.GetInt("LOAD_DATA", 0)
	p.save_data = cc.GetInt("SAVE_DATA", 0)

	p.Ticker = ticker.NewGlobalTicker(p.getPrefix())

	log.Printf("proc(%v) inited 2.\n", p.getPrefix())
	consensusLogger.Infof("ProcessorId:%v", p.getPrefix())
	return true
}

//预留接口
//后续如有全局定时器，从这个函数启动
func (p *Processor) Start() bool {
	p.Ticker.RegisterRoutine(p.getCastCheckRoutineName(), p.checkSelfCastRoutine, 50)
	p.prepareMiner()
	p.ready = true
	return true
}

//预留接口
func (p *Processor) Stop() {
	return
}

//增加一个铸块上下文（一个组有一个铸块上下文）
func (p *Processor) AddBlockContext(bc *BlockContext) bool {
	var add bool
	if p.GetBlockContext(bc.MinerID.gid.GetHexString()) == nil {
		p.bcs[bc.MinerID.gid.GetHexString()] = bc
		add = true
	}
	log.Printf("AddBlockContext, gid=%v, result=%v\n.", GetIDPrefix(bc.MinerID.gid), add)
	return add
}

//取得一个铸块上下文
//gid:组ID hex 字符串
func (p *Processor) GetBlockContext(gid string) *BlockContext {
	if v, ok := p.bcs[gid]; ok {
		return v
	} else {
		return nil
	}
}


//取得矿工ID（和组无关）
func (p Processor) GetMinerID() groupsig.ID {
	return p.mi.MinerID
}

//取得矿工参与的所有铸块组私密私钥，正式版不提供
func (p Processor) getMinerGroups() map[string]JoinedGroup {
	return p.belongGroups
}

//加入一个组（一个矿工ID可以加入多个组）
//gid : 组ID(非dummy id)
//sk：用户的组成员签名私钥
func (p *Processor) addInnerGroup(g *JoinedGroup, save bool) {
	log.Printf("begin Processor::addInnerGroup, gid=%v...\n", GetIDPrefix(g.GroupID))
	if !p.IsMinerGroup(g.GroupID) {
		p.belongGroups[g.GroupID.GetHexString()] = *g
		if save {
			p.saveJoinedGroup(g)
		}
	} else {
		log.Printf("Error::Processor::AddSignKey failed, already exist.\n")
	}
	log.Printf("SUCCESS:node=%v inited group=%v, sign key=%v.\n", p.getPrefix(), GetIDPrefix(g.GroupID), GetSecKeyPrefix(g.SignKey))
	return
}

//取得矿工在某个组的签名私钥
func (p Processor) getSignKey(gid groupsig.ID) groupsig.Seckey {
	return p.belongGroups[gid.GetHexString()].SignKey //如该组不存在则返回空值
}

//检测某个组是否矿工的铸块组（一个矿工可以参与多个组）
func (p Processor) IsMinerGroup(gid groupsig.ID) bool {
	_, ok := p.belongGroups[gid.GetHexString()]
	return ok
}

func (p *Processor) calcCastGroup(preBH *types.BlockHeader, height uint64) *groupsig.ID {
	var hash common.Hash
	data := preBH.Signature

	deltaHeight := height - preBH.Height
	for ; deltaHeight > 0; deltaHeight -- {
		hash = rand.Data2CommonHash(data)
		data = hash.Bytes()
	}

	selectGroup, err := p.gg.SelectNextGroup(hash, height)
	if err != nil {
		log.Println("calcCastGroup err:", err)
		return nil
	}
	return &selectGroup
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

	groupInfo, err := p.gg.GetGroupByID(*selectGroupId) //取得合法的铸块组
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
func (p *Processor) CreateDummyGroup(miners []PubKeyInfo, gn string) int {
	if len(miners) != GROUP_MAX_MEMBERS {
		log.Printf("create group error, group max members=%v, real=%v.\n", GROUP_MAX_MEMBERS, len(miners))
		return -1
	}
	var gis ConsensusGroupInitSummary
	//gis.ParentID = p.GetMinerID()

	//todo future bug
	parentID := groupsig.DeserializeId([]byte("genesis group dummy"))

	gis.ParentID = *parentID
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
	grm.GI = gis
	grm.SI = GenSignData(grm.GI.GenHash(), p.GetMinerID(), p.getMinerInfo().GetDefaultSecKey())
	log.Printf("proc(%v) Create NewAccountDB Group, send network msg to members...\n", p.getPrefix())
	log.Printf("call network service SendGroupInitMessage...\n")
	//dummy 组写入组链 add by 小熊
	//members := make([]core.Member, 0)
	//for _, miner := range miners {
	//	member := core.Member{Id: miner.ID.Serialize(), PubKey: miner.PK.Serialize()}
	//	members = append(members, member)
	//}
	////此时组ID 跟组公钥是没有的
	//group := core.Group{Members: members, Dummy: gis.DummyID.Serialize(), Parent: []byte("genesis group dummy")}
	//err := p.GroupChain.AddGroup(&group, nil, nil)
	//if err != nil {
	//	log.Printf("Add dummy group error:%s\n", err.Error())
	//} else {
	//	log.Printf("Add dummy to chain success! count: %d, now: %d", core.GroupChainImpl.Count(), len(core.GroupChainImpl.GetAllGroupID()))
	//}
	//log.Printf("Waiting 60s for dummy group sync...\n")
	//time.Sleep(30 * time.Second)
	SendGroupInitMessage(grm)
	return 0
}

//检测是否激活成为当前铸块组，成功激活返回有效的bc，激活失败返回nil
func (p *Processor) verifyCastSign(cgs *CastGroupSummary, si *SignData) bool {

	if !p.IsMinerGroup(cgs.GroupID) { //检测当前节点是否在该铸块组
		log.Printf("beingCastGroup failed, node not in this group.\n")
		return false
	}
	_, err := p.gg.GetGroupByID(cgs.GroupID)
	if err != nil {
		panic("gg.GetGroupByID failed.")
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

//在某个区块高度的QN值成功出块，保存上链，向组外广播
//同一个高度，可能会因QN不同而多次调用该函数
//但一旦低的QN出过，就不该出高的QN。即该函数可能被多次调用，但是调用的QN值越来越小
func (p *Processor) SuccessNewBlock(bh *types.BlockHeader, vctx *VerifyContext, gid groupsig.ID) {
	begin := time.Now()
	defer func() {
		log.Printf("SuccessNewBlock begin at %v, cost %v\n", begin.String(), time.Since(begin).String())
	}()

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

	var cbm ConsensusBlockMessage
	cbm.Block = *block
	cbm.GroupID = gid
	ski := SecKeyInfo{p.GetMinerID(), p.mi.GetDefaultSecKey()}
	cbm.GenSign(ski)
	if !PROC_TEST_MODE {
		logHalfway("SuccessNewBlock", bh.Height, bh.QueueNumber, p.getPrefix(), "SuccessNewBlock, hash %v, 耗时%v秒", GetHashPrefix(bh.Hash), time.Since(bh.CurTime).Seconds())
		go BroadcastNewBlock(&cbm)
		p.triggerCastCheck()
	}
	return
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
	gc := p.jgs.GetGroup(gid)
	node := gc.GetNode()
	if node != nil {
		pub_piece = node.GetSeedPubKey()
	}
	return pub_piece
}

//取得自己参与的某个铸块组的私钥片段（聚合一个组所有成员的私钥片段，可以生成组私钥）
//用于测试目的，正式版对外不提供。
func (p Processor) getMinerSecKeyPieceForGroup(gid groupsig.ID) groupsig.Seckey {
	var sec_piece groupsig.Seckey
	gc := p.jgs.GetGroup(gid)
	node := gc.GetNode()
	if node != nil {
		sec_piece = node.getSeedSecKey()
	}
	return sec_piece
}

//取得特定的组
func (p Processor) getGroup(gid groupsig.ID) StaticGroupInfo {
	g, err := p.gg.GetGroupByID(gid)
	if err != nil {
		panic("GetSelfGroup failed.")
	}
	return g
}

//取得一个铸块组的公钥(processer初始化时从链上加载)
func (p Processor) getGroupPubKey(gid groupsig.ID) groupsig.Pubkey {
	g, err := p.gg.GetGroupByID(gid)
	if err != nil {
		panic("GetSelfGroup failed.")
	}
	return g.GetPubKey()
}

func genDummyBH(qn int, uid groupsig.ID) *types.BlockHeader {
	bh := new(types.BlockHeader)
	bh.PreTime = time.Now()
	bh.Hash = rand.String2CommonHash("thiefox")
	bh.Height = 2
	bh.PreHash = rand.String2CommonHash("TASchain")
	bh.QueueNumber = uint64(qn)
	bh.Nonce = 123
	bh.Castor = uid.Serialize()
	sleep_d, err := time.ParseDuration("20ms")
	if err == nil {
		time.Sleep(sleep_d)
	} else {
		panic("time.ParseDuration failed.")
	}
	bh.CurTime = time.Now()

	hash := bh.GenHash()
	bh.Hash = hash
	log.Printf("bh.Hash=%v.\n", GetHashPrefix(bh.Hash))
	hash2 := bh.GenHash()
	if hash != hash2 {
		log.Printf("bh GenHash twice failed, first=%v, second=%v.\n", GetHashPrefix(hash), GetHashPrefix(hash2))
		panic("bh GenHash twice failed, hash diff.")
	}
	return bh
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

//当前节点成为KING，出块
func (p Processor) castBlock(bc *BlockContext, vctx *VerifyContext, qn int64) *types.BlockHeader {

	height := vctx.castHeight

	log.Printf("begin Processor::castBlock, height=%v, qn=%v...\n", height, qn)
	//var hash common.Hash
	//hash = bh.Hash //TO DO:替换成出块头的哈希
	//to do : change nonce
	nonce := time.Now().Unix()
	gid := bc.MinerID.gid

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

	var si SignData
	si.DataHash = bh.Hash
	si.SignMember = p.GetMinerID()

	if bh.Height > 0 && si.DataSign.IsValid() && bh.Height == height && bh.PreHash == vctx.prevHash {
		//发送该出块消息
		var ccm ConsensusCastMessage
		ccm.BH = *bh
		//ccm.GroupID = gid
		ccm.GenSign(SecKeyInfo{p.GetMinerID(), p.getSignKey(gid)})

		logHalfway("CASTBLOCK", height, uint64(qn), p.getPrefix(), "铸块成功, SendVerifiedCast, hash %v, 时间间隔 %v", GetHashPrefix(bh.Hash), bh.CurTime.Sub(bh.PreTime).Seconds())
		if !PROC_TEST_MODE {
			go SendCastVerify(&ccm)
		} else {
			for _, proc := range p.GroupProcs {
				proc.OnMessageCast(ccm)
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
