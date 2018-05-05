package logical

import (
	"common"
	"consensus/groupsig"
	"fmt"
)

//新组的上链处理（全网节点/父亲组节点需要处理）
//组的索引ID为DUMMY ID。
//待共识的数据由链上获取(公信力)，不由消息获取。
//消息提供4样东西，成员ID，共识数据哈希，组公钥，组ID。
type NewGroupMemberData struct {
	h   common.Hash     //父亲组指定信息的哈希（不可改变）
	gid groupsig.ID     //组ID(非父亲组指定的DUMMY ID),而是跟组内成员的初始化共识结果有关
	gpk groupsig.Pubkey //组公钥
}

//组外矿工节点处理器
type NewGroupChained struct {
	sgi    StaticGroupInfo               //共识数据（基准）和组成员列表
	mems   map[string]NewGroupMemberData //接收到的组成员共识结果（成员ID->组ID和组公钥）
	status int                           //-1,组初始化失败（超时或无法达成共识，不可逆）；=0，组初始化中；=1，组初始化成功
	gpk    groupsig.Pubkey               //输出：生成的组公钥
}

//创建一个初始化中的组
func CreateInitingGroup(s StaticGroupInfo) NewGroupChained {
	var ngc NewGroupChained
	ngc.sgi = s
	ngc.mems = make(map[string]NewGroupMemberData, 0)
	return ngc
}

//生成一个静态组信息（用于加入到全局静态组）
func (ngc NewGroupChained) GetStaticGroupInfo() StaticGroupInfo {
	return ngc.sgi
}

func (ngc NewGroupChained) getSize() int {
	return len(ngc.mems)
}

//找出收到最多的相同值
func (ngc *NewGroupChained) Convergence() bool {
	fmt.Printf("begin Convergence, K=%v, mems=%v.\n", GetGroupK(), len(ngc.mems))
	/*
		if ngc.gpk.IsValid() {
			fmt.Printf("gpk already valid, =%v.\n", ngc.gpk.GetHexString())
			return true
		}
	*/
	type count_data struct {
		count int
		pk    groupsig.Pubkey
	}
	countMap := make(map[string]count_data, 0)
	//统计出现次数
	for _, v := range ngc.mems {
		if k, ok := countMap[v.gpk.GetHexString()]; ok {
			k.count = k.count + 1
			countMap[v.gpk.GetHexString()] = k
		} else {
			var item count_data
			item.pk = v.gpk
			item.count = 1
			countMap[v.gpk.GetHexString()] = item
		}
	}
	/*
		fmt.Printf("CountMap size=%v.\n", len(countMap))
		for k, v := range countMap {
			fmt.Printf("---countMap info : count=%v, gpk=%v.\n", v.count, k)
		}
	*/
	//查找最多的元素
	var gpk groupsig.Pubkey
	var count int
	for _, v := range countMap {
		if count == 0 || v.count > count {
			count = v.count
			gpk = v.pk
		}
	}
	if count >= GetGroupK() {
		fmt.Printf("found max count gpk=%v, count=%v.\n", gpk.GetHexString(), count)
		ngc.gpk = gpk
		return true
	}
	fmt.Printf("found max count gpk failed, max_gpk=%v, count=%v.\n", gpk.GetHexString(), count)
	return false
}

//检查和更新组初始化状态
//to do : 失败处理可以更精细化
func (ngc *NewGroupChained) UpdateStatus() int {
	fmt.Printf("begin UpdateStatus, cur_status=%v.\n", ngc.status)
	if ngc.status == -1 || ngc.status == 1 {
		return ngc.status
	}
	if len(ngc.mems) >= GetGroupK() { //收到超过阈值成员的数据
		if ngc.Convergence() { //相同性测试
			ngc.status = 1
			return ngc.status //有超过阈值的组成员生成的组公钥相同
		} else {
			if len(ngc.mems) == GROUP_MAX_MEMBERS { //收到了所有组员的结果，仍然失败
				ngc.status = -1
				return ngc.status
			}
		}
	}
	return ngc.status
}

//组生成器，父亲组节点或全网节点组外处理器（非组内初始化共识器）
type NewGroupGenerator struct {
	groups     map[string]*NewGroupChained //组ID（dummyID）->组创建共识
	globalInfo *GlobalGroups
}

func (ngg *NewGroupGenerator) Init(gg *GlobalGroups) {
	ngg.globalInfo = gg
	ngg.groups = make(map[string]*NewGroupChained, 1)
	//to do : 从主链加载待初始化的组信息
}

func (ngg *NewGroupGenerator) addInitingGroup(ngc NewGroupChained) {
	dummy_id := ngc.sgi.GIS.DummyID
	if _, ok := ngg.groups[dummy_id.GetHexString()]; !ok {
		fmt.Printf("add initing group %p ok, dummy_id=%v.\n", &ngc, GetIDPrefix(dummy_id))
		ngg.groups[dummy_id.GetHexString()] = &ngc
	} else {
		fmt.Printf("InitingGroup dummy_gid=%v already exist.\n", GetIDPrefix(dummy_id))
	}
	return
}

//创建新组数据接收处理
//gid：待初始化组的dummy id
//uid：组成员的公开id（和组无关）
//ngmd：组的初始化共识结果
//返回：-1异常；0正常；1正常，且该组已达到阈值验证条件，可上链。
func (ngg *NewGroupGenerator) ReceiveData(id GroupMinerID, ngmd NewGroupMemberData) int {
	fmt.Printf("ngg ReceiveData, dummy_gid=%v...\n", GetIDPrefix(id.gid))
	ngc, ge := ngg.groups[id.gid.GetHexString()]
	fmt.Printf("ReceiveData, ngg initing group count=%v.\n", len(ngg.groups))
	if !ge { //不存在该组
		sgi, err := ngg.globalInfo.GetGroupByDummyID(id.gid) //在全局组对象中查找
		if err != nil {
			fmt.Printf("ReceiveData failed, not found initing group.\n")
			return -1
		} else {
			fmt.Printf("found new init group %v in gg and add it to ngg.\n", GetIDPrefix(id.gid))
			ngg.addInitingGroup(CreateInitingGroup(sgi))
			ngc, ge = ngg.groups[id.gid.GetHexString()]
			if !ge {
				panic("addInitingGroup ERROR.")
			}
		}
	}
	fmt.Printf("already exist %v mem's data, status=%v.\n ", ngc.getSize(), ngc.status)
	if ngc.sgi.GIS.IsExpired() { //该组初始化共识已超时
		fmt.Printf("ReceiveData failed, group initing timeout.\n")
		return -1
	}
	if !ngc.sgi.MemExist(id.uid) { //消息发送方不属于待初始化的组
		fmt.Printf("ReceiveData failed, msg sender not in group.\n")
		return -1
	}
	_, ue := ngc.mems[id.uid.GetHexString()]
	if ue { //已收到过该用户的数据
		fmt.Printf("ReceiveData failed, receive same node data, ignore it, existed size=%v. mems=%p.\n", len(ngc.mems), &ngc.mems)
		for m, _ := range ngc.mems {
			fmt.Printf("---exist member %v.\n", m)
		}
		return 0
	}
	if ngmd.h != ngc.sgi.GIS.GenHash() { //共识数据异常
		fmt.Printf("ReceiveData failed, parent data hash diff.\n")
		return -1
	}
	ngc.mems[id.uid.GetHexString()] = ngmd //数据接收
	fmt.Printf("ReceiveData OK, sender size=%v, status=%v.\n", len(ngc.mems), ngc.status)
	if len(ngc.mems) >= GetGroupK() {
		check_result := ngc.UpdateStatus()
		fmt.Printf("Check gourp inited result=%v, status=%v.\n", check_result, ngc.status)
		if check_result == 1 {
			new_gpk := ngc.gpk
			fmt.Printf("SUCCESS ACCEPT A NEW GROUP!!! group pub key=%v.\n", new_gpk.GetHexString())
		}
		return 1
	} else {
		return 0
	}
	fmt.Printf("ReceiveData failed, because common error.\n")
	return -1
}

///////////////////////////////////////////////////////////////////////////////
//组初始化共识状态
type GROUP_INIT_STATUS int

const (
	GIS_RAW    GROUP_INIT_STATUS = iota //组处于原始状态（知道有哪些人是一组的，但是组公钥和组ID尚未生成）
	GIS_PIECE                           //没有收到父亲组的组初始化消息，而先收到了组成员发给我的秘密分享
	GIS_SHARED                          //当前节点已经生成秘密分享片段，并发送给组内成员
	GIS_INITED                          //组公钥和ID已生成，可以进行铸块
)

//组共识上下文
//判断一个消息是否合法，在外层验证
//判断一个消息是否来自组内，由GroupContext验证
type GroupContext struct {
	is   GROUP_INIT_STATUS         //组初始化状态
	node GroupNode                 //组节点信息（用于初始化生成组公钥和签名私钥）
	mems []PubKeyInfo              //组内成员ID列表（由父亲组指定）
	gis  ConsensusGroupInitSummary //组初始化信息（由父亲组指定）
}

func (gc *GroupContext) GetNode() *GroupNode {
	return &gc.node
}

func (gc GroupContext) GetGroupStatus() GROUP_INIT_STATUS {
	return gc.is
}

func (gc GroupContext) getMembers() []PubKeyInfo {
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
func (gc *GroupContext) UpdateMesageFromParent(grm ConsensusGroupRawMessage) {
	if gc.is == GIS_PIECE {
		gc.mems = make([]PubKeyInfo, len(grm.MEMS))
		copy(gc.mems[:], grm.MEMS[:])
		gc.gis = grm.GI
		gc.is = GIS_RAW
	} else {
		fmt.Printf("GroupContext::UpdateMesageFromParent failed, status=%v.\n", gc.is)
	}
	return
}

//从秘密分享消息创建GroupContext结构
func CreateGroupContextWithPieceMessage(spm ConsensusSharePieceMessage, mi MinerInfo) *GroupContext {
	gc := new(GroupContext)
	gc.is = GIS_PIECE
	gc.node.InitForMiner(mi.GetMinerID(), mi.SecretSeed)
	gc.node.InitForGroup(spm.GISHash)
	return gc
}

//从组初始化消息创建GroupContext结构
func CreateGroupContextWithRawMessage(grm ConsensusGroupRawMessage, mi MinerInfo) *GroupContext {
	if len(grm.MEMS) != GROUP_MAX_MEMBERS {
		fmt.Printf("group member size failed=%v.\n", len(grm.MEMS))
		return nil
	}
	for k, v := range grm.MEMS {
		if !v.GetID().IsValid() {
			fmt.Printf("i=%v, ID failed=%v.\n", k, v.GetID().GetHexString())
			return nil
		}
	}
	gc := new(GroupContext)
	gc.mems = make([]PubKeyInfo, len(grm.MEMS))
	copy(gc.mems[:], grm.MEMS[:])
	gc.gis = grm.GI
	gc.is = GIS_RAW
	gc.node.InitForMiner(mi.GetMinerID(), mi.SecretSeed)
	gc.node.InitForGroup(grm.GI.GenHash())
	return gc
}

//收到一片秘密分享消息
//返回-1为异常，返回0为正常接收，返回1为已收到所有组成员的签名私钥
func (gc *GroupContext) SignPKMessage(spkm ConsensusSignPubKeyMessage) int {
	result := gc.node.SetSignPKPiece(spkm.SI.SignMember, spkm.SignPK)
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
func (gc *GroupContext) PieceMessage(spm ConsensusSharePieceMessage) int {
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
func (gc *GroupContext) GenSharePieces() ShareMapID {
	shares := make(ShareMapID, 0)
	if len(gc.mems) == GROUP_MAX_MEMBERS && gc.is == GIS_RAW {
		secs := gc.node.GenSharePiece(gc.getIDs())
		var piece SharePiece
		piece.Pub = gc.node.GetSeedPubKey()
		for k, v := range secs {
			piece.Share = v
			shares[k] = piece
		}
		gc.is = GIS_SHARED
	} else {
		fmt.Printf("GenSharePieces failed, mems=%v, status=%v.\n", len(gc.mems), gc.is)
	}
	return shares
}

//（收到所有组内成员的秘密共享后）取得组信息
func (gc GroupContext) GetGroupInfo() JoinedGroup {
	return gc.node.GenInnerGroup()
}

//未初始化完成的加入组
type JoiningGroups struct {
	groups map[string]*GroupContext //group dummy id->GroupContext
}

func (jgs *JoiningGroups) Init() {
	jgs.groups = make(map[string]*GroupContext, 0)
}

func (jgs *JoiningGroups) ConfirmGroupFromRaw(grm ConsensusGroupRawMessage, mi MinerInfo) *GroupContext {
	if v, ok := jgs.groups[grm.GI.DummyID.GetHexString()]; ok {
		gs := v.GetGroupStatus()
		fmt.Printf("found initing group info BY RAW, status=%v...\n", gs)
		if gs == GIS_PIECE {
			v.UpdateMesageFromParent(grm)
			fmt.Printf("after UpdateParentMessage, status=%v.\n", v.GetGroupStatus())
		}
		return v
	} else {
		fmt.Printf("create new initing group info by RAW...\n")
		v = CreateGroupContextWithRawMessage(grm, mi)
		if v != nil {
			jgs.groups[grm.GI.DummyID.GetHexString()] = v
		}
		return v
	}
}

func (jgs *JoiningGroups) ConfirmGroupFromPiece(spm ConsensusSharePieceMessage, mi MinerInfo) *GroupContext {
	if v, ok := jgs.groups[spm.DummyID.GetHexString()]; ok {
		fmt.Printf("found initing group info by SP...\n")
		return v
	} else {
		fmt.Printf("create new initing group info by SP...\n")
		v = CreateGroupContextWithPieceMessage(spm, mi)
		if v != nil {
			jgs.groups[spm.DummyID.GetHexString()] = v
		}
		return v
	}
}

//gid : group dummy id
func (jgs *JoiningGroups) GetGroup(gid groupsig.ID) *GroupContext {
	if v, ok := jgs.groups[gid.GetHexString()]; ok {
		return v
	} else {
		return nil
	}
}
