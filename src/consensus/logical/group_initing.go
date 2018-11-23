package logical

import (
	"consensus/groupsig"
	"log"
	"sync"
	"sync/atomic"
	"consensus/model"
	"common"
)

const (
	INIT_NOTFOUND = -2
	INIT_FAIL = -1
	INITING = 0
	INIT_SUCCESS = 1	//初始化成功, 组公钥生成
)

//组外矿工节点处理器
type InitingGroup struct {
	//sgi    StaticGroupInfo               //共识数据（基准）和组成员列表
	gInfo	*model.ConsensusGroupInitInfo
	//mems   map[string]NewGroupMemberData //接收到的组成员共识结果（成员ID->组ID和组公钥）
	receivedGPKs map[string]groupsig.Pubkey
	lock 	sync.RWMutex

	status int32                           //-1,组初始化失败（超时或无法达成共识，不可逆）；=0，组初始化中；=1，组初始化成功
	gpk    groupsig.Pubkey               //输出：生成的组公钥
}

//创建一个初始化中的组
func CreateInitingGroup(raw *model.ConsensusGroupRawMessage) *InitingGroup {
	return &InitingGroup{
		gInfo: &raw.GInfo,
		receivedGPKs: make(map[string]groupsig.Pubkey),
		status: INITING,
	}
}

func (ig *InitingGroup) MemberExist(id groupsig.ID) bool {
	return ig.gInfo.MemberExists(id)
}

func (ig *InitingGroup) receive(id groupsig.ID, pk groupsig.Pubkey) int32 {
	status := atomic.LoadInt32(&ig.status)
	if status != INITING {
		return status
	}

	ig.lock.Lock()
	defer ig.lock.Unlock()

	ig.receivedGPKs[id.GetHexString()] = pk
	ig.convergence()
	return ig.status
}

func (ig *InitingGroup) receiveSize() int {
	ig.lock.RLock()
	defer ig.lock.RUnlock()

	return len(ig.receivedGPKs)
}

//找出收到最多的相同值
func (ig *InitingGroup) convergence() bool {
	threshold := model.Param.GetGroupK(ig.gInfo.MemberSize())
	log.Printf("begin Convergence, K=%v, mems=%v.\n", threshold, ig.gInfo.MemberSize())

	type countData struct {
		count int
		pk    groupsig.Pubkey
	}
	countMap := make(map[string]*countData, 0)

	//统计出现次数
	for _, v := range ig.receivedGPKs {
		ps := v.GetHexString()
		if k, ok := countMap[ps]; ok {
			k.count++
			countMap[ps] = k
		} else {
			item := &countData{
				count: 1,
				pk: v,
			}
			countMap[ps] = item
		}
	}

	//查找最多的元素
	var gpk groupsig.Pubkey
	var maxCnt = common.MinInt64
	for _, v := range countMap {
		if v.count > maxCnt {
			maxCnt = v.count
			gpk = v.pk
		}
	}

	if maxCnt >= threshold && atomic.CompareAndSwapInt32(&ig.status, INITING, INIT_SUCCESS){
		log.Printf("found max maxCnt gpk=%v, maxCnt=%v.\n", gpk.ShortS(), maxCnt)
		ig.gpk = gpk
		return true
	}
	return false
}

func (ig *InitingGroup) ReadyTimeout(h uint64) bool {
    return ig.gInfo.GI.ReadyTimeout(h)
}

//组生成器，父亲组节点或全网节点组外处理器（非组内初始化共识器）
type NewGroupGenerator struct {
	groups     sync.Map //组ID（dummyID）->组创建共识 string -> *InitingGroup
	//groups     map[string]*InitingGroup //组ID（dummyID）->组创建共识
}

func CreateNewGroupGenerator() *NewGroupGenerator {
    return &NewGroupGenerator{
    	groups: sync.Map{},
	}
}

func (ngg *NewGroupGenerator) addInitingGroup(initingGroup *InitingGroup) bool {
	gHash := initingGroup.gInfo.GroupHash()

	//log.Println("------dummyId:", dummyId.GetHexString())
	_, load := ngg.groups.LoadOrStore(gHash.Hex(), initingGroup)
	if load {
		log.Printf("InitingGroup gHash=%v already exist.\n", gHash.ShortS())
	} else {
		log.Printf("add initing group %p ok, gHash=%v.\n", initingGroup, gHash.ShortS())
	}
	return !load
}

func (ngg *NewGroupGenerator) getInitingGroup(gHash common.Hash) *InitingGroup {
    if v, ok := ngg.groups.Load(gHash.Hex()); ok {
    	return v.(*InitingGroup)
	}
	return nil
}

func (ngg *NewGroupGenerator) removeInitingGroup(gHash common.Hash)  {
    ngg.groups.Delete(gHash.Hex())
}

//创建新组数据接收处理
//gid：待初始化组的dummy id
//uid：组成员的公开id（和组无关）
//ngmd：组的初始化共识结果
//返回：-1异常；0正常；1正常，且该组已达到阈值验证条件，可上链。
func (ngg *NewGroupGenerator) ReceiveData(msg *model.ConsensusGroupInitedMessage, height uint64) int32 {
	gHash := msg.GHash
	log.Printf("generator ReceiveData, gHash=%v...\n", gHash.ShortS())
	initingGroup := ngg.getInitingGroup(gHash)

	if initingGroup == nil { //不存在该组
		return INIT_NOTFOUND
	}
	if initingGroup.ReadyTimeout(height) { //该组初始化共识已超时
		log.Printf("ReceiveData failed, group initing timeout.\n")
		atomic.CompareAndSwapInt32(&initingGroup.status, INITING, INIT_FAIL)
		return INIT_FAIL
	}

	return initingGroup.receive(msg.SI.GetID(), msg.GroupPK) //数据接收
	//
}

///////////////////////////////////////////////////////////////////////////////

const (
	GIS_RAW    int32 = iota //组处于原始状态（知道有哪些人是一组的，但是组公钥和组ID尚未生成）
	GIS_PIECE                           //没有收到父亲组的组初始化消息，而先收到了组成员发给我的秘密分享
	GIS_SHARED                          //当前节点已经生成秘密分享片段，并发送给组内成员
	GIS_INITED                          //组公钥和ID已生成，可以进行铸块
)

//组共识上下文
//判断一个消息是否合法，在外层验证
//判断一个消息是否来自组内，由GroupContext验证
type GroupContext struct {
	is   int32         //组初始化状态
	node GroupNode                 //组节点信息（用于初始化生成组公钥和签名私钥）
	gInfo  *model.ConsensusGroupInitInfo //组初始化信息（由父亲组指定）
}

func (gc *GroupContext) GetNode() *GroupNode {
	return &gc.node
}

func (gc GroupContext) GetGroupStatus() int32 {
	return gc.is
}

func (gc GroupContext) getMembers() []groupsig.ID {
	return gc.gInfo.Mems
}

func (gc GroupContext) MemExist(id groupsig.ID) bool {
	return gc.gInfo.MemberExists(id)
}


//从组初始化消息创建GroupContext结构
func CreateGroupContextWithRawMessage(grm *model.ConsensusGroupRawMessage, mi *model.SelfMinerDO) *GroupContext {
	if len(grm.GInfo.Mems) != model.Param.GetGroupMemberNum() {
		log.Printf("group member size failed=%v.\n", len(grm.GInfo.Mems))
		return nil
	}
	for k, v := range grm.GInfo.Mems {
		if !v.IsValid() {
			log.Printf("i=%v, ID failed=%v.\n", k, v.GetHexString())
			return nil
		}
	}
	gc := new(GroupContext)
	gc.is = GIS_RAW
	gc.gInfo = &grm.GInfo
	gc.node.memberNum = grm.GInfo.MemberSize()
	gc.node.InitForMiner(mi)
	gc.node.InitForGroup(grm.GInfo.GroupHash())
	return gc
}

//收到一片秘密分享消息
//返回-1为异常，返回0为正常接收，返回1为已收到所有组成员的签名私钥
func (gc *GroupContext) SignPKMessage(spkm *model.ConsensusSignPubKeyMessage) int {
	result := gc.node.SetSignPKPiece(spkm)
	switch result {
	case 1:
	case 0:
	case -1:
		panic("GroupContext::SignPKMessage failed, SetSignPKPiece result -1.")
	}
	return result
}

//收到一片秘密分享消息
//返回-1为异常，返回0为正常接收，返回1为已聚合出组成员私钥（用于签名）
func (gc *GroupContext) PieceMessage(spm *model.ConsensusSharePieceMessage) int {
	/*可能父亲组消息还没到，先收到组成员的piece消息
	if !gc.MemExist(spm.si.SignMember) { //非组内成员
		return -1
	}
	*/
	result := gc.node.SetInitPiece(spm.SI.SignMember, spm.Share)
	switch result {
	case 1: //完成聚合（已生成组公钥和组成员签名私钥）
		//由外层启动组外广播（to do : 升级到通知父亲组节点）
	case 0: //正常接收
	case -1:
		panic("GroupContext::PieceMessage failed, SetInitPiece result -1.")
	}
	return result
}

//生成发送给组内成员的秘密分享: si = F(IDi)
func (gc *GroupContext) GenSharePieces() model.SharePieceMap {
	shares := make(model.SharePieceMap, 0)
	if atomic.CompareAndSwapInt32(&gc.is, GIS_RAW, GIS_SHARED) {
		secs := gc.node.GenSharePiece(gc.getMembers())
		var piece model.SharePiece
		piece.Pub = gc.node.GetSeedPubKey()
		for k, v := range secs {
			piece.Share = v
			shares[k] = piece
		}
	}
	return shares
}

//（收到所有组内成员的秘密共享后）取得组信息
func (gc *GroupContext) GetGroupInfo() *JoinedGroup {
	return gc.node.GenInnerGroup()
}

//未初始化完成的加入组
type JoiningGroups struct {
	groups sync.Map
	//groups map[string]*GroupContext //group dummy id->*GroupContext
}

func NewJoiningGroups() *JoiningGroups {
	return &JoiningGroups{
		groups: sync.Map{},
	}
}

func (jgs *JoiningGroups) ConfirmGroupFromRaw(grm *model.ConsensusGroupRawMessage, mi *model.SelfMinerDO) *GroupContext {
	gHash := grm.GInfo.GroupHash()
	if v := jgs.GetGroup(gHash); v != nil {
		gs := v.GetGroupStatus()
		log.Printf("found initing group info BY RAW, status=%v...\n", gs)
		return v
	} else {
		log.Printf("create new initing group info by RAW...\n")
		v = CreateGroupContextWithRawMessage(grm, mi)
		if v != nil {
			jgs.groups.Store(gHash.Hex(), v)
		}
		return v
	}
}

//gid : group dummy id
func (jgs *JoiningGroups) GetGroup(gHash common.Hash) *GroupContext {
	if v, ok := jgs.groups.Load(gHash.Hex()); ok {
		return v.(*GroupContext)
	}

	//fmt.Println("gc is NULL, gid:", gid.GetHexString())

	return nil
}

func (jgs *JoiningGroups) RemoveGroup(gHash common.Hash)  {
    jgs.groups.Delete(gHash.Hex())
}
func (jgs *JoiningGroups) forEach(f func(gc *GroupContext) bool) {
    jgs.groups.Range(func(key, value interface{}) bool {
		return f(value.(*GroupContext))
	})
}