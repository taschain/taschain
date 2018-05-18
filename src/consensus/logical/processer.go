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
)

//计算当前距上一个铸块完成已经过去了几个铸块时间窗口（组间）
func getBlockTimeWindow(b time.Time) int {
	diff := time.Since(b).Seconds() //从上个铸块完成到现在的时间（秒）
	if diff >= 0 {
		return int(diff) / MAX_GROUP_BLOCK_TIME
	} else {
		return -1
	}
}

//计算当前距上一个铸块完成已经过去了几个出块时间窗口（组内）
func getCastTimeWindow(b time.Time) int {
	diff := time.Since(b).Seconds() //从上个铸块完成到现在的时间（秒）
	fmt.Printf("getCastTimeWindow, time_begin=%v, diff=%v.\n", b.Format(time.Stamp), diff)

	begin := 0.0
	cnt := 0
	for begin < diff {
		begin += float64(MAX_USER_CAST_TIME)
		cnt++
	}
	return cnt
}

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

	piece_count  int
	inited_count int
	load_data    int
	save_data    int
	//////链接口
	MainChain core.BlockChainI
	//锁
	initLock sync.Mutex //组初始化锁
	castLock sync.Mutex //组铸块锁
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
		link := str[0:6] + "-" + str[len(str)-6:len(str)]
		return link
	} else {
		return str[0:len(str)]
	}
}

func GetPubKeyPrefix(pk groupsig.Pubkey) string {
	str := pk.GetHexString()
	if len(str) >= 12 {
		link := str[0:6] + "-" + str[len(str)-6:len(str)]
		return link
	} else {
		return str[0:len(str)]
	}
}

func GetIDPrefix(id groupsig.ID) string {
	str := id.GetHexString()
	if len(str) >= 12 {
		link := str[0:6] + "-" + str[len(str)-6:len(str)]
		return link
	} else {
		return str[0:len(str)]
	}
}

func GetHashPrefix(h common.Hash) string {
	str := h.Hex()
	if len(str) >= 12 {
		link := str[0:6] + "-" + str[len(str)-6:len(str)]
		return link
	} else {
		return str[0:len(str)]
	}
}

func GetSignPrefix(sign groupsig.Signature) string {
	str := sign.GetHexString()
	if len(str) >= 12 {
		link := str[0:6] + "-" + str[len(str)-6:len(str)]
		return link
	} else {
		return str[0:len(str)]
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
	fmt.Printf("proc(%v) begin Load, group_count=%v, bcg_count=%v, bg_count=%v...\n",
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
	fmt.Printf("unmarshal string=%v.\n", str)
	buf = []byte(str)
	err = json.Unmarshal(buf, &p.belongGroups)
	if err != nil {
		fmt.Println("error:", err)
		panic("Processer::Load Unmarshal failed 2.")
	}
	fmt.Printf("belongGroups info: len=%v...\n", len(p.belongGroups))
	for _, v := range p.belongGroups {
		fmt.Printf("---gid=%v, gpk=%v, seed_sk=%v, sign_sk=%v, mems=%v.\n",
			GetIDPrefix(v.GroupID), GetPubKeyPrefix(v.GroupPK), GetSecKeyPrefix(v.SeedKey), GetSecKeyPrefix(v.SignKey), len(v.Members))
	}

	p.gg.Load()

	fmt.Printf("end Precesser::Load, group_count=%v, bcg_count=%v, bg_count=%v...\n", len(p.gg.groups), len(p.bcs), len(p.belongGroups))
	return true
}

func (p Processer) Save() {
	fmt.Printf("proc(%v) begin Save, group_count=%v, bcg_count=%v, bg_count=%v...\n",
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
	fmt.Printf("belongGroups info: len=%v...\n", len(p.belongGroups))
	for _, v := range p.belongGroups {
		fmt.Printf("---gid=%v, gpk=%v, seed_sk=%v, sign_sk=%v, mems=%v.\n",
			GetIDPrefix(v.GroupID), GetPubKeyPrefix(v.GroupPK), GetSecKeyPrefix(v.SeedKey), GetSecKeyPrefix(v.SignKey), len(v.Members))
	}
	buf, err = json.Marshal(p.belongGroups)
	if err != nil {
		fmt.Println("error 2:", err)
		panic("Processer::Save Marshal failed 2.")
	}
	str = string(buf[:])
	fmt.Printf("marshal string=%v.\n", str)
	cc.SetString("BelongGroups", str)

	p.gg.Save()

	fmt.Printf("end Processer::Save.\n")
	return
}

//初始化矿工数据（和组无关）
func (p *Processer) Init(mi MinerInfo) bool {
	p.MainChain = core.BlockChainImpl
	p.mi = mi
	p.gg.Init()
	p.jgs.Init()
	p.belongGroups = make(map[string]JoinedGroup, 0)
	p.bcs = make(map[string]*BlockContext, 0)
	p.sci.Init()

	cc := common.GlobalConf.GetSectionManager("consensus")
	p.load_data = cc.GetInt("LOAD_DATA", 0)
	p.save_data = cc.GetInt("SAVE_DATA", 0)

	fmt.Printf("proc(%v) inited 1, load_data=%v, save_data=%v.\n", p.getPrefix(), p.load_data, p.save_data)

	if p.load_data == 1 {
		b := p.Load()
		fmt.Printf("proc(%v) load_data result=%v.\n", p.getPrefix(), b)
	}

	fmt.Printf("proc(%v) inited 2.\n", p.getPrefix())
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
	fmt.Printf("AddBlockContext, gid=%v, result=%v\n.", GetIDPrefix(bc.MinerID.gid), add)
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

//取得当前在铸块中的组数据
//该函数最多返回一个bc，或者=nil。不允许同时返回多个bc，实际也不会发生这种情况。
func (p *Processer) GetCastingBC() *BlockContext {
	var bc *BlockContext
	for _, v := range p.bcs {
		if v.IsCasting() {
			if bc != nil {
				panic("Processer::GetCastingBC failed, same time more than one casting group.")
			}
			bc = v
			//break    //TO DO : 验证正确后打开
		}
	}
	return bc
}

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
	fmt.Printf("begin Processer::addInnerGroup, gid=%v...\n", GetIDPrefix(g.GroupID))
	if !p.IsMinerGroup(g.GroupID) {
		p.belongGroups[g.GroupID.GetHexString()] = g
	} else {
		fmt.Printf("Error::Processer::AddSignKey failed, already exist.\n")
	}
	fmt.Printf("SUCCESS:node=%v inited group=%v, sign key=%v.\n", p.getPrefix(), GetIDPrefix(g.GroupID), GetSecKeyPrefix(g.SignKey))
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

//检查区块头是否合法
func (p *Processer) isBHCastLegal(bh core.BlockHeader, sd SignData) (result bool) {
	//to do : 检查是否基于链上最高块的出块
	var gid groupsig.ID
	if gid.Deserialize(bh.GroupId) != nil {
		panic("isBHCastLegal, group id Deserialize failed.")
	}

	preHeader := p.MainChain.QueryBlockByHash(bh.PreHash)
	if preHeader == nil {
		fmt.Printf("isBHCastLegal: cannot find pre block header!,ignore block. %v, %v, %v\n", bh.PreHash, bh.Height, bh.Hash)
		return false
		//panic("isBHCastLegal: cannot find pre block header!,ignore block")
	}

	var sign groupsig.Signature
	if sign.Deserialize(preHeader.Signature) != nil {
		panic("OMB group sign Deserialize failed.")
	}

	gi := p.gg.GetCastGroup(sign.GetHash(), bh.Height) //取得合法的铸块组

	if gi.GroupID.IsEqual(gid) {
		fmt.Printf("BHCastLegal, real cast group is expect group(=%v), VerifySign...\n", GetIDPrefix(gid))
		//检查组签名是否正确
		var g_sign groupsig.Signature
		if g_sign.Deserialize(bh.Signature) != nil {
			panic("isBHCastLegal sign Deserialize failed.")
		}
		result = groupsig.VerifySig(gi.GroupPK, bh.Hash.Bytes(), g_sign)
		if !result {
			fmt.Printf("isBHCastLegal::verify group sign failed, gpk=%v, hash=%v, sign=%v. gid=%v.\n",
				GetPubKeyPrefix(gi.GroupPK), GetHashPrefix(bh.Hash), GetSignPrefix(g_sign), GetIDPrefix(gid))
			panic("isBHCastLegal::verify group sign failed")
		}
		//to do ：对铸块的矿工（组内最终铸块者，非KING）签名做验证
	} else {
		fmt.Printf("BHCastLegal failed, expect group=%v, real cast group=%v.\n", GetIDPrefix(gi.GroupID), GetIDPrefix(gid))
		//panic("isBHCastLegal failed, not expect group")  非法铸块组 直接跳过就行了吧?
	}
	fmt.Printf("BHCastLegal result=%v.\n", result)
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
	fmt.Printf("\nBegin Genesis Group Member, ID=%v, gpk_piece=%v.\n", GetIDPrefix(pki.GetID()), GetPubKeyPrefix(pki.PK))
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
		fmt.Printf("create group error, group max members=%v, real=%v.\n", GROUP_MAX_MEMBERS, len(miners))
		return -1
	}
	var gis ConsensusGroupInitSummary
	//gis.ParentID = p.GetMinerID()

	var parentID groupsig.ID
	//todo future bug
	parentID.Deserialize([]byte("genesis group dummy"))
	gis.ParentID = parentID
	gis.DummyID = *groupsig.NewIDFromString(gn)
	fmt.Printf("create group, group name=%v, group dummy id=%v.\n", gn, GetIDPrefix(gis.DummyID))
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
	fmt.Printf("proc(%v) Create New Group, send network msg to members...\n", p.getPrefix())
	fmt.Printf("call network service SendGroupInitMessage...\n")
	//dummy 组写入组链 add by 小熊
	members := make([]core.Member, 0)
	for _, miner := range miners {
		member := core.Member{Id: miner.ID.Serialize(), PubKey: miner.PK.Serialize()}
		members = append(members, member)
	}
	//此时组ID 跟组公钥是没有的
	group := core.Group{Members: members, Dummy: gis.DummyID.Serialize(), Parent: []byte("genesis group dummy")}
	err := core.GroupChainImpl.AddGroup(&group, nil, nil)
	if err != nil {
		fmt.Printf("Add dummy group error:%s\n", err.Error())
	} else {
		fmt.Printf("Add dummy to chain success! count: %d, now: %d", core.GroupChainImpl.Count(), len(core.GroupChainImpl.GetAllGroupID()))
	}
	fmt.Printf("Waiting 60s for dummy group sync...\n")
	time.Sleep(30 * time.Second)
	SendGroupInitMessage(grm)
	return 0
}

//检测是否激活成为当前铸块组，成功激活返回有效的bc，激活失败返回nil
func (p *Processer) beingCastGroup(cgs CastGroupSummary, si SignData) (bc *BlockContext, first bool) {
	fmt.Printf("proc(%v) beingCastGroup, sender=%v, height=%v, pre_time=%v...\n", p.getPrefix(),
		GetIDPrefix(si.GetID()), cgs.BlockHeight, cgs.PreTime.Format(time.Stamp))
	if !p.IsMinerGroup(cgs.GroupID) { //检测当前节点是否在该铸块组
		fmt.Printf("beingCastGroup failed, node not in this group.\n")
		return
	}
	_, err := p.gg.GetGroupByID(cgs.GroupID)
	if err != nil {
		panic("gg.GetGroupByID failed.")
	}
	gmi := GroupMinerID{cgs.GroupID, si.GetID()}
	sign_pk := p.GetMemberSignPubKey(gmi) //取得消息发送方的组内签名公钥
	if sign_pk.IsValid() {                //该用户和我是同一组
		fmt.Printf("message sender's sign_pk=%v.\n", GetPubKeyPrefix(sign_pk))
		fmt.Printf("bCG::si info: id=%v, data hash=%v, sign=%v.\n",
			GetIDPrefix(si.GetID()), GetHashPrefix(si.DataHash), GetSignPrefix(si.DataSign))
		if si.VerifySign(sign_pk) { //消息合法
			fmt.Printf("message verify sign OK, find gid=%v blockContext...\n", GetIDPrefix(cgs.GroupID))
			bc = p.GetBlockContext(cgs.GroupID.GetHexString())
			if bc == nil {
				panic("ERROR, BlockContext = nil.")
			} else {
				if !bc.IsCasting() { //之前没有在铸块状态
					b := bc.BeingCastGroup(cgs.BlockHeight, cgs.PreTime, cgs.PreHash) //设置当前铸块高度
					first = true
					fmt.Printf("blockContext::BeingCastGroup result=%v, bc::status=%v.\n", b, bc.ConsensusStatus)
				} else {
					fmt.Printf("bc already in casting, height=%v...\n", bc.CastHeight)
				}
			}
		} else {
			fmt.Printf("ERROR, message verify failed, data_hash=%v.\n", GetHashPrefix(si.DataHash))
			panic("ERROR, message verify failed.")
		}
	} else {
		fmt.Printf("message sender's sign_pk not in joined groups, ignored.\n")
	}
	return
}

//收到成为当前铸块组消息
func (p *Processer) OnMessageCurrent(ccm ConsensusCurrentMessage) {
	p.castLock.Lock()
	defer p.castLock.Unlock()

	fmt.Printf("proc(%v) begin OMCur, sender=%v, time=%v, height=%v...\n", p.getPrefix(),
		GetIDPrefix(ccm.SI.GetID()), time.Now().Format(time.Stamp), ccm.BlockHeight)
	var gid groupsig.ID
	if gid.Deserialize(ccm.GroupID) != nil {
		panic("Processer::OMCur failed, reason=group id Deserialize.")
	}

	//如果是自己发的, 不处理
	if p.GetMinerID().IsEqual(ccm.SI.SignMember) {
		fmt.Printf("OMC receive self msg, ingore! \n")
		return
	}
	//
	//topBH := p.MainChain.QueryTopBlock()
	//if topBH.Height == ccm.BlockHeight && topBH.PreHash == ccm.PreHash {	//已经在链上了
	//	fmt.Printf("OMCur block height already on chain!, ingore!, topBlockHeight=%v, ccm.Height=%v, topHash=%v, ccm.PreHash=%v", topBH.Height, ccm.BlockHeight, GetHashPrefix(topBH.PreHash), GetHashPrefix(ccm.PreHash))
	//	return
	//}


	var cgs CastGroupSummary
	cgs.GroupID = gid
	cgs.PreHash = ccm.PreHash
	cgs.PreTime = ccm.PreTime
	cgs.BlockHeight = ccm.BlockHeight
	fmt.Printf("OMCur::SIGN_INFO: id=%v, data hash=%v, sign=%v.\n",
		GetIDPrefix(ccm.SI.GetID()), GetHashPrefix(ccm.SI.DataHash), GetSignPrefix(ccm.SI.DataSign))
	bc, first := p.beingCastGroup(cgs, ccm.SI)
	if bc == nil {
		fmt.Printf("proc(%v) OMCur can't get valid bc, ignore message.\n", p.getPrefix())
		return
	}
	fmt.Printf("OMCur after beingCastGroup, bc.height=%v, first=%v, status=%v.\n", bc.CastHeight, first, bc.ConsensusStatus)
	if bc != nil {
		switched := bc.Switch2Height(cgs)
		if !switched {
			fmt.Printf("bc::Switch2Height failed, ignore message.\n")
			return
		}
		//if first { //第一次收到“当前组成为铸块组”消息
		//	ccm_local := ccm
		//	ccm_local.GenSign(SecKeyInfo{p.GetMinerID(), p.getSignKey(gid)})
		//	if locked {
		//		p.castLock.Unlock()
		//		locked = false
		//	}
		//	if !PROC_TEST_MODE {
		//		fmt.Printf("call network service SendCurrentGroupCast...\n")
		//		SendCurrentGroupCast(&ccm_local)
		//	} else {
		//		fmt.Printf("proc(%v) OMCur BEGIN SEND OnMessageCurrent 2 ALL PROCS...\n", p.getPrefix())
		//		for _, v := range p.GroupProcs {
		//			v.OnMessageCurrent(ccm_local)
		//		}
		//	}
		//}
	}
	fmt.Printf("proc(%v) end OMCur, time=%v.\n", p.getPrefix(), time.Now().Format(time.Stamp))
	return
}

//收到组内成员的出块消息，出块人（KING）用组分片密钥进行了签名
//有可能没有收到OnMessageCurrent就提前接收了该消息（网络时序问题）
func (p *Processer) OnMessageCast(ccm ConsensusCastMessage) {
	p.castLock.Lock()
	locked := true


	var g_id groupsig.ID
	if g_id.Deserialize(ccm.BH.GroupId) != nil {
		panic("OMC Deserialize group_id failed")
	}
	var castor groupsig.ID
	castor.Deserialize(ccm.BH.Castor)
	fmt.Printf("proc(%v) begin OMC, group=%v, sender=%v, height=%v, qn=%v, castor=%v...\n", p.getPrefix(),
		GetIDPrefix(g_id), GetIDPrefix(ccm.SI.GetID()), ccm.BH.Height, ccm.BH.QueueNumber, GetIDPrefix(castor))

	//如果是自己发的, 不处理
	if p.GetMinerID().IsEqual(ccm.SI.SignMember) {
		fmt.Printf("OMC receive self msg, ingore! \n")
		p.castLock.Unlock()
		return
	}

	fmt.Printf("OMCCCCC message bh %v\n", ccm.BH)
	fmt.Printf("OMCCCCC chain top bh %v\n", p.MainChain.QueryTopBlock())

	fmt.Printf("proc(%v) OMC rece hash=%v.\n", p.getPrefix(), GetHashPrefix(ccm.SI.DataHash))
	var cgs CastGroupSummary
	cgs.BlockHeight = ccm.BH.Height
	cgs.GroupID = g_id
	cgs.PreHash = ccm.BH.PreHash
	cgs.PreTime = ccm.BH.PreTime
	bc, first := p.beingCastGroup(cgs, ccm.SI)
	if bc == nil {
		if locked {
			p.castLock.Unlock()
			locked = false
		}
		fmt.Printf("proc(%v) OMC can't get valid bc, ignore message.\n", p.getPrefix())
		return
	}
	fmt.Printf("OMC after beingCastGroup, bc.Height=%v, first=%v, status=%v.\n", bc.CastHeight, first, bc.ConsensusStatus)
	switched := bc.Switch2Height(cgs)
	if !switched {
		fmt.Printf("bc::Switch2Height failed, ignore message.\n")
		if locked {
			p.castLock.Unlock()
			locked = false
		}
		return
	}

	if !bc.IsCasting() { //当前没有在组铸块中
		if locked {
			p.castLock.Unlock()
			locked = false
		}
		fmt.Printf("proc(%v) OMC failed, group not in cast.\n", p.getPrefix())
		return
	}

	var ccr int8
	var lost_trans_list []common.Hash

	slot := bc.getSlotByQN(int64(ccm.BH.QueueNumber))
	if slot != nil {
		if slot.IsFailed() {
			if locked {
				p.castLock.Unlock()
				locked = false
			}
			fmt.Printf("proc(%v) OMC slot irreversible failed, ignore message.\n", p.getPrefix())
			return
		}
		if slot.isAllTransExist() { //所有交易都已本地存在
			ccr = 0
		} else {
			if !PROC_TEST_MODE {
				lost_trans_list, ccr, _, _ = p.MainChain.VerifyCastingBlock(ccm.BH)
				fmt.Printf("proc(%v) OMC chain check result=%v, lost_count=%v.\n", p.getPrefix(), ccr, len(lost_trans_list))
				//slot.InitLostingTrans(lost_trans_list)
			}
		}
	}

	cs := GenConsensusSummary(ccm.BH)
	switch ccr {
	case 0: //主链验证通过
		n := bc.UserCasted(ccm.BH, ccm.SI)
		fmt.Printf("proc(%v) OMC UserCasted result=%v.\n", p.getPrefix(), n)
		switch n {
		case CBMR_THRESHOLD_SUCCESS:
			fmt.Printf("proc(%v) OMC msg_count reach threshold!\n", p.getPrefix())
			b := bc.VerifyGroupSign(cs, p.getGroupPubKey(g_id))
			fmt.Printf("proc(%v) OMC VerifyGroupSign=%v.\n", p.getPrefix(), b)
			if b { //组签名验证通过
				slot := bc.getSlotByQN(cs.QueueNumber)
				if slot == nil {
					panic("getSlotByQN return nil.")
				} else {
					sign := slot.GetGroupSign()
					gpk := p.getGroupPubKey(g_id)
					if !slot.VerifyGroupSign(gpk) {
						fmt.Printf("OMC group pub key local check failed, gpk=%v, sign=%v, hash in slot=%v, hash in bh=%v.\n",
							GetPubKeyPrefix(gpk), GetSignPrefix(sign), GetHashPrefix(slot.BH.Hash), GetHashPrefix(ccm.BH.Hash))
						panic("OMC group pub key local check failed")
					} else {
						fmt.Printf("OMC group pub key local check OK, gpk=%v, sign=%v, hash in slot=%v, hash in bh=%v.\n",
							GetPubKeyPrefix(gpk), GetSignPrefix(sign), GetHashPrefix(slot.BH.Hash), GetHashPrefix(ccm.BH.Hash))
					}
					ccm.BH.Signature = sign.Serialize()
					fmt.Printf("OMC BH hash=%v, update group sign data=%v.\n", GetHashPrefix(ccm.BH.Hash), GetSignPrefix(sign))

				}
				fmt.Printf("proc(%v) OMC SUCCESS CAST GROUP BLOCK, height=%v, qn=%v!!!\n", p.getPrefix(), ccm.BH.Height, cs.QueueNumber)
				p.SuccessNewBlock(&ccm.BH, g_id) //上链和组外广播
			} else {
				panic("proc OMC VerifyGroupSign failed.")
			}
		case CBMR_PIECE:
			var cvm ConsensusVerifyMessage
			cvm.BH = ccm.BH
			//cvm.GroupID = g_id
			cvm.GenSign(SecKeyInfo{p.GetMinerID(), p.getSignKey(g_id)})
			if ccm.SI.SignMember.GetHexString() == p.GetMinerID().GetHexString() { //local node is KING
				equal := cvm.SI.IsEqual(ccm.SI)
				if !equal {
					fmt.Printf("proc(%v) cur prov is KING, but cast sign and verify sign diff.\n", p.getPrefix())
					fmt.Printf("proc(%v) cast sign: id=%v, hash=%v, sign=%v.\n", p.getPrefix(), GetIDPrefix(ccm.SI.SignMember), ccm.SI.DataHash.Hex(), ccm.SI.DataSign.GetHexString())
					fmt.Printf("proc(%v) verify sign: id=%v, hash=%v, sign=%v.\n", p.getPrefix(), GetIDPrefix(cvm.SI.SignMember), cvm.SI.DataHash.Hex(), cvm.SI.DataSign.GetHexString())
					panic("cur prov is KING, but cast sign and verify sign diff.")
				}
			}
			if locked {
				p.castLock.Unlock()
				locked = false
			}
			if !PROC_TEST_MODE {
				fmt.Printf("call network service SendVerifiedCast...\n")
				SendVerifiedCast(&cvm)
			} else {
				fmt.Printf("proc(%v) OMC BEGIN SEND OnMessageVerify 2 ALL PROCS...\n", p.getPrefix())
				for _, v := range p.GroupProcs {
					v.OnMessageVerify(cvm)
				}
			}

		}
	case 1: //本地交易缺失
		n := bc.UserCasted(ccm.BH, ccm.SI)
		fmt.Printf("proc(%v) OMC UserCasted result=%v.\n", p.getPrefix(), n)
		switch n {
		case CBMR_THRESHOLD_SUCCESS:
			fmt.Printf("proc(%v) OMC msg_count reach threshold, but local missing trans, still waiting.\n", p.getPrefix())
		case CBMR_PIECE:
			fmt.Printf("proc(%v) OMC normal receive verify, but local missing trans, waiting.\n", p.getPrefix())
		}
	case -1:
		slot.statusChainFailed()
	}
	if locked {
		p.castLock.Unlock()
		locked = false
	}
	fmt.Printf("proc(%v) end OMC.\n", p.getPrefix())
	return
}

//收到组内成员的出块验证通过消息（组内成员消息）
func (p *Processer) OnMessageVerify(cvm ConsensusVerifyMessage) {
	p.castLock.Lock()
	locked := true
	var g_id groupsig.ID
	if g_id.Deserialize(cvm.BH.GroupId) != nil {
		panic("OMV Deserialize group_id failed")
	}
	var castor groupsig.ID
	castor.Deserialize(cvm.BH.Castor)
	fmt.Printf("proc(%v) begin OMV, group=%v, sender=%v, height=%v, qn=%v, rece hash=%v castor=%v...\n", p.getPrefix(),
		GetIDPrefix(g_id), GetIDPrefix(cvm.SI.GetID()), cvm.BH.Height, cvm.BH.QueueNumber, cvm.SI.DataHash.Hex(), GetIDPrefix(castor))

	//如果是自己发的, 不处理
	if p.GetMinerID().IsEqual(cvm.SI.SignMember) {
		fmt.Printf("OMC receive self msg, ingore! \n")
		p.castLock.Unlock()
		return
	}

	fmt.Printf("OMVVVVVV message bh %v\n", cvm.BH)
	fmt.Printf("OMVVVVVV message bh hash %v\n", GetHashPrefix(cvm.BH.Hash))
	fmt.Printf("OMVVVVVV chain top bh %v\n", p.MainChain.QueryTopBlock())

	var cgs CastGroupSummary
	cgs.BlockHeight = cvm.BH.Height
	cgs.GroupID = g_id
	cgs.PreHash = cvm.BH.PreHash
	cgs.PreTime = cvm.BH.PreTime
	bc, first := p.beingCastGroup(cgs, cvm.SI)
	if bc == nil {
		if locked {
			p.castLock.Unlock()
			locked = false
		}
		fmt.Printf("proc(%v) OMV can't get valid bc, ignore message.\n", p.getPrefix())
		return
	}
	fmt.Printf("OMV after beingCastGroup, bc.Height=%v, first=%v, status=%v.\n", bc.CastHeight, first, bc.ConsensusStatus)

	switched := bc.Switch2Height(cgs)
	if !switched {
		fmt.Printf("bc::Switch2Height failed, ignore message.\n")
		if locked {
			p.castLock.Unlock()
			locked = false
		}
		return
	}

	if !bc.IsCasting() { //当前没有在组铸块中
		if locked {
			p.castLock.Unlock()
			locked = false
		}
		fmt.Printf("proc(%v) OMV failed, group not in cast, ignore message.\n", p.getPrefix())
		return
	}

	var ccr int8 = 0
	var lost_trans_list []common.Hash

	slot := bc.getSlotByQN(int64(cvm.BH.QueueNumber))
	if slot != nil {
		if slot.IsFailed() {
			if locked {
				p.castLock.Unlock()
				locked = false
			}
			fmt.Printf("proc(%v) OMV slot irreversible failed, ignore message.\n", p.getPrefix())
			return
		}
		if slot.isAllTransExist() { //所有交易都已本地存在
			ccr = 0
		} else {
			if !PROC_TEST_MODE {
				lost_trans_list, ccr, _, _ = p.MainChain.VerifyCastingBlock(cvm.BH)
				fmt.Printf("proc(%v) OMV chain check result=%v, lost_trans_count=%v.\n", p.getPrefix(), ccr, len(lost_trans_list))
				//slot.LostingTrans(lost_trans_list)
			}
		}
	}

	cs := GenConsensusSummary(cvm.BH)
	switch ccr { //链检查结果
	case 0: //验证通过
		n := bc.UserVerified(cvm.BH, cvm.SI)
		fmt.Printf("proc(%v) OMV UserVerified result=%v.\n", p.getPrefix(), n)
		switch n {
		case CBMR_THRESHOLD_SUCCESS:
			fmt.Printf("proc(%v) OMV msg_count reach threshold!\n", p.getPrefix())
			b := bc.VerifyGroupSign(cs, p.getGroupPubKey(g_id))
			fmt.Printf("proc(%v) OMV VerifyGroupSign=%v.\n", p.getPrefix(), b)
			if b { //组签名验证通过
				fmt.Printf("proc(%v) OMV SUCCESS CAST GROUP BLOCK, height=%v, qn=%v.!!!\n", p.getPrefix(), cvm.BH.Height, cvm.BH.QueueNumber)
				slot := bc.getSlotByQN(int64(cvm.BH.QueueNumber))
				if slot == nil {
					panic("getSlotByQN return nil.")
				} else {
					sign := slot.GetGroupSign()

					gpk := p.getGroupPubKey(g_id)
					if !slot.VerifyGroupSign(gpk) {
						fmt.Printf("OMC group pub key local check failed, gpk=%v, sign=%v, hash in slot=%v, hash in bh=%v.\n",
							GetPubKeyPrefix(gpk), GetSignPrefix(sign), GetHashPrefix(slot.BH.Hash), GetHashPrefix(cvm.BH.Hash))
						panic("OMC group pub key local check failed")
					} else {
						fmt.Printf("OMC group pub key local check OK, gpk=%v, sign=%v, hash in slot=%v, hash in bh=%v.\n",
							GetPubKeyPrefix(gpk), GetSignPrefix(sign), GetHashPrefix(slot.BH.Hash), GetHashPrefix(cvm.BH.Hash))
					}
					cvm.BH.Signature = sign.Serialize()
					fmt.Printf("OMV BH hash=%v, update group sign data=%v.\n", GetHashPrefix(cvm.BH.Hash), GetSignPrefix(sign))
				}
				fmt.Printf("proc(%v) OMV SUCCESS CAST GROUP BLOCK!!!\n", p.getPrefix())
				p.SuccessNewBlock(&cvm.BH, g_id) //上链和组外广播

			} else {
				if locked {
					p.castLock.Unlock()
					locked = false
				}
				panic("proc OMV VerifyGroupSign failed.")
			}
		case CBMR_PIECE:
			var send_message ConsensusVerifyMessage
			send_message.BH = cvm.BH
			//send_message.GroupID = g_id
			send_message.GenSign(SecKeyInfo{p.GetMinerID(), p.getSignKey(g_id)})
			if locked {
				p.castLock.Unlock()
				locked = false
			}
			if !PROC_TEST_MODE {
				fmt.Printf("call network service SendVerifiedCast...\n")
				SendVerifiedCast(&send_message)
			} else {
				fmt.Printf("proc(%v) OMV BEGIN SEND OnMessageVerify 2 ALL PROCS...\n", p.getPrefix())
				for _, v := range p.GroupProcs {
					v.OnMessageVerify(send_message)
				}
			}
		}
	case 1: //本地交易缺失
		n := bc.UserVerified(cvm.BH, cvm.SI)
		fmt.Printf("proc(%v) OnMessageVerify UserVerified result=%v.\n", p.getPrefix(), n)
		switch n {
		case CBMR_THRESHOLD_SUCCESS:
			fmt.Printf("proc(%v) OMV msg_count reach threshold, but local missing trans, still waiting.\n", p.getPrefix())
		case CBMR_PIECE:
			fmt.Printf("proc(%v) OMV normal receive verify, but local missing trans, waiting.\n", p.getPrefix())
		}
	case -1: //不可逆异常
		slot.statusChainFailed()
	}
	if locked {
		p.castLock.Unlock()
		locked = false
	}
	fmt.Printf("proc(%v) end OMV.\n", p.getPrefix())
	return
}

//检查自身所在的组（集合）是否成为当前铸块组，如是，则启动相应处理
//sign：组签名
func (p *Processer) checkCastingGroup(groupId groupsig.ID, sign groupsig.Signature, height uint64, t time.Time, h common.Hash) (bool, ConsensusCurrentMessage) {
	var casting bool
	var ccm ConsensusCurrentMessage
	sign_hash := sign.GetHash()
	fmt.Printf("cCG pre_block group sign hash=%v, find next group...\n", GetHashPrefix(sign_hash))
	next_group, err := p.gg.SelectNextGroup(sign_hash,height+1) //查找下一个铸块组
	if err == nil {
		fmt.Printf("cCG next cast group=%v.\n", GetIDPrefix(next_group))
		bc := p.GetBlockContext(next_group.GetHexString())
		if p.IsMinerGroup(next_group) { //自身属于下一个铸块组
			fmt.Printf("IMPORTANT : OMB local miner belong next cast group!.\n")
			if bc == nil {
				panic("current proc belong next cast group, but GetBlockContext=nil.")
			}
			bc.BeingCastGroup(height, t, h)
			ccm.GroupID = next_group.Serialize() //groupId.Serialize()
			ccm.BlockHeight = height + 1
			ccm.PreHash = h
			ccm.PreTime = t
			ski := SecKeyInfo{p.GetMinerID(), p.getSignKey(next_group)}
			temp_spk := groupsig.NewPubkeyFromSeckey(ski.SK)
			if temp_spk == nil {
				panic("ccg spk nil failed")
			} else {
				fmt.Printf("id=%v, sign_pk=%v notify group members being current.\n", GetIDPrefix(ski.GetID()), GetPubKeyPrefix(*temp_spk))
			}
			ccm.GenSign(ski)
			casting = true
			fmt.Printf("cCG: id=%v, sign_sk=%v, data hash=%v.\n", GetIDPrefix(ccm.SI.GetID()), GetSecKeyPrefix(ski.SK), GetHashPrefix(ccm.SI.DataHash))
		} else {
			fmt.Printf("current proc not in next group.\n")
			if bc.IsCasting() {
				bc.Reset()
			}
		}
	} else {
		panic("find next cast group failed.")
	}
	return casting, ccm
}

//收到铸块上链消息(组外矿工节点处理)
func (p *Processer) OnMessageBlock(cbm ConsensusBlockMessage) *core.Block {
	p.castLock.Lock()
	locked := true
	var g_id groupsig.ID
	if g_id.Deserialize(cbm.Block.Header.GroupId) != nil {
		panic("OMB Deserialize group_id failed")
	}
	fmt.Printf("proc(%v) begin OMB, group=%v(bh gid=%v), sender=%v, height=%v, qn=%v...\n", p.getPrefix(),
		GetIDPrefix(cbm.GroupID), GetIDPrefix(g_id), GetIDPrefix(cbm.SI.GetID()), cbm.Block.Header.Height, cbm.Block.Header.QueueNumber)

	if p.GetMinerID().IsEqual(cbm.SI.SignMember) {
		p.castLock.Unlock()
		fmt.Println("OMB receive self msg, ingored!")
		return &cbm.Block
	}


	var block *core.Block
	//bc := p.GetBlockContext(cbm.GroupID.GetHexString())
	if p.isBHCastLegal(*cbm.Block.Header, cbm.SI) { //铸块头合法

		//上链
		//onchain := p.MainChain.AddBlockOnChain(&cbm.Block)
		onchain, future := p.AddOnChain(&cbm.Block)
		fmt.Printf("OMB onchain result %v, %v\n", onchain, future)


	} else {
		//丢弃该块
		fmt.Printf("OMB received invalid new block, height = %v.\n", cbm.Block.Header.Height)
	}

	//收到不合法的块也要做一次检查自己是否属于下一个铸块组, 否则会形成死循环, 没有组出块
	preHeader := p.MainChain.QueryTopBlock()
	if preHeader == nil {
		panic("cannot find top block header!")
	}

	var sign groupsig.Signature
	if sign.Deserialize(preHeader.Signature) != nil {
		panic("OMB group sign Deserialize failed.")
	}
	casting, ccm := p.checkCastingGroup(cbm.GroupID, sign, preHeader.Height, preHeader.CurTime, preHeader.Hash)
	if locked {
		p.castLock.Unlock()
		locked = false
	}

	if casting {
		fmt.Printf("OMB current proc being casting group...\n")
		fmt.Printf("OMB call network service SendCurrentGroupCast...\n")
		SendCurrentGroupCast(&ccm) //通知所有组员“我们组成为当前铸块组”
	}
	block = &cbm.Block //返回成功的块

	fmt.Printf("proc(%v) end OMB, group=%v, sender=%v...\n", p.getPrefix(), GetIDPrefix(cbm.GroupID), GetIDPrefix(cbm.SI.GetID()))
	return block
}

//新的交易到达通知（用于处理大臣验证消息时缺失的交易）
func (p *Processer) OnMessageNewTransactions(ths []common.Hash) {
	p.castLock.Lock()
	locked := true
	fmt.Printf("proc(%v) begin OMNT, trans count=%v...\n", p.getPrefix(), len(ths))
	if len(ths) > 0 {
		fmt.Printf("proc(%v) OMNT, first trans=%v.\n", p.getPrefix(), ths[0].Hex())
	}

	bc := p.GetCastingBC() //to do ：实际的场景，可能会返回多个bc，需要处理。（一个矿工加入多个组，考虑现在测试的极端情况，矿工加入了连续出块的2个组）
	if bc != nil {
		fmt.Printf("OMNT, bc height=%v, status=%v. slots_info=%v.\n", bc.CastHeight, bc.ConsensusStatus, bc.PrintSlotInfo())
		qns := bc.ReceTrans(ths)
		fmt.Printf("OMNT, bc.ReceTrans result qns_count=%v.\n", len(qns))
		for _, v := range qns { //对不再缺失交易集的插槽处理
			slot := bc.getSlotByQN(int64(v))
			if slot != nil {
				lost_trans_list, result, _, _ := p.MainChain.VerifyCastingBlock(slot.BH)
				fmt.Printf("OMNT slot (qn=%v) info : lost_trans=%v, mainchain check result=%v.\n", v, len(lost_trans_list), result)
				if len(lost_trans_list) > 0 {
					if locked {
						p.castLock.Unlock()
						locked = false
					}
					panic("OMNT still losting trans on main chain, ERROR.")
				}
				switch result {
				case 0: //验证通过
					var send_message ConsensusVerifyMessage
					send_message.BH = slot.BH
					//send_message.GroupID = bc.MinerID.gid
					send_message.GenSign(SecKeyInfo{p.GetMinerID(), p.getSignKey(bc.MinerID.gid)})
					if locked {
						p.castLock.Unlock()
						locked = false
					}
					if !PROC_TEST_MODE {
						fmt.Printf("call network service SendVerifiedCast...\n")
						SendVerifiedCast(&send_message)
					} else {
						fmt.Printf("proc(%v) OMV BEGIN SEND OnMessageVerify 2 ALL PROCS...\n", p.getPrefix())
						for _, v := range p.GroupProcs {
							v.OnMessageVerify(send_message)
						}
					}
				case 1:
					panic("Processer::OMNT failed, check xiaoxiong's src code.")
				case -1:
					fmt.Printf("OMNT set slot (qn=%v) failed irreversible.\n", v)
					slot.statusChainFailed()
				}
			} else {
				fmt.Printf("OMNT failed, after ReceTrans slot %v is nil.\n", v)
				panic("OMNT failed, slot is nil.")
			}
		}
	} else {
		fmt.Printf("OMNT, current proc not in casting, ignore OMNT message.\n")
	}
	if locked {
		p.castLock.Unlock()
		locked = false
	}
	fmt.Printf("proc(%v) end OMNT.\n", p.getPrefix())
	return
}

//在某个区块高度的QN值成功出块，保存上链，向组外广播
//同一个高度，可能会因QN不同而多次调用该函数
//但一旦低的QN出过，就不该出高的QN。即该函数可能被多次调用，但是调用的QN值越来越小
func (p *Processer) SuccessNewBlock(bh *core.BlockHeader, gid groupsig.ID) {
	if bh == nil {
		panic("SuccessNewBlock arg failed.")
	}
	fmt.Printf("proc(%v) begin SuccessNewBlock, group=%v, qn=%v...\n", p.getPrefix(), GetIDPrefix(gid), bh.QueueNumber)
	bc := p.GetBlockContext(gid.GetHexString())
	block := p.MainChain.GenerateBlock(*bh)
	if block == nil {
		panic("core.GenerateBlock failed.")
	}
	if !PROC_TEST_MODE {
		r, _ := p.AddOnChain(block)
		if r == 2 || (r != 0 && r != 1) {	//分叉调整或 上链失败都不走下面的逻辑
			return
		}

		//r := p.MainChain.AddBlockOnChain(block)
		//fmt.Printf("AddBlockOnChain header %v \n", block.Header)
		//fmt.Printf("QueryTopBlock header %v \n", p.MainChain.QueryTopBlock())
		//fmt.Printf("proc(%v) core.AddBlockOnChain, height=%v, qn=%v, result=%v.\n", p.getPrefix(), block.Header.Height, block.Header.QueueNumber, r)
		//if r == 0 || r == 1 {	//上链成功
		//
		//} else if r == 2 {	//分叉调整, 未上链
		//	return
		//} else { //上链失败
		//	//可能多次掉次方法, 要区分是否同一个块上链失败
		//	panic("core.AddBlockOnChain failed.")
		//}
	}
	bc.CastedUpdateStatus(uint(bh.QueueNumber))
	bc.SignedUpdateMinQN(uint(bh.QueueNumber))
	var cbm ConsensusBlockMessage
	cbm.Block = *block
	cbm.GroupID = gid
	ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
	cbm.GenSign(ski)
	if !PROC_TEST_MODE {
		fmt.Printf("call network service BroadcastNewBlock...\n")
		BroadcastNewBlock(&cbm)
	}
	fmt.Printf("proc(%v) end SuccessNewBlock.\n", p.getPrefix())
	return
}

//检查是否轮到自己出块
func (p *Processer) CheckCastRoutine(gid groupsig.ID, king_index int32, qn int64, height uint64) {
	p.castLock.Lock()
	defer p.castLock.Unlock()
	fmt.Printf("prov(%v) begin CheckCastRoutine, gid=%v, king_index=%v, qn=%v, height=%v.\n", p.getPrefix(), GetIDPrefix(gid), king_index, qn, height)
	if king_index < 0 || qn < 0 {
		return
	}
	sgi := p.getGroup(gid)
	pos := sgi.GetMinerPos(p.GetMinerID()) //取得当前节点在组中的排位
	fmt.Printf("time=%v, Current KING=%v.\n", time.Now().Format(time.Stamp), GetIDPrefix(sgi.GetCastor(int(king_index))))
	fmt.Printf("Current node=%v, pos_index in group=%v.\n", p.getPrefix(), pos)
	if sgi.GetCastor(int(king_index)).GetHexString() == p.GetMinerID().GetHexString() { //轮到自己铸块
		fmt.Printf("curent node IS KING!\n")
		if !p.sci.FindQN(height, uint(qn)) { //在该高度该QN，自己还没铸过快
			head := p.castBlock(gid, height, qn) //铸块
			if head != nil {
				p.sci.AddQN(height, uint(qn))
			}
		} else {
			fmt.Printf("In height=%v, qn=%v current node already casted.", height, qn)
		}
	}
	return
}

///////////////////////////////////////////////////////////////////////////////
//组初始化相关消息
//组初始化的相关消息都用（和组无关的）矿工ID和公钥验签

func (p *Processer) OnMessageGroupInit(grm ConsensusGroupRawMessage) {
	fmt.Printf("proc(%v) begin OMGI, sender=%v, dummy_gid=%v...\n", p.getPrefix(), GetIDPrefix(grm.SI.GetID()), GetIDPrefix(grm.GI.DummyID))
	p.initLock.Lock()
	locked := true

	//to do : 从链上检查消息发起人（父亲组成员）是否有权限发该消息（鸠兹）
	sgi_info := NewSGIFromRawMessage(grm)
	//p.gg.AddDummyGroup(sgi)
	p.gg.ngg.addInitingGroup(CreateInitingGroup(sgi_info))

	//非组内成员不走后续流程
	if !sgi_info.MemExist(p.GetMinerID()) {
		p.initLock.Unlock()
		locked = false
		return
	}

	gc := p.jgs.ConfirmGroupFromRaw(grm, p.mi)
	if gc == nil {
		panic("Processer::OMGI failed, ConfirmGroupFromRaw return nil.")
	}
	gs := gc.GetGroupStatus()
	fmt.Printf("OMGI joining group(%v) status=%v.\n", GetIDPrefix(grm.GI.DummyID), gs)
	if gs == GIS_RAW {
		fmt.Printf("begin GenSharePieces in OMGI...\n")
		shares := gc.GenSharePieces() //生成秘密分享
		fmt.Printf("proc(%v) end GenSharePieces in OMGI, piece size=%v.\n", p.getPrefix(), len(shares))

		if locked {
			p.initLock.Unlock()
			locked = false
		}

		var spm ConsensusSharePieceMessage
		spm.GISHash = grm.GI.GenHash()
		spm.DummyID = grm.GI.DummyID
		ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
		spm.SI.SignMember = p.GetMinerID()
		for id, piece := range shares {
			if id != "0x0" && piece.IsValid() {
				spm.Dest.SetHexString(id)
				spm.Share = piece
				sb := spm.GenSign(ski)
				fmt.Printf("OMGI spm.GenSign result=%v.\n", sb)
				fmt.Printf("OMGI piece to ID(%v), share=%v, pub=%v.\n", GetIDPrefix(spm.Dest), GetSecKeyPrefix(spm.Share.Share), GetPubKeyPrefix(spm.Share.Pub))
				if !PROC_TEST_MODE {
					fmt.Printf("call network service SendKeySharePiece...\n")
					SendKeySharePiece(spm)
				} else {
					fmt.Printf("test mode, call OMSP direct...\n")
					dest_proc, ok := p.GroupProcs[spm.Dest.GetHexString()]
					if ok {
						dest_proc.OnMessageSharePiece(spm)
					} else {
						panic("ERROR, dest proc not found!\n")
					}
				}

			} else {
				panic("GenSharePieces data not IsValid.\n")
			}
		}
		fmt.Printf("end GenSharePieces.\n")
	} else {
		fmt.Printf("group(%v) status=%v, ignore init message.\n", GetIDPrefix(grm.GI.DummyID), gs)
	}
	if locked {
		p.initLock.Unlock()
		locked = false
	}
	fmt.Printf("proc(%v) end OMGI, sender=%v.\n", p.getPrefix(), GetIDPrefix(grm.SI.GetID()))
	return
}

//收到组内成员发给我的秘密分享片段消息
func (p *Processer) OnMessageSharePiece(spm ConsensusSharePieceMessage) {
	fmt.Printf("proc(%v)begin Processer::OMSP, sender=%v...\n", p.getPrefix(), GetIDPrefix(spm.SI.GetID()))
	p.initLock.Lock()
	locked := true

	gc := p.jgs.ConfirmGroupFromPiece(spm, p.mi)
	if gc == nil {
		if locked {
			p.initLock.Unlock()
			locked = false
		}
		panic("OMSP failed, receive SHAREPIECE msg but gc=nil.\n")
		return
	}
	result := gc.PieceMessage(spm)
	fmt.Printf("proc(%v) OMSP after gc.PieceMessage, piecc_count=%v, gc result=%v.\n", p.getPrefix(), p.piece_count, result)
	p.piece_count++
	if result < 0 {
		panic("OMSP failed, gc.PieceMessage result less than 0.\n")
	}
	if result == 1 { //已聚合出签名私钥
		jg := gc.GetGroupInfo()
		//这时还没有所有组成员的签名公钥
		if jg.GroupPK.IsValid() && jg.SignKey.IsValid() {
			fmt.Printf("OMSP SUCCESS gen sign sec key and group pub key, msk=%v, gpk=%v.\n", GetSecKeyPrefix(jg.SignKey), GetPubKeyPrefix(jg.GroupPK))
			{
				ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
				var msg ConsensusSignPubKeyMessage
				msg.GISHash = spm.GISHash
				msg.DummyID = spm.DummyID
				msg.SignPK = *groupsig.NewPubkeyFromSeckey(jg.SignKey)
				msg.GenGISSign(jg.SignKey)
				if !msg.VerifyGISSign(msg.SignPK) {
					panic("verify GISSign with group member sign pub key failed.")
				}

				msg.GenSign(ski)
				//todo : 组内广播签名公钥
				fmt.Printf("OMSP send sign pub key to group members, spk=%v...\n", GetPubKeyPrefix(msg.SignPK))
				if locked {
					p.initLock.Unlock()
					locked = false
				}
				if !PROC_TEST_MODE {
					fmt.Printf("OMSP call network service SendSignPubKey...\n")
					SendSignPubKey(msg)
				} else {
					fmt.Printf("test mode, call OnMessageSignPK direct...\n")
					for _, proc := range p.GroupProcs {
						proc.OnMessageSignPK(msg)
					}
				}
			}

		} else {
			panic("Processer::OMSP failed, aggr key error.")
		}
	}
	if locked {
		p.initLock.Unlock()
		locked = false
	}
	fmt.Printf("prov(%v) end OMSP, sender=%v.\n", p.getPrefix(), GetIDPrefix(spm.SI.GetID()))
	return
}

//收到组内成员发给我的组成员签名公钥消息
func (p *Processer) OnMessageSignPK(spkm ConsensusSignPubKeyMessage) {
	fmt.Printf("proc(%v) begin OMSPK, sender=%v, dummy_gid=%v...\n", p.getPrefix(), GetIDPrefix(spkm.SI.GetID()), GetIDPrefix(spkm.DummyID))
	p.initLock.Lock()
	locked := true

	/* 待小熊增加GISSign成员的流化后打开
	if !spkm.VerifyGISSign(spkm.SignPK) {
		panic("OMSP verify GISSign with sign pub key failed.")
	}
	*/

	gc := p.jgs.GetGroup(spkm.DummyID)
	if gc == nil {
		if locked {
			p.initLock.Unlock()
			locked = false
		}
		fmt.Printf("OMSPK failed, local node not found joining group with dummy id=%v.\n", GetIDPrefix(spkm.DummyID))
		return
	}
	fmt.Printf("before SignPKMessage already exist mem sign pks=%v.\n", len(gc.node.m_sign_pks))
	result := gc.SignPKMessage(spkm)
	fmt.Printf("after SignPKMessage exist mem sign pks=%v, result=%v.\n", len(gc.node.m_sign_pks), result)
	if result == 1 { //收到所有组成员的签名公钥
		jg := gc.GetGroupInfo()
		if jg.GroupID.IsValid() && jg.SignKey.IsValid() {
			p.addInnerGroup(jg)
			fmt.Printf("SUCCESS INIT GROUP: gid=%v, gpk=%v.\n", GetIDPrefix(jg.GroupID), GetPubKeyPrefix(jg.GroupPK))
			{
				var msg ConsensusGroupInitedMessage
				ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
				msg.GI.GIS = gc.gis
				msg.GI.GroupID = jg.GroupID
				msg.GI.GroupPK = jg.GroupPK
				var mems []PubKeyInfo
				for _, v := range gc.mems {
					mems = append(mems, v)
				}
				msg.GI.Members = mems
				pTop := p.MainChain.QueryTopBlock()
				if 0 == pTop.Height {
					msg.GI.BeginHeight = 1
				} else {
					msg.GI.BeginHeight = pTop.Height + uint64(GROUP_INIT_IDLE_HEIGHT)
				}
				msg.GenSign(ski)
				if locked {
					p.initLock.Unlock()
					locked = false
				}
				if !PROC_TEST_MODE {
					//组写入组链 add by 小熊
					members := make([]core.Member, 0)
					for _, miner := range mems {
						member := core.Member{Id: miner.ID.Serialize(), PubKey: miner.PK.Serialize()}
						members = append(members, member)
					}
					group := core.Group{Id: msg.GI.GroupID.Serialize(), Members: members, PubKey: msg.GI.GroupPK.Serialize(), Parent: msg.GI.GIS.ParentID.Serialize()}
					e := core.GroupChainImpl.AddGroup(&group, nil, nil)
					if e != nil {
						fmt.Printf("group inited add group error:%s\n", e.Error())
					} else {
						fmt.Printf("group inited add group success. count: %d, now: %d\n", core.GroupChainImpl.Count(), len(core.GroupChainImpl.GetAllGroupID()))
					}
					fmt.Printf("call network service BroadcastGroupInfo...\n")
					BroadcastGroupInfo(msg)
				} else {
					fmt.Printf("test mode, call OnMessageGroupInited direct...\n")
					for _, proc := range p.GroupProcs {
						proc.OnMessageGroupInited(msg)
					}
				}
			}
		} else {
			panic("Processer::OnMessageSharePiece failed, aggr key error.")
		}
	}
	if locked {
		p.initLock.Unlock()
		locked = false
	}
	fmt.Printf("proc(%v) end OMSPK, sender=%v, dummy gid=%v.\n", p.getPrefix(), GetIDPrefix(spkm.SI.GetID()), GetIDPrefix(spkm.DummyID))
	return
}

//全网节点收到某组已初始化完成消息（在一个时间窗口内收到该组51%成员的消息相同，才确认上链）
//最终版本修改为父亲节点进行验证（51%）和上链
//全网节点处理函数->to do : 调整为父亲组节点处理函数
func (p *Processer) OnMessageGroupInited(gim ConsensusGroupInitedMessage) {
	fmt.Printf("proc(%v) begin OMGIED, sender=%v, dummy_gid=%v, gid=%v, gpk=%v...\n", p.getPrefix(),
		GetIDPrefix(gim.SI.GetID()), GetIDPrefix(gim.GI.GIS.DummyID), GetIDPrefix(gim.GI.GroupID), GetPubKeyPrefix(gim.GI.GroupPK))
	p.initLock.Lock()
	defer p.initLock.Unlock()

	var ngmd NewGroupMemberData
	ngmd.h = gim.GI.GIS.GenHash()
	ngmd.gid = gim.GI.GroupID
	ngmd.gpk = gim.GI.GroupPK
	var mid GroupMinerID
	mid.gid = gim.GI.GIS.DummyID
	mid.uid = gim.SI.SignMember
	result := p.gg.GroupInitedMessage(mid, ngmd)
	p.inited_count++
	fmt.Printf("proc(%v) OMGIED gg.GroupInitedMessage result=%v, inited_count=%v.\n", p.getPrefix(), result, p.inited_count)
	if result < 0 {
		panic("OMGIED gg.GroupInitedMessage failed because of return value.")
	}
	switch result {
	case 1: //收到组内相同消息>=阈值，可上链
		fmt.Printf("OMGIED SUCCESS accept a new group, gid=%v, gpk=%v.\n", GetIDPrefix(gim.GI.GroupID), GetPubKeyPrefix(gim.GI.GroupPK))
		b := p.gg.AddGroup(gim.GI)
		fmt.Printf("OMGIED Add to Global static groups, result=%v, groups=%v.\n", b, p.gg.GetGroupSize())
		bc := new(BlockContext)
		bc.Init(GroupMinerID{gim.GI.GroupID, p.GetMinerID()})
		sgi, err := p.gg.GetGroupByID(gim.GI.GroupID)
		if err != nil {
			panic("OMGIED GetGroupByID failed.\n")
		}
		bc.pos = sgi.GetMinerPos(p.GetMinerID())
		fmt.Printf("OMGIED current ID in group pos=%v.\n", bc.pos)
		bc.Proc = p
		//to do:只有自己属于这个组的节点才需要调用AddBlockConext
		b = p.AddBlockContext(bc)
		fmt.Printf("(proc:%v) OMGIED Add BlockContext result = %v, bc_size=%v.\n", p.getPrefix(), b, len(p.bcs))
		//to do : 上链已初始化的组
		//to do ：从待初始化组中删除

		//检查是否写入配置文件
		if p.save_data == 1 {
			p.Save()
		}

		fmt.Printf("begin sleeping 5 seconds, now=%v...\n", time.Now().Format(time.Stamp))
		sleep_d, err := time.ParseDuration("5s")
		if err == nil {
			time.Sleep(sleep_d)
		} else {
			panic("time.ParseDuration 5s failed.")
		}
		fmt.Printf("end sleeping, now=%v.\n", time.Now().Format(time.Stamp))

		//拉取当前最高块
		if !PROC_TEST_MODE {
			top_bh := p.MainChain.QueryTopBlock()
			if top_bh == nil {
				panic("QueryTopBlock failed")
			} else {
				fmt.Printf("top height on chain=%v.\n", top_bh.Height)
			}
			var g_sign groupsig.Signature
			if g_sign.Deserialize(top_bh.Signature) != nil {
				panic("OMGIED group sign Deserialize failed.")
			}
			casting, ccm := p.checkCastingGroup(gim.GI.GroupID, g_sign, top_bh.Height, top_bh.CurTime, top_bh.Hash)
			fmt.Printf("checkCastingGroup, current proc being casting group=%v.", casting)
			if casting {
				fmt.Printf("OMB: id=%v, data hash=%v, sign=%v.\n",
					GetIDPrefix(ccm.SI.GetID()), GetHashPrefix(ccm.SI.DataHash), GetSignPrefix(ccm.SI.DataSign))
				fmt.Printf("OMB call network service SendCurrentGroupCast...\n")
				SendCurrentGroupCast(&ccm) //通知所有组员“我们组成为当前铸块组”
			}
		}
	case -1: //该组初始化异常，且无法恢复
		//to do : 从待初始化组中删除
	case 0:
		//继续等待下一包数据
	}
	fmt.Printf("proc(%v) end OMGIED, sender=%v...\n", p.getPrefix(), GetIDPrefix(gim.SI.GetID()))
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
	fmt.Printf("bh.Hash=%v.\n", GetHashPrefix(bh.Hash))
	hash2 := bh.GenHash()
	if hash != hash2 {
		fmt.Printf("bh GenHash twice failed, first=%v, second=%v.\n", GetHashPrefix(hash), GetHashPrefix(hash2))
		panic("bh GenHash twice failed, hash diff.")
	}
	return bh
}

//当前节点成为KING，出块
func (p Processer) castBlock(gid groupsig.ID, height uint64, qn int64) *core.BlockHeader {
	fmt.Printf("begin Processer::castBlock, height=%v, qn=%v...\n", height, qn)
	var bh *core.BlockHeader
	//var hash common.Hash
	//hash = bh.Hash //TO DO:替换成出块头的哈希
	//to do : change nonce
	nonce := time.Now().Unix()
	//调用鸠兹的铸块处理
	if !PROC_TEST_MODE {
		block := p.MainChain.CastingBlock(uint64(height), uint64(nonce), uint64(qn), p.GetMinerID().Serialize(), gid.Serialize())
		if block == nil {
			fmt.Printf("MainChain::CastingBlock failed, height=%v, qn=%v, gid=%v, mid=%v.\n", height, qn, GetIDPrefix(gid), GetIDPrefix(p.GetMinerID()))
			panic("MainChain::CastingBlock failed, jiuci return nil.\n")
		}
		bh = block.Header

		fmt.Printf("AAAAAA castBlock bh %v\n", bh)
		fmt.Printf("AAAAAA chain top bh %v\n", p.MainChain.QueryTopBlock())

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
	fmt.Printf("castor sign bh result = %v.\n", b)

	if bh.Height > 0 && si.DataSign.IsValid() && bh.Height == height {
		var tmp_id groupsig.ID
		if tmp_id.Deserialize(bh.Castor) != nil {
			panic("ID Deserialize failed.")
		}
		fmt.Printf("success cast block, height= %v, castor= %v.\n", bh.Height, GetIDPrefix(tmp_id))
		//发送该出块消息
		var ccm ConsensusCastMessage
		ccm.BH = *bh
		//ccm.GroupID = gid
		ccm.GenSign(SecKeyInfo{p.GetMinerID(), p.getSignKey(gid)})
		if !PROC_TEST_MODE {
			fmt.Printf("call network service SendCastVerify...\n")
			fmt.Printf("cast block info hash=%v, height=%v, prehash=%v, pretime=%v, castor=%v", GetHashPrefix(bh.Hash), bh.Height, GetHashPrefix(bh.PreHash), bh.PreTime, GetIDPrefix(p.GetMinerID()))
			SendCastVerify(&ccm)
		} else {
			fmt.Printf("call proc.OnMessageCast direct...\n")
			for _, proc := range p.GroupProcs {
				proc.OnMessageCast(ccm)
			}
		}
	} else {
		fmt.Printf("bh Error or sign Error, bh=%v, ds=%v, real height=%v.\n", height, GetSignPrefix(si.DataSign), bh.Height)
		//panic("bh Error or sign Error.")
		return nil
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
