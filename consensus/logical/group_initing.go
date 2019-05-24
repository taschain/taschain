package logical

import (
	"github.com/hashicorp/golang-lru"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/model"
	"sync"
	"sync/atomic"
	"time"
)

const (
	INIT_NOTFOUND = -2
	INIT_FAIL     = -1
	INITING       = 0
	INIT_SUCCESS  = 1 //初始化成功, 组公钥生成
)

//
// 矿工节点处理器
type InitedGroup struct {
	//sgi    StaticGroupInfo               //共识数据（基准）和组成员列表
	gInfo *model.ConsensusGroupInitInfo
	//mems   map[string]NewGroupMemberData //接收到的组成员共识结果（成员ID->组ID和组公钥）
	receivedGPKs map[string]groupsig.Pubkey
	lock         sync.RWMutex

	threshold int
	status    int32           //-1,组初始化失败（超时或无法达成共识，不可逆）；=0，组初始化中；=1，组初始化成功
	gpk       groupsig.Pubkey //输出：生成的组公钥
}

//创建一个初始化中的组
func createInitedGroup(gInfo *model.ConsensusGroupInitInfo) *InitedGroup {
	threshold := model.Param.GetGroupK(len(gInfo.Mems))
	return &InitedGroup{
		receivedGPKs: make(map[string]groupsig.Pubkey),
		status:       INITING,
		threshold:    threshold,
		gInfo:        gInfo,
	}
}

func (ig *InitedGroup) receive(id groupsig.ID, pk groupsig.Pubkey) int32 {
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

func (ig *InitedGroup) receiveSize() int {
	ig.lock.RLock()
	defer ig.lock.RUnlock()

	return len(ig.receivedGPKs)
}

func (ig *InitedGroup) hasRecived(id groupsig.ID) bool {
	ig.lock.RLock()
	defer ig.lock.RUnlock()

	_, ok := ig.receivedGPKs[id.GetHexString()]
	return ok
}

//找出收到最多的相同值
func (ig *InitedGroup) convergence() bool {
	stdLogger.Debug("begin Convergence, K=%v\n", ig.threshold)

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
				pk:    v,
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

	if maxCnt >= ig.threshold && atomic.CompareAndSwapInt32(&ig.status, INITING, INIT_SUCCESS) {
		stdLogger.Debug("found max maxCnt gpk=%v, maxCnt=%v.\n", gpk.ShortS(), maxCnt)
		ig.gpk = gpk
		return true
	}
	return false
}

//组生成器，父亲组节点或全网节点组外处理器（非组内初始化共识器）
type NewGroupGenerator struct {
	groups sync.Map //组ID（dummyID）->组创建共识 string -> *InitedGroup
	//groups     map[string]*InitedGroup //组ID（dummyID）->组创建共识
}

func CreateNewGroupGenerator() *NewGroupGenerator {
	return &NewGroupGenerator{
		groups: sync.Map{},
	}
}

func (ngg *NewGroupGenerator) getInitedGroup(gHash common.Hash) *InitedGroup {
	if v, ok := ngg.groups.Load(gHash.Hex()); ok {
		return v.(*InitedGroup)
	}
	return nil
}

func (ngg *NewGroupGenerator) addInitedGroup(g *InitedGroup) *InitedGroup {
	v, _ := ngg.groups.LoadOrStore(g.gInfo.GroupHash().Hex(), g)
	return v.(*InitedGroup)
}

func (ngg *NewGroupGenerator) removeInitedGroup(gHash common.Hash) {
	ngg.groups.Delete(gHash.Hex())
}

func (ngg *NewGroupGenerator) forEach(f func(ig *InitedGroup) bool) {
	ngg.groups.Range(func(key, value interface{}) bool {
		g := value.(*InitedGroup)
		return f(g)
	})
}

//创建新组数据接收处理
//gid：待初始化组的dummy id
//uid：组成员的公开id（和组无关）
//ngmd：组的初始化共识结果
//返回：-1异常；0正常；1正常，且该组已达到阈值验证条件，可上链。
//func (ngg *NewGroupGenerator) ReceiveData(msg *model.ConsensusGroupInitedMessage, threshold int) int32 {
//	gHash := msg.GHash
//	stdLogger.Debug("generator ReceiveData, gHash=%v...\n", gHash.ShortS())
//	initedGroup := ngg.getInitedGroup(gHash)
//
//	if initedGroup == nil { //不存在该组
//		g := createInitedGroup(threshold)
//		initedGroup = ngg.addInitedGroup(g)
//	}
//
//	return initedGroup.receive(msg.SI.GetID(), msg.GroupPK) //数据接收
//	//
//}

///////////////////////////////////////////////////////////////////////////////

const (
	GisInit           int32 = iota //组处于原始状态（知道有哪些人是一组的，但是组公钥和组ID尚未生成）
	GisSendSharePiece              //已发送sharepiece
	GisSendSignPk                  //已发送自己的签名公钥
	GisSendInited                  //组公钥和ID已生成，可以进行铸块
	GisGroupInitDone               //组已初始化完成已上链
)

//组共识上下文
//判断一个消息是否合法，在外层验证
//判断一个消息是否来自组内，由GroupContext验证
type GroupContext struct {
	createTime    time.Time
	is            int32                         //组初始化状态
	node          *GroupNode                    //组节点信息（用于初始化生成组公钥和签名私钥）
	gInfo         *model.ConsensusGroupInitInfo //组初始化信息（由父亲组指定）
	candidates    []groupsig.ID
	sharePieceMap model.SharePieceMap
	sendLog       bool
}

func (gc *GroupContext) GetNode() *GroupNode {
	return gc.node
}

func (gc *GroupContext) GetGroupStatus() int32 {
	return atomic.LoadInt32(&gc.is)
}

func (gc GroupContext) getMembers() []groupsig.ID {
	return gc.gInfo.Mems
}

func (gc *GroupContext) MemExist(id groupsig.ID) bool {
	return gc.gInfo.MemberExists(id)
}

func (gc *GroupContext) StatusTransfrom(from, to int32) bool {
	return atomic.CompareAndSwapInt32(&gc.is, from, to)
}

func (gc *GroupContext) generateMemberMask() (mask []byte) {
	mask = make([]byte, (len(gc.candidates)+7)/8)

	for i, id := range gc.candidates {
		b := mask[i/8]
		if gc.MemExist(id) {
			b |= 1 << byte(i%8)
			mask[i/8] = b
		}
	}
	return
}

//从组初始化消息创建GroupContext结构
func CreateGroupContextWithRawMessage(grm *model.ConsensusGroupRawMessage, candidates []groupsig.ID, mi *model.SelfMinerDO) *GroupContext {
	for k, v := range grm.GInfo.Mems {
		if !v.IsValid() {
			stdLogger.Debug("i=%v, ID failed=%v.\n", k, v.GetHexString())
			return nil
		}
	}
	gc := new(GroupContext)
	gc.createTime = time.Now()
	gc.is = GisInit
	gc.candidates = candidates
	gc.gInfo = &grm.GInfo
	gc.node = &GroupNode{}
	gc.node.memberNum = grm.GInfo.MemberSize()
	gc.node.InitForMiner(mi)
	gc.node.InitForGroup(grm.GInfo.GroupHash())
	return gc
}

//收到一片秘密分享消息
//返回-1为异常，返回0为正常接收，返回1为已收到所有组成员的签名私钥
//func (gc *GroupContext) SignPKMessage(spkm *model.ConsensusSignPubKeyMessage) int {
//	result := gc.node.SetSignPKPiece(spkm)
//	switch result {
//	case 1:
//	case 0:
//	case -1:
//		panic("GroupContext::SignPKMessage failed, SetSignPKPiece result -1.")
//	}
//	return result
//}

//收到一片秘密分享消息
//返回-1为异常，返回0为正常接收，返回1为已聚合出组成员私钥（用于签名）
func (gc *GroupContext) PieceMessage(id groupsig.ID, share *model.SharePiece) int {
	/*可能父亲组消息还没到，先收到组成员的piece消息
	if !gc.MemExist(spm.si.SignMember) { //非组内成员
		return -1
	}
	*/
	result := gc.node.SetInitPiece(id, share)
	switch result {
	case 1: //完成聚合（已生成组公钥和组成员签名私钥）
		//由外层启动组外广播（to do : 升级到通知父亲组节点）
	case 0: //正常接收
	case -1:
		//panic("GroupContext::PieceMessage failed, SetInitPiece result -1.")
	}
	return result
}

//生成发送给组内成员的秘密分享: si = F(IDi)
func (gc *GroupContext) GenSharePieces() model.SharePieceMap {
	shares := make(model.SharePieceMap, 0)
	secs := gc.node.GenSharePiece(gc.getMembers())
	var piece model.SharePiece
	piece.Pub = gc.node.GetSeedPubKey()
	for k, v := range secs {
		piece.Share = v
		shares[k] = piece
	}
	gc.sharePieceMap = shares
	return shares
}

//（收到所有组内成员的秘密共享后）取得组信息
func (gc *GroupContext) GetGroupInfo() *JoinedGroup {
	return gc.node.GenInnerGroup(gc.gInfo.GroupHash())
}

//未初始化完成的加入组
type JoiningGroups struct {
	//groups sync.Map
	groups *lru.Cache
	//groups map[string]*GroupContext //group dummy id->*GroupContext
}

func NewJoiningGroups() *JoiningGroups {
	return &JoiningGroups{
		groups: common.MustNewLRUCache(50),
	}
}

func (jgs *JoiningGroups) ConfirmGroupFromRaw(grm *model.ConsensusGroupRawMessage, candidates []groupsig.ID, mi *model.SelfMinerDO) *GroupContext {
	gHash := grm.GInfo.GroupHash()
	if v := jgs.GetGroup(gHash); v != nil {
		gs := v.GetGroupStatus()
		stdLogger.Debug("found initing group info BY RAW, status=%v...\n", gs)
		return v
	} else {
		stdLogger.Debug("create new initing group info by RAW...\n")
		v = CreateGroupContextWithRawMessage(grm, candidates, mi)
		if v != nil {
			jgs.groups.Add(gHash.Hex(), v)
		}
		return v
	}
}

//gid : group dummy id
func (jgs *JoiningGroups) GetGroup(gHash common.Hash) *GroupContext {
	if v, ok := jgs.groups.Get(gHash.Hex()); ok {
		return v.(*GroupContext)
	}

	//fmt.Println("gc is NULL, gid:", gid.GetHexString())

	return nil
}

func (jgs *JoiningGroups) Clean(gHash common.Hash) {
	gc := jgs.GetGroup(gHash)
	if gc != nil && gc.StatusTransfrom(GisSendInited, GisGroupInitDone) {
		//gc.gInfo = nil
		//gc.node = nil
	}
}

func (jgs *JoiningGroups) RemoveGroup(gHash common.Hash) {
	jgs.groups.Remove(gHash.Hex())
}

func (jgs *JoiningGroups) forEach(f func(gc *GroupContext) bool) {
	for _, key := range jgs.groups.Keys() {
		v, ok := jgs.groups.Get(key)
		if !ok {
			continue
		}
		gc := v.(*GroupContext)
		if !f(gc) {
			break
		}
	}
}
