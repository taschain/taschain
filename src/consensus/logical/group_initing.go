package logical

import (
	"consensus/groupsig"
	"log"
	"sync"
	"sync/atomic"
	"vm/common/math"
	"consensus/model"
)

//新组的上链处理（全网节点/父亲组节点需要处理）
//组的索引ID为DUMMY ID。
//待共识的数据由链上获取(公信力)，不由消息获取。
//消息提供4样东西，成员ID，共识数据哈希，组公钥，组ID。
//type NewGroupMemberData struct {
//	h   common.Hash     //父亲组指定信息的哈希（不可改变）
//	gid groupsig.ID     //组ID(非父亲组指定的DUMMY ID),而是跟组内成员的初始化共识结果有关
//	gpk groupsig.Pubkey //组公钥
//}

const (
	INIT_NOTFOUND = -2
	INIT_FAIL = -1
	INITING = 0
	INIT_SUCCESS = 1	//初始化成功, 组公钥生成
)

//组外矿工节点处理器
type InitingGroup struct {
	//sgi    StaticGroupInfo               //共识数据（基准）和组成员列表
	gis		model.ConsensusGroupInitSummary
	//mems   map[string]NewGroupMemberData //接收到的组成员共识结果（成员ID->组ID和组公钥）
	receivedGPKs map[string]groupsig.Pubkey
	lock 	sync.RWMutex

	mems   []model.PubKeyInfo //接收到的组成员共识结果（成员ID->组ID和组公钥）
	status int32                           //-1,组初始化失败（超时或无法达成共识，不可逆）；=0，组初始化中；=1，组初始化成功
	gpk    groupsig.Pubkey               //输出：生成的组公钥
}

//创建一个初始化中的组
func CreateInitingGroup(raw *model.ConsensusGroupRawMessage) *InitingGroup {
	mems := make([]model.PubKeyInfo, len(raw.MEMS))
	copy(mems, raw.MEMS)
	return &InitingGroup{
		gis: raw.GI,
		mems: mems,
		receivedGPKs: make(map[string]groupsig.Pubkey),
		status: INITING,
	}
}

func (ig *InitingGroup) MemberExist(id groupsig.ID) bool {
	for _, mem := range ig.mems {
		if mem.ID.IsEqual(id) {
			return true
		}
	}
	return false
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
	threshold := model.Param.GetGroupK(int(ig.gis.Members))
	log.Printf("begin Convergence, K=%v, mems=%v.\n", threshold, len(ig.mems))

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
	var maxCnt = math.MinInt64
	for _, v := range countMap {
		if v.count > maxCnt {
			maxCnt = v.count
			gpk = v.pk
		}
	}

	if maxCnt >= threshold && atomic.CompareAndSwapInt32(&ig.status, INITING, INIT_SUCCESS){
		log.Printf("found max maxCnt gpk=%v, maxCnt=%v.\n", GetPubKeyPrefix(gpk), maxCnt)
		ig.gpk = gpk
		return true
	}
	return false
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
	dummyId := initingGroup.gis.DummyID
	_, load := ngg.groups.LoadOrStore(dummyId.GetHexString(), initingGroup)
	if load {
		log.Printf("InitingGroup dummy_gid=%v already exist.\n", GetIDPrefix(dummyId))
	} else {
		log.Printf("add initing group %p ok, dummyId=%v.\n", initingGroup, GetIDPrefix(dummyId))
	}
	return !load
}

func (ngg *NewGroupGenerator) getInitingGroup(dummyId groupsig.ID) *InitingGroup {
    if v, ok := ngg.groups.Load(dummyId.GetHexString()); ok {
    	return v.(*InitingGroup)
	}
	return nil
}

func (ngg *NewGroupGenerator) removeInitingGroup(dummyId groupsig.ID)  {
    ngg.groups.Delete(dummyId.GetHexString())
}

//创建新组数据接收处理
//gid：待初始化组的dummy id
//uid：组成员的公开id（和组无关）
//ngmd：组的初始化共识结果
//返回：-1异常；0正常；1正常，且该组已达到阈值验证条件，可上链。
func (ngg *NewGroupGenerator) ReceiveData(sgs *model.StaticGroupSummary, sender groupsig.ID, height uint64) int32 {
	id := sgs.GIS.DummyID
	log.Printf("generator ReceiveData, dummy_gid=%v...\n", GetIDPrefix(id))
	initingGroup := ngg.getInitingGroup(id)

	if initingGroup == nil { //不存在该组
		return INIT_NOTFOUND
	}
	if initingGroup.gis.ReadyTimeout(height) { //该组初始化共识已超时
		log.Printf("ReceiveData failed, group initing timeout.\n")
		atomic.CompareAndSwapInt32(&initingGroup.status, INITING, INIT_FAIL)
		return INIT_FAIL
	}

	return initingGroup.receive(sender, sgs.GroupPK) //数据接收
	//
	//
	//size := initingGroup.receiveSize()
	//log.Printf("ReceiveData OK, sender size=%v, status=%v.\n", size, initingGroup.status)
	//if size >= GetGroupK() {
	//	checkResult := initingGroup.UpdateStatus()
	//	log.Printf("Check gourp inited result=%v, status=%v.\n", checkResult, initingGroup.status)
	//	if checkResult == 1 {
	//		newGpk := initingGroup.gpk
	//		log.Printf("SUCCESS ACCEPT A NEW GROUP!!! group pub key=%v.\n", GetPubKeyPrefix(newGpk))
	//	}
	//	return 1
	//} else {
	//	return 0
	//}
	//log.Printf("ReceiveData failed, because common error.\n")
	//return -1
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
	mems []model.PubKeyInfo              //组内成员ID列表（由父亲组指定）
	gis  model.ConsensusGroupInitSummary //组初始化信息（由父亲组指定）
}

func (gc *GroupContext) GetNode() *GroupNode {
	return &gc.node
}

func (gc GroupContext) GetGroupStatus() int32 {
	return gc.is
}

func (gc GroupContext) getMembers() []model.PubKeyInfo {
	return gc.mems
}

func (gc GroupContext) getIDs() []groupsig.ID {
	var ids []groupsig.ID
	for _, v := range gc.mems {
		ids = append(ids, v.GetID())
	}
	return ids
}

func (gc GroupContext) MemExist(id groupsig.ID) bool {
	for _, v := range gc.mems {
		if v.GetID().GetHexString() == id.GetHexString() {
			return true
		}
	}
	return false
}

//更新组信息（先收到piece消息再收到raw消息的处理）
//func (gc *GroupContext) UpdateMesageFromParent(grm ConsensusGroupRawMessage) {
//	if gc.is == GIS_PIECE {
//		gc.mems = make([]PubKeyInfo, len(grm.MEMS))
//		copy(gc.mems[:], grm.MEMS[:])
//		gc.gis = grm.GI
//		gc.is = GIS_RAW
//	} else {
//		log.Printf("GroupContext::UpdateMesageFromParent failed, status=%v.\n", gc.is)
//	}
//	return
//}

//从秘密分享消息创建GroupContext结构
func CreateGroupContextWithPieceMessage(spm model.ConsensusSharePieceMessage, mi model.MinerInfo) *GroupContext {
	gc := new(GroupContext)
	gc.is = GIS_PIECE
	gc.node.InitForMiner(mi.GetMinerID(), mi.SecretSeed)
	gc.node.InitForGroup(spm.GISHash)
	return gc
}

//从组初始化消息创建GroupContext结构
func CreateGroupContextWithRawMessage(grm *model.ConsensusGroupRawMessage, mi *model.MinerInfo) *GroupContext {
	if len(grm.MEMS) != model.Param.GetGroupMemberNum() || len(grm.MEMS) != int(grm.GI.Members) {
		log.Printf("group member size failed=%v.\n", len(grm.MEMS))
		return nil
	}
	for k, v := range grm.MEMS {
		if !v.GetID().IsValid() {
			log.Printf("i=%v, ID failed=%v.\n", k, v.GetID().GetHexString())
			return nil
		}
	}
	gc := new(GroupContext)
	gc.mems = make([]model.PubKeyInfo, len(grm.MEMS))
	copy(gc.mems[:], grm.MEMS[:])
	gc.gis = grm.GI
	gc.is = GIS_RAW
	gc.node.memberNum = len(gc.mems)
	gc.node.InitForMiner(mi.GetMinerID(), mi.SecretSeed)
	gc.node.InitForGroup(grm.GI.GenHash())
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

//生成发送给组内成员的秘密分享
func (gc *GroupContext) GenSharePieces() model.ShareMapID {
	shares := make(model.ShareMapID, 0)
	if len(gc.mems) == int(gc.gis.Members) && atomic.CompareAndSwapInt32(&gc.is, GIS_RAW, GIS_SHARED) {
		secs := gc.node.GenSharePiece(gc.getIDs())
		var piece model.SharePiece
		piece.Pub = gc.node.GetSeedPubKey()
		for k, v := range secs {
			piece.Share = v
			shares[k] = piece
		}
	} else {
		log.Printf("GenSharePieces failed, mems=%v, status=%v.\n", len(gc.mems), gc.is)
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

func (jgs *JoiningGroups) ConfirmGroupFromRaw(grm *model.ConsensusGroupRawMessage, mi *model.MinerInfo) *GroupContext {
	if v := jgs.GetGroup(grm.GI.DummyID); v != nil {
		gs := v.GetGroupStatus()
		log.Printf("found initing group info BY RAW, status=%v...\n", gs)
		return v
	} else {
		log.Printf("create new initing group info by RAW...\n")
		v = CreateGroupContextWithRawMessage(grm, mi)
		if v != nil {
			jgs.groups.Store(grm.GI.DummyID.GetHexString(), v)
		}
		return v
	}
}

//gid : group dummy id
func (jgs *JoiningGroups) GetGroup(gid groupsig.ID) *GroupContext {
	if v, ok := jgs.groups.Load(gid.GetHexString()); ok {
		return v.(*GroupContext)
	}
	return nil
}

func (jgs *JoiningGroups) RemoveGroup(gid groupsig.ID)  {
    jgs.groups.Delete(gid.GetHexString())
}
func (jgs *JoiningGroups) forEach(f func(gc *GroupContext) bool) {
    jgs.groups.Range(func(key, value interface{}) bool {
		return f(value.(*GroupContext))
	})
}