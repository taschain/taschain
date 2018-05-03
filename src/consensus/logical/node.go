package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/rand"
	"fmt"
)

//数据接收池
type GroupInitPool struct {
	_pool ShareMapID
}

func (gmd *GroupInitPool) init() {
	gmd._pool = make(ShareMapID, 0)
}

//接收数据
func (gmd *GroupInitPool) ReceiveData(id groupsig.ID, piece SharePiece) int {
	fmt.Printf("GroupInitPool::ReceiveData, sender=%v, share=%v, pub=%v...\n", GetIDPrefix(id), GetSecKeyPrefix(piece.Share), GetPubKeyPrefix(piece.Pub))
	if _, ok := gmd._pool[id.GetHexString()]; !ok {
		gmd._pool[id.GetHexString()] = piece //没有收到过该成员消息
		return 0
	} else { //收到过
		if !gmd._pool[id.GetHexString()].IsEqual(piece) { //两次数据不一致
			fmt.Printf("GroupInitPool::ReceiveData failed, data diff.\n")
			return -1
		}
	}
	return 0
}

func (gmd *GroupInitPool) GetSize() int {
	return len(gmd._pool)
}

//生成组成员签名公钥列表（用于铸块相关消息的验签）
func (gmd GroupInitPool) GenMemberPubKeys() groupsig.PubkeyMapID {
	pubs := make(groupsig.PubkeyMapID, 0)
	for k, v := range gmd._pool {
		pubs[k] = v.Pub
	}
	return pubs
}

//生成矿工签名私钥
func (gmd GroupInitPool) GenMinerSignSecKey() *groupsig.Seckey {
	if len(gmd._pool) != GROUP_MAX_MEMBERS {
		return nil
	}
	shares := make([]groupsig.Seckey, 0)
	for _, v := range gmd._pool {
		shares = append(shares, v.Share)
	}
	sk := groupsig.AggregateSeckeys(shares)
	return sk
}

//生成组公钥
func (gmd GroupInitPool) GenGroupPubKey() *groupsig.Pubkey {
	if len(gmd._pool) != GROUP_MAX_MEMBERS {
		return nil
	}
	pubs := make([]groupsig.Pubkey, 0)
	for _, v := range gmd._pool {
		pubs = append(pubs, v.Pub)
	}
	gpk := groupsig.AggregatePubkeys(pubs)
	return gpk
}

//组相关的秘密
type MinerGroupSecret struct {
	secretSeed rand.Rand //某个矿工针对某个组的私密种子（矿工个人私密种子固定和组信息固定的情况下，该值固定）
}

func NewMinerGroupSecret(secret rand.Rand) MinerGroupSecret {
	var mgs MinerGroupSecret
	mgs.secretSeed = secret
	return mgs
}

//生成针对某个组的私密私钥
func (mgs MinerGroupSecret) GenSecKey() groupsig.Seckey {
	return *groupsig.NewSeckeyFromRand(mgs.secretSeed.Deri(0))
}

//生成针对某个组的私密私钥列表
//n : 门限数
func (mgs MinerGroupSecret) GenSecKeyList(n int) []groupsig.Seckey {
	secs := make([]groupsig.Seckey, n)
	for i := 0; i < n; i++ {
		secs[i] = *groupsig.NewSeckeyFromRand(mgs.secretSeed.Deri(i))
	}
	return secs
}

//组节点（一个矿工加入多个组，则有多个组节点）
type GroupNode struct {
	//用户属性（本质上可以跨多个GroupNode共享）
	u_sk      common.PrivateKey //用户私钥（非组签名私钥）
	u_address common.Address    //用户地址
	//矿工属性
	ms MinerInfo //和组无关的矿工信息（本质上可以跨多个GroupNode共享）
	//组（相关）属性
	mgs         MinerGroupSecret     //和组相关的矿工信息
	m_init_pool GroupInitPool        //组初始化消息池
	m_sign_sk   groupsig.Seckey      //输出：矿工签名私钥（由秘密共享接收池聚合而来）
	m_gpk       groupsig.Pubkey      //输出：组公钥（由矿工签名公钥接收池聚合而来）
	m_sign_pks  groupsig.PubkeyMapID //组成员签名公钥
}

func (n GroupNode) GenInnerGroup() JoinedGroup {
	var jg JoinedGroup
	jg.GroupPK = n.GetGroupPubKey()
	jg.SignKey = n.getSignSecKey()
	//jg.Members = n.m_init_pool.GenMemberPubKeys()
	jg.Members = n.m_sign_pks
	jg.GroupID = *groupsig.NewIDFromPubkey(jg.GroupPK)
	jg.SeedKey = n.mgs.GenSecKey()
	return jg
}

//用户初始化
func (n *GroupNode) InitUser(sk_str string) {
	n.u_sk = common.GenerateKey(sk_str)
	pk := n.u_sk.GetPubKey()
	n.u_address = pk.GetAddress()
}

//用户导入
func (n *GroupNode) ImportUser(sk common.PrivateKey, addr common.Address) {
	n.u_sk = sk
	n.u_address = addr
}

//矿工初始化(和组无关)
func (n *GroupNode) InitForMiner(id groupsig.ID, secret rand.Rand) {
	fmt.Printf("begin GroupNode::InitForMiner...\n")
	n.ms.Init(id, secret)
	return
}

//加入某个组初始化
func (n *GroupNode) InitForGroup(h common.Hash) {
	fmt.Printf("begin GroupNode::InitForGroup...\n")
	n.mgs = NewMinerGroupSecret(n.ms.GenSecretForGroup(h)) //生成用户针对该组的私密种子
	n.m_init_pool = *new(GroupInitPool)                    //初始化秘密接收池
	n.m_init_pool.init()
	n.m_sign_sk = *new(groupsig.Seckey) //初始化
	n.m_gpk = *new(groupsig.Pubkey)
	n.m_sign_pks = make(groupsig.PubkeyMapID, 0)
	return
}

//针对矿工的初始化(可以分两层，一个节点ID可以加入多个组)
func (n *GroupNode) InitForMinerStr(id string, secret string, gis ConsensusGroupInitSummary) {
	fmt.Printf("begin GroupNode::InitForMinerStr...\n")
	n.ms = NewMinerInfo(id, secret)
	n.mgs = NewMinerGroupSecret(n.ms.GenSecretForGroup(gis.GenHash()))

	n.m_init_pool = *new(GroupInitPool)
	n.m_init_pool.init()
	n.m_sign_sk = *new(groupsig.Seckey)
	n.m_gpk = *new(groupsig.Pubkey)
	return
}

func (n GroupNode) GetMinerID() groupsig.ID {
	return n.ms.MinerID
}

//生成针对组内所有成员的秘密共享
func (n *GroupNode) GenSharePiece(mems []groupsig.ID) groupsig.SeckeyMapID {
	shares := make(groupsig.SeckeyMapID)
	//生成门限个密钥
	secs := n.mgs.GenSecKeyList(GetGroupK())
	//生成成员数量个共享秘密
	for _, id := range mems { //组成员遍历
		shares[id.GetHexString()] = *groupsig.ShareSeckey(secs, id)
	}
	return shares
}

//接收秘密共享
//返回：0正常接收，-1异常，1完成签名私钥聚合和组公钥聚合
func (n *GroupNode) SetInitPiece(id groupsig.ID, share SharePiece) int {
	fmt.Printf("begin GroupNode::SetInitPiece...\n")
	if n.m_init_pool.ReceiveData(id, share) == -1 {
		return -1
	}
	if n.m_init_pool.GetSize() == GROUP_MAX_MEMBERS { //已经收到所有组内成员发送的秘密共享
		if n.BeingValidMiner() {
			return 1
		} else {
			return -1
		}
	}
	return 0
}

//接收签名公钥
//返回：0正常接收，-1异常，1收到全量组成员签名公钥（可以启动上链和通知）
func (n *GroupNode) SetSignPKPiece(id groupsig.ID, sign_pk groupsig.Pubkey) int {
	fmt.Printf("begin GroupNode::SetSignPKPiece...\n")
	if v, ok := n.m_sign_pks[id.GetHexString()]; ok {
		if v.IsEqual(sign_pk) {
			return 0
		} else {
			return -1 //两次收到的数据不一致
		}
	} else {
		n.m_sign_pks[id.GetHexString()] = sign_pk
		if len(n.m_sign_pks) == GROUP_MAX_MEMBERS { //已经收到所有组内成员发送的签名公钥
			return 1
		}
	}
	return 0
}

//成为有效矿工
func (n *GroupNode) BeingValidMiner() bool {
	if !n.m_gpk.IsValid() || n.m_sign_sk.IsValid() {
		n.m_gpk = *n.m_init_pool.GenGroupPubKey()         //生成组公钥
		n.m_sign_sk = *n.m_init_pool.GenMinerSignSecKey() //生成矿工签名私钥
	}
	return n.m_gpk.IsValid() && n.m_sign_sk.IsValid()
}

//取得（和组相关的）私密私钥（这个函数在正式版本中不提供）
func (n GroupNode) getSeedSecKey() groupsig.Seckey {
	return n.mgs.GenSecKey()
}

//取得签名私钥（这个函数在正式版本中不提供）
func (n GroupNode) getSignSecKey() groupsig.Seckey {
	return n.m_sign_sk
}

//取得（和组相关的）私密公钥
func (n GroupNode) GetSeedPubKey() groupsig.Pubkey {
	return *groupsig.NewPubkeyFromSeckey(n.getSeedSecKey())
}

//取得组公钥（在秘密交换后有效）
func (n GroupNode) GetGroupPubKey() groupsig.Pubkey {
	return n.m_gpk
}
