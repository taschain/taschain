package logical

import (
	"common"
	"consensus/groupsig"
	"encoding/json"
	"sync"
	//"consensus/net"
	"consensus/rand"
	"core"
	"fmt"
	"time"
	"log"
	"consensus/ticker"
)

//自己的出块信息
type SelfCastInfo struct {
	block_qns map[uint64][]uint //当前节点已经出过的块(高度->出块QN列表)
}

func (sci *SelfCastInfo) Init() {
	sci.block_qns = make(map[uint64][]uint, 0)
}

func (sci *SelfCastInfo) FindQN(height uint64, newQN uint) bool {
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
func (sci *SelfCastInfo) AddQN(height uint64, newQN uint) bool {
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

var PROC_TEST_MODE bool

//见证人处理器
type Processer struct {
	jgs JoiningGroups //已加入未完成初始化的组(组初始化完成上链后，不再需要)。组内成员数据过程数据。

	bcs map[string]*BlockContext //组ID->组铸块上下文
	gg  GlobalGroups             //全网组静态信息（包括待完成组初始化的组，即还没有组ID只有DUMMY ID的组）

	sci SelfCastInfo //当前节点的出块信息（包括当前节点在不同高度不同QN值所有成功和不成功的出块）。组内成员动态数据。
	//////和组无关的矿工信息
	mi MinerInfo
	//////加入(成功)的组信息(矿工节点数据)
	belongGroups map[string]JoinedGroup //当前ID参与了哪些(已上链，可铸块的)组, 组id_str->组内私密数据（组外不可见或加速缓存）
	//////测试数据，代替屮逸的网络消息
	GroupProcs map[string]*Processer
	Ticker 			*ticker.GlobalTicker		//全局定时器, 组初始化完成后启动

	futureBlockMsg map[common.Hash][]*ConsensusBlockMessage		//存储缺少父块的块
	futureBlockLock sync.RWMutex

	futureVerifyMsg	map[common.Hash][]*ConsensusBlockMessageBase	//存储缺失前一块的验证消息
	futureVerifyLock sync.RWMutex

	piece_count  int
	inited_count int
	load_data    int
	save_data    int
	//////链接口
	MainChain core.BlockChainI
	GroupChain *core.GroupChain
	//锁
	initLock sync.Mutex //组初始化锁
	rwLock sync.RWMutex 	//读写锁
}

//取得组内成员的签名公钥
func (p Processer) GetMemberSignPubKey(gmi GroupMinerID) (pk groupsig.Pubkey) {
	if jg, ok := p.belongGroups[gmi.gid.GetHexString()]; ok {
		pk = jg.GetMemSignPK(gmi.uid)
	}
	return
}

//取得组内自身的私密私钥（正式版本不提供）
func (p Processer) getGroupSeedSecKey(gid groupsig.ID) (sk groupsig.Seckey) {
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

func (p Processer) getPrefix() string {
	return GetIDPrefix(p.GetMinerID())
}

//私密函数，用于测试，正式版本不提供
func (p Processer) getmi() MinerInfo {
	return p.mi
}

func (p Processer) GetMinerInfo() PubKeyInfo {
	return PubKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultPubKey()}
}

func (p *Processer) setProcs(gps map[string]*Processer) {
	p.GroupProcs = gps
}

func (p *Processer) Load() bool {
	log.Printf("proc(%v) begin Load, group_count=%v, bcg_count=%v, bg_count=%v...\n",
		p.getPrefix(), len(p.gg.groups), len(p.bcs), len(p.belongGroups))
	cc := common.GlobalConf.GetSectionManager("consensus")
	var str string
	var buf []byte
	var err error
	/*
		str = cc.GetString("BlockContexts", "")
		if len(str) == 0 {
			return false
		}
		var buf []byte = []byte(str)
		err = json.Unmarshal(buf, p.bcs)
		if err != nil {
			fmt.Println("error:", err)
			panic("Processer::Load Unmarshal failed 1.")
		}
	*/
	p.belongGroups = make(map[string]JoinedGroup, 0)
	str = cc.GetString("BelongGroups", "")
	if len(str) == 0 {
		return false
	}
	log.Printf("unmarshal string=%v.\n", str)
	buf = []byte(str)
	err = json.Unmarshal(buf, &p.belongGroups)
	if err != nil {
		fmt.Println("error:", err)
		panic("Processer::Load Unmarshal failed 2.")
	}
	log.Printf("belongGroups info: len=%v...\n", len(p.belongGroups))
	for _, v := range p.belongGroups {
		log.Printf("---gid=%v, gpk=%v, seed_sk=%v, sign_sk=%v, mems=%v.\n",
			GetIDPrefix(v.GroupID), GetPubKeyPrefix(v.GroupPK), GetSecKeyPrefix(v.SeedKey), GetSecKeyPrefix(v.SignKey), len(v.Members))
	}

	p.gg.Load()

	log.Printf("end Precesser::Load, group_count=%v, bcg_count=%v, bg_count=%v...\n", len(p.gg.groups), len(p.bcs), len(p.belongGroups))
	return true
}

func (p Processer) Save() {
	log.Printf("proc(%v) begin Save, group_count=%v, bcg_count=%v, bg_count=%v...\n",
		p.getPrefix(), len(p.gg.groups), len(p.bcs), len(p.belongGroups))
	cc := common.GlobalConf.GetSectionManager("consensus")
	var str string
	var buf []byte
	var err error
	/*
		buf, err = json.Marshal(p.bcs)
		if err != nil {
			fmt.Println("error 1:", err)
			panic("Processer::Save Marshal failed 1.")
		}
		str = string(buf[:])

		cc.SetString("BlockContexts", str)
	*/
	log.Printf("belongGroups info: len=%v...\n", len(p.belongGroups))
	for _, v := range p.belongGroups {
		log.Printf("---gid=%v, gpk=%v, seed_sk=%v, sign_sk=%v, mems=%v.\n",
			GetIDPrefix(v.GroupID), GetPubKeyPrefix(v.GroupPK), GetSecKeyPrefix(v.SeedKey), GetSecKeyPrefix(v.SignKey), len(v.Members))
	}
	buf, err = json.Marshal(p.belongGroups)
	if err != nil {
		fmt.Println("error 2:", err)
		panic("Processer::Save Marshal failed 2.")
	}
	str = string(buf[:])
	log.Printf("marshal string=%v.\n", str)
	cc.SetString("BelongGroups", str)

	p.gg.Save()

	log.Printf("end Processer::Save.\n")
	return
}

//立即触发一次检查自己是否下个铸块组
func (p *Processer) triggerCastCheck()  {
    p.Ticker.StartTickerRoutine(p.getCastCheckRoutineName(), true)
}

//检查是否当前组铸块
func (p *Processer) checkSelfCastRoutine() bool {
	begin := time.Now()
	defer func() {
		log.Printf("checkSelfCastRoutine: begin at %v, cost %v", begin, time.Since(begin).String())
	}()

	if len(p.belongGroups) == 0 || len(p.bcs) == 0 {
		log.Printf("current node don't belong to anygroup!!")
		return false
	}

	top := p.MainChain.QueryTopBlock()

	castHeight := uint64(1)

	if top.Height > 0 {
		d := time.Since(top.CurTime)
		if d < 0 {
			return false
		}

		deltaHeight := uint64(d.Seconds()) / uint64(MAX_GROUP_BLOCK_TIME) + 1
		castHeight = top.Height + deltaHeight
	}

	log.Printf("checkSelfCastRoutine: topHeight=%v, topHash=%v, topCurTime=%v, castHeight=%v", top.Height, top.Hash, top.CurTime, castHeight)

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
		bc.StartCast(castHeight, top.CurTime, top.Hash, true)

		return true
	} else  {	//自己不是下一个铸块组, 但是当前在铸块
		for _, _bc := range p.bcs {
			log.Printf("reset casting blockcontext: castingInfo=%v", _bc.castingInfo())
			_bc.Reset()
		}
	}

	return false
}

func (p *Processer) getCastCheckRoutineName() string {
    return "self_cast_check_" + p.getPrefix()
}

//初始化矿工数据（和组无关）
func (p *Processer) Init(mi MinerInfo) bool {
	p.futureBlockMsg = make(map[common.Hash][]*ConsensusBlockMessage)
	p.futureVerifyMsg = make(map[common.Hash][]*ConsensusBlockMessageBase)
	p.MainChain = core.BlockChainImpl
	p.GroupChain = core.GroupChainImpl
	p.mi = mi
	p.gg.Init()
	p.jgs.Init()
	p.belongGroups = make(map[string]JoinedGroup, 0)
	p.bcs = make(map[string]*BlockContext, 0)
	p.sci.Init()

	cc := common.GlobalConf.GetSectionManager("consensus")
	p.load_data = cc.GetInt("LOAD_DATA", 0)
	p.save_data = cc.GetInt("SAVE_DATA", 0)

	log.Printf("proc(%v) inited 1, load_data=%v, save_data=%v.\n", p.getPrefix(), p.load_data, p.save_data)

	if p.load_data == 1 {
		b := p.Load()
		log.Printf("proc(%v) load_data result=%v.\n", p.getPrefix(), b)
	}
	p.Ticker = ticker.NewGlobalTicker(p.getPrefix())
	p.Ticker.RegisterRoutine(p.getCastCheckRoutineName(), p.checkSelfCastRoutine, 4)

	log.Printf("proc(%v) inited 2.\n", p.getPrefix())
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

//增加一个铸块上下文（一个组有一个铸块上下文）
func (p *Processer) AddBlockContext(bc *BlockContext) bool {
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
func (p *Processer) GetBlockContext(gid string) *BlockContext {
	if v, ok := p.bcs[gid]; ok {
		return v
	} else {
		return nil
	}
}

////取得当前在铸块中的组数据
////该函数最多返回一个bc，或者=nil。不允许同时返回多个bc，实际也不会发生这种情况。
//func (p *Processer) GetCastingBC() *BlockContext {
//	var bc *BlockContext
//	for _, v := range p.bcs {
//		if v.IsCasting() {
//			if bc != nil {
//				panic("Processer::GetCastingBC failed, same time more than one casting group.")
//			}
//			bc = v
//			//break    //TO DO : 验证正确后打开
//		}
//	}
//	return bc
//}

//取得矿工ID（和组无关）
func (p Processer) GetMinerID() groupsig.ID {
	return p.mi.MinerID
}

//取得矿工参与的所有铸块组私密私钥，正式版不提供
func (p Processer) getMinerGroups() map[string]JoinedGroup {
	return p.belongGroups
}

//加入一个组（一个矿工ID可以加入多个组）
//gid : 组ID(非dummy id)
//sk：用户的组成员签名私钥
func (p *Processer) addInnerGroup(g JoinedGroup) {
	log.Printf("begin Processer::addInnerGroup, gid=%v...\n", GetIDPrefix(g.GroupID))
	if !p.IsMinerGroup(g.GroupID) {
		p.belongGroups[g.GroupID.GetHexString()] = g
	} else {
		log.Printf("Error::Processer::AddSignKey failed, already exist.\n")
	}
	log.Printf("SUCCESS:node=%v inited group=%v, sign key=%v.\n", p.getPrefix(), GetIDPrefix(g.GroupID), GetSecKeyPrefix(g.SignKey))
	return
}

//取得矿工在某个组的签名私钥
func (p Processer) getSignKey(gid groupsig.ID) groupsig.Seckey {
	return p.belongGroups[gid.GetHexString()].SignKey //如该组不存在则返回空值
}

//检测某个组是否矿工的铸块组（一个矿工可以参与多个组）
func (p Processer) IsMinerGroup(gid groupsig.ID) bool {
	_, ok := p.belongGroups[gid.GetHexString()]
	return ok
}

func (p *Processer) calcCastGroup(preBH *core.BlockHeader, height uint64) *groupsig.ID {
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
func (p *Processer) verifyGroupSign(b *core.Block, sd SignData) bool {
	bh := b.Header
	var gid groupsig.ID
	if gid.Deserialize(bh.GroupId) != nil {
		panic("verifyGroupSign: group id Deserialize failed.")
	}

	groupInfo := p.getGroup(gid)
	if !groupInfo.GroupID.IsValid() {
		 panic("verifyGroupSign: get group is nil!, gid=" + GetIDPrefix(gid))
	}

	log.Printf("verifyGroupSign: real cast group is expect group(=%v), VerifySign...\n", GetIDPrefix(gid))
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
func (p *Processer) isCastGroupLeagal(bh *core.BlockHeader, preHeader *core.BlockHeader) (result bool) {
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
func (p *Processer) BeginGenesisGroupMember() PubKeyInfo {
	gis := p.GenGenesisGroupSummary()
	temp_mi := p.getmi()
	temp_mgs := NewMinerGroupSecret(temp_mi.GenSecretForGroup(gis.GenHash()))
	gsk_piece := temp_mgs.GenSecKey()
	gpk_piece := *groupsig.NewPubkeyFromSeckey(gsk_piece)
	pki := PubKeyInfo{p.GetMinerID(), gpk_piece}
	log.Printf("\nBegin Genesis Group Member, ID=%v, gpk_piece=%v.\n", GetIDPrefix(pki.GetID()), GetPubKeyPrefix(pki.PK))
	return pki
}

func (p *Processer) GenGenesisGroupSummary() ConsensusGroupInitSummary {
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
func (p *Processer) CreateDummyGroup(miners []PubKeyInfo, gn string) int {
	if len(miners) != GROUP_MAX_MEMBERS {
		log.Printf("create group error, group max members=%v, real=%v.\n", GROUP_MAX_MEMBERS, len(miners))
		return -1
	}
	var gis ConsensusGroupInitSummary
	//gis.ParentID = p.GetMinerID()

	var parentID groupsig.ID
	//todo future bug
	parentID.Deserialize([]byte("genesis group dummy"))
	gis.ParentID = parentID
	gis.DummyID = *groupsig.NewIDFromString(gn)
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
	grm.SI = GenSignData(grm.GI.GenHash(), p.GetMinerID(), p.getmi().GetDefaultSecKey())
	log.Printf("proc(%v) Create New Group, send network msg to members...\n", p.getPrefix())
	log.Printf("call network service SendGroupInitMessage...\n")
	//dummy 组写入组链 add by 小熊
	members := make([]core.Member, 0)
	for _, miner := range miners {
		member := core.Member{Id: miner.ID.Serialize(), PubKey: miner.PK.Serialize()}
		members = append(members, member)
	}
	//此时组ID 跟组公钥是没有的
	group := core.Group{Members: members, Dummy: gis.DummyID.Serialize(), Parent: []byte("genesis group dummy")}
	err := p.GroupChain.AddGroup(&group, nil, nil)
	if err != nil {
		log.Printf("Add dummy group error:%s\n", err.Error())
	} else {
		log.Printf("Add dummy to chain success! count: %d, now: %d", core.GroupChainImpl.Count(), len(core.GroupChainImpl.GetAllGroupID()))
	}
	log.Printf("Waiting 20s for dummy group sync...\n")
	time.Sleep(20 * time.Second)
	log.Printf("Begin group init!\n")
	SendGroupInitMessage(grm)
	return 0
}

//检测是否激活成为当前铸块组，成功激活返回有效的bc，激活失败返回nil
func (p *Processer) verifyCastSign(cgs *CastGroupSummary, si *SignData) bool {
	log.Printf("proc(%v) verifyCast, sender=%v, height=%v, pre_time=%v...\n", p.getPrefix(), GetIDPrefix(si.GetID()), cgs.BlockHeight, cgs.PreTime.Format(time.Stamp))

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
		log.Printf("message sender's signPk=%v.\n", GetPubKeyPrefix(signPk))
		log.Printf("verifyCast::si info: id=%v, data hash=%v, sign=%v.\n",
			GetIDPrefix(si.GetID()), GetHashPrefix(si.DataHash), GetSignPrefix(si.DataSign))
		if si.VerifySign(signPk) { //消息合法
			log.Printf("message verify sign OK, find gid=%v blockContext...\n", GetIDPrefix(cgs.GroupID))
			return true
		} else {
			log.Printf("ERROR, message verify failed, data_hash=%v.\n", GetHashPrefix(si.DataHash))
			return false
		}
	} else {
		log.Printf("message sender's signPk not in joined groups, ignored.\n")
		return false
	}
}



//在某个区块高度的QN值成功出块，保存上链，向组外广播
//同一个高度，可能会因QN不同而多次调用该函数
//但一旦低的QN出过，就不该出高的QN。即该函数可能被多次调用，但是调用的QN值越来越小
func (p *Processer) SuccessNewBlock(bh *core.BlockHeader, vctx *VerifyContext, gid groupsig.ID) {
	begin := time.Now()
	defer func() {
		log.Printf("SuccessNewBlock begin at %v, cost %v\n", begin.String(), time.Since(begin).String())
	}()

	if bh == nil {
		panic("SuccessNewBlock arg failed.")
	}
	log.Printf("proc(%v) begin SuccessNewBlock, group=%v, qn=%v...\n", p.getPrefix(), GetIDPrefix(gid), bh.QueueNumber)


	block := p.MainChain.GenerateBlock(*bh)

	if block == nil {
		log.Printf("SuccessNewBlock core.GenerateBlock is nil! won't broadcast block!")
		if p.MainChain.QueryBlockByHash(bh.Hash) != nil {	//已经上链
			log.Printf("SuccessNewBlock core.GenerateBlock is nil! block alreayd onchain!")
			vctx.CastedUpdateStatus(int64(bh.QueueNumber))
		}
		return
	}

	r := p.doAddOnChain(block)

	if r != 0 && r != 1 {	//分叉调整或 上链失败都不走下面的逻辑
		return
	}
	vctx.CastedUpdateStatus(int64(bh.QueueNumber))

	var cbm ConsensusBlockMessage
	cbm.Block = *block
	cbm.GroupID = gid
	ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
	cbm.GenSign(ski)
	if !PROC_TEST_MODE {
		log.Printf("call network service BroadcastNewBlock...\n")
		go BroadcastNewBlock(&cbm)
		p.triggerCastCheck()
		p.triggerFutureVerifyMsg(bh.Hash)
	}
	log.Printf("proc(%v) end SuccessNewBlock.\n", p.getPrefix())
	return
}

//检查是否轮到自己出块
func (p *Processer) checkCastRoutine(bc *BlockContext, vctx *VerifyContext, kingIndex int32, qn int64) {
	//p.castLock.Lock()
	//defer p.castLock.Unlock()
	gid := bc.MinerID.gid
	height := vctx.castHeight

	log.Printf("prov(%v) begin CheckCastRoutine, gid=%v, kingIndex=%v, qn=%v, height=%v.\n", p.getPrefix(), GetIDPrefix(gid), kingIndex, qn, height)
	if kingIndex < 0 || qn < 0 {
		return
	}
	sgi := p.getGroup(gid)
	pos := sgi.GetMinerPos(p.GetMinerID()) //取得当前节点在组中的排位
	log.Printf("time=%v, Current KING=%v.\n", time.Now().Format(time.Stamp), GetIDPrefix(sgi.GetCastor(int(kingIndex))))
	log.Printf("Current node=%v, pos_index in group=%v.\n", p.getPrefix(), pos)
	if sgi.GetCastor(int(kingIndex)).GetHexString() == p.GetMinerID().GetHexString() { //轮到自己铸块
		log.Printf("curent node IS KING!\n")
		if !p.sci.FindQN(height, uint(qn)) { //在该高度该QN，自己还没铸过快
			head := p.castBlock(bc, vctx, qn) //铸块
			if head != nil {
				p.sci.AddQN(height, uint(qn))
			}
		} else {
			log.Printf("In height=%v, qn=%v current node already casted.\n", height, qn)
		}
	}
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


func outputBlockHeaderAndSign(prefix string, bh *core.BlockHeader, si *SignData)  {
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
func (p Processer) castBlock(bc *BlockContext, vctx *VerifyContext, qn int64) *core.BlockHeader {

	height := vctx.castHeight

	log.Printf("begin Processer::castBlock, height=%v, qn=%v...\n", height, qn)
	var bh *core.BlockHeader
	//var hash common.Hash
	//hash = bh.Hash //TO DO:替换成出块头的哈希
	//to do : change nonce
	nonce := time.Now().Unix()
	gid := bc.MinerID.gid

	//调用鸠兹的铸块处理
	if !PROC_TEST_MODE {
		block := p.MainChain.CastingBlock(uint64(height), uint64(nonce), uint64(qn), p.GetMinerID().Serialize(), gid.Serialize())
		if block == nil {
			log.Printf("MainChain::CastingBlock failed, height=%v, qn=%v, gid=%v, mid=%v.\n", height, qn, GetIDPrefix(gid), GetIDPrefix(p.GetMinerID()))
			//panic("MainChain::CastingBlock failed, jiuci return nil.\n")
			return nil
		}
		for _, tx := range block.Transactions {
			if tx == nil {
				log.Printf("castBlock tx hashs %v\n", block.Header.Transactions)
				//panic(fmt.Sprintf("castBlock transaction is nil!, height=%v, hash=%v", block.Header.Height, block.Header.Hash))
			}
		}

		bh = block.Header

		log.Printf("AAAAAA castBlock bh %v\n", p.blockPreview(bh))
		log.Printf("AAAAAA chain top bh %v\n", p.blockPreview(p.MainChain.QueryTopBlock()))

	} else {
		bh = genDummyBH(int(qn), p.GetMinerID())
		bh.GroupId = gid.Serialize()
		bh.Castor = p.GetMinerID().Serialize()
		bh.Nonce = uint64(nonce)
		bh.Height = uint64(height)
		bh.Hash = bh.GenHash()
	}

	var si SignData
	si.DataHash = bh.Hash
	si.SignMember = p.GetMinerID()
	b := si.GenSign(p.getSignKey(gid)) //组成员签名私钥片段签名
	log.Printf("castor sign bh result = %v.\n", b)

	if bh.Height > 0 && si.DataSign.IsValid() && bh.Height == height && bh.PreHash == vctx.prevHash {
		var tmp_id groupsig.ID
		if tmp_id.Deserialize(bh.Castor) != nil {
			panic("ID Deserialize failed.")
		}
		log.Printf("success cast block, height= %v, castor= %v.\n", bh.Height, GetIDPrefix(tmp_id))
		//发送该出块消息
		var ccm ConsensusCastMessage
		ccm.BH = *bh
		//ccm.GroupID = gid
		ccm.GenSign(SecKeyInfo{p.GetMinerID(), p.getSignKey(gid)})
		outputBlockHeaderAndSign("castBlock", bh, &ccm.SI)

		if !PROC_TEST_MODE {
			log.Printf("call network service SendCastVerify...\n")
			log.Printf("cast block info hash=%v, height=%v, prehash=%v, pretime=%v, castor=%v", GetHashPrefix(bh.Hash), bh.Height, GetHashPrefix(bh.PreHash), bh.PreTime, GetIDPrefix(p.GetMinerID()))
			go SendCastVerify(&ccm)
		} else {
			log.Printf("call proc.OnMessageCast direct...\n")
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
	log.Printf("end Processer::castBlock.\n")
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
