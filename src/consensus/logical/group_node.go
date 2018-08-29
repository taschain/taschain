package logical

import (
	"common"
	"consensus/groupsig"
	"log"
	"sync"
	"consensus/model"
	"consensus/base"
)

//数据接收池
type GroupInitPool struct {
	pool model.ShareMapID
}

func newGroupInitPool() *GroupInitPool {
	return &GroupInitPool{
		pool: make(model.ShareMapID),
	}
}

//接收数据
func (gmd *GroupInitPool) ReceiveData(id groupsig.ID, piece model.SharePiece) int {
	log.Printf("GroupInitPool::ReceiveData, sender=%v, share=%v, pub=%v...\n", GetIDPrefix(id), GetSecKeyPrefix(piece.Share), GetPubKeyPrefix(piece.Pub))
	if _, ok := gmd.pool[id.GetHexString()]; !ok {
		gmd.pool[id.GetHexString()] = piece //没有收到过该成员消息
		return 0
	} else { //收到过
		if !gmd.pool[id.GetHexString()].IsEqual(piece) { //两次数据不一致
			log.Printf("GroupInitPool::ReceiveData failed, data diff.\n")
			return -1
		}
	}
	return 0
}

func (gmd *GroupInitPool) GetSize() int {
	return len(gmd.pool)
}

//生成组成员签名公钥列表（用于铸块相关消息的验签）
func (gmd GroupInitPool) GenMemberPubKeys() groupsig.PubkeyMapID {
	pubs := make(groupsig.PubkeyMapID, 0)
	for k, v := range gmd.pool {
		pubs[k] = v.Pub
	}
	return pubs
}

//生成矿工签名私钥
func (gmd GroupInitPool) GenMinerSignSecKey() *groupsig.Seckey {
	shares := make([]groupsig.Seckey, 0)
	for _, v := range gmd.pool {
		shares = append(shares, v.Share)
	}
	sk := groupsig.AggregateSeckeys(shares)
	return sk
}

//生成组公钥
func (gmd GroupInitPool) GenGroupPubKey() *groupsig.Pubkey {
	pubs := make([]groupsig.Pubkey, 0)
	for _, v := range gmd.pool {
		pubs = append(pubs, v.Pub)
	}
	gpk := groupsig.AggregatePubkeys(pubs)
	return gpk
}

//组相关的秘密
type MinerGroupSecret struct {
	secretSeed base.Rand //某个矿工针对某个组的私密种子（矿工个人私密种子固定和组信息固定的情况下，该值固定）
}

func NewMinerGroupSecret(secret base.Rand) MinerGroupSecret {
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
	privateKey common.PrivateKey //用户私钥（非组签名私钥）
	address    common.Address    //用户地址
	//矿工属性
	minerInfo model.MinerInfo //和组无关的矿工信息（本质上可以跨多个GroupNode共享）
	//组（相关）属性
	minerGroupSecret  MinerGroupSecret     //和组相关的矿工信息
	memberNum		int					//组成员数量

	groupInitPool     GroupInitPool        //组初始化消息池
	minerSignedSecret groupsig.Seckey      //输出：矿工签名私钥（由秘密共享接收池聚合而来）
	groupPubKey       groupsig.Pubkey      //输出：组公钥（由矿工签名公钥接收池聚合而来）
	memberPubKeys     groupsig.PubkeyMapID //组成员签名公钥
	groupSecretSignMap	map[string]groupsig.Signature	//组成员秘密签名
	groupSecret		*GroupSecret	//输出: 由signMap恢复出组秘密签名

	lock sync.RWMutex
}

func (n GroupNode) threshold() int {
    return model.Param.GetGroupK(n.memberNum)
}

func (n GroupNode) GenInnerGroup() *JoinedGroup {
	gpk := n.GetGroupPubKey()
	joinedGroup := &JoinedGroup{
		GroupPK: gpk,
		SignKey: n.getSignSecKey(),
		Members: n.memberPubKeys,
		GroupID: *groupsig.NewIDFromPubkey(gpk),
		SeedKey: n.minerGroupSecret.GenSecKey(),
	}

	//log.Println("GroupPK:", joinedGroup.GroupPK.Serialize())

	if n.groupSecret != nil {
		joinedGroup.GroupSec = *n.groupSecret
	}
	return joinedGroup
}

//用户初始化
func (n *GroupNode) InitUser(skStr string) {
	n.privateKey = common.GenerateKey(skStr)
	pk := n.privateKey.GetPubKey()
	n.address = pk.GetAddress()
}

//用户导入
func (n *GroupNode) ImportUser(sk common.PrivateKey, addr common.Address) {
	n.privateKey = sk
	n.address = addr
}

//矿工初始化(和组无关)
func (n *GroupNode) InitForMiner(id groupsig.ID, secret base.Rand) {
	//log.Printf("begin GroupNode::InitForMiner...\n")
	n.minerInfo.Init(id, secret)
	return
}

//加入某个组初始化
func (n *GroupNode) InitForGroup(h common.Hash) {
	//log.Printf("begin GroupNode::InitForGroup...\n")
	n.minerGroupSecret = NewMinerGroupSecret(n.minerInfo.GenSecretForGroup(h)) //生成用户针对该组的私密种子
	n.groupInitPool = *newGroupInitPool()                               //初始化秘密接收池
	n.minerSignedSecret = groupsig.Seckey{} //初始化
	n.groupPubKey = groupsig.Pubkey{}
	n.memberPubKeys = make(groupsig.PubkeyMapID, 0)
	n.groupSecretSignMap = make(map[string]groupsig.Signature)
	return
}

//针对矿工的初始化(可以分两层，一个节点ID可以加入多个组)
func (n *GroupNode) InitForMinerStr(id string, secret string, gis model.ConsensusGroupInitSummary) {
	log.Printf("begin GroupNode::InitForMinerStr...\n")
	n.minerInfo = model.NewMinerInfo(id, secret)
	n.minerGroupSecret = NewMinerGroupSecret(n.minerInfo.GenSecretForGroup(gis.GenHash()))

	n.groupInitPool = *newGroupInitPool()
	n.minerSignedSecret = groupsig.Seckey{}
	n.groupPubKey = groupsig.Pubkey{}
	return
}

func (n GroupNode) GetMinerID() groupsig.ID {
	return n.minerInfo.MinerID
}

//生成针对组内所有成员的秘密共享
func (n *GroupNode) GenSharePiece(mems []groupsig.ID) groupsig.SeckeyMapID {
	shares := make(groupsig.SeckeyMapID)
	//生成门限个密钥
	secs := n.minerGroupSecret.GenSecKeyList(n.threshold())
	//生成成员数量个共享秘密
	for _, id := range mems { //组成员遍历
		shares[id.GetHexString()] = *groupsig.ShareSeckey(secs, id)
	}
	return shares
}

//接收秘密共享
//返回：0正常接收，-1异常，1完成签名私钥聚合和组公钥聚合
func (n *GroupNode) SetInitPiece(id groupsig.ID, share model.SharePiece) int {
	n.lock.Lock()
	defer n.lock.Unlock()

	//log.Printf("begin GroupNode::SetInitPiece...\n")
	if n.groupInitPool.ReceiveData(id, share) == -1 {
		return -1
	}
	if n.groupInitPool.GetSize() == n.memberNum { //已经收到所有组内成员发送的秘密共享
		if n.beingValidMiner() {
			return 1
		} else {
			return -1
		}
	}
	return 0
}

//接收签名公钥
//返回：0正常接收，-1异常，1收到全量组成员签名公钥（可以启动上链和通知）
func (n *GroupNode) SetSignPKPiece(spkm *model.ConsensusSignPubKeyMessage) int {
	//log.Printf("begin GroupNode::SetSignPKPiece...\n")
	idHex := spkm.SI.SignMember.GetHexString()
	signPk := spkm.SignPK
	gisHash := spkm.GISHash
	gisSign := spkm.GISSign
	n.groupSecretSignMap[idHex] = gisSign

	n.lock.Lock()
	defer n.lock.Unlock()

	if v, ok := n.memberPubKeys[idHex]; ok {
		if v.IsEqual(signPk) {
			return 0
		} else {
			return -1 //两次收到的数据不一致
		}
	} else {
		n.memberPubKeys[idHex] = signPk
		if len(n.memberPubKeys) == n.memberNum { //已经收到所有组内成员发送的签名公钥
			gisSign = *groupsig.RecoverSignatureByMapI(n.groupSecretSignMap, n.threshold())
			if !groupsig.VerifySig(n.groupPubKey, gisHash.Bytes(), gisSign) {
				log.Printf("recover group secret gisSign failed!\n")
				return -1
			}
			n.groupSecret = NewGroupSecret(gisSign, 0, gisHash)
			return 1
		}
	}
	return 0
}

//成为有效矿工
func (n *GroupNode) beingValidMiner() bool {
	if !n.groupPubKey.IsValid() || !n.minerSignedSecret.IsValid() {
		n.groupPubKey = *n.groupInitPool.GenGroupPubKey()           //生成组公钥
		n.minerSignedSecret = *n.groupInitPool.GenMinerSignSecKey() //生成矿工签名私钥
	}
	return n.groupPubKey.IsValid() && n.minerSignedSecret.IsValid()
}

//取得（和组相关的）私密私钥（这个函数在正式版本中不提供）
func (n GroupNode) getSeedSecKey() groupsig.Seckey {
	return n.minerGroupSecret.GenSecKey()
}

//取得签名私钥（这个函数在正式版本中不提供）
func (n GroupNode) getSignSecKey() groupsig.Seckey {
	return n.minerSignedSecret
}

//取得（和组相关的）私密公钥
func (n GroupNode) GetSeedPubKey() groupsig.Pubkey {
	return *groupsig.NewPubkeyFromSeckey(n.getSeedSecKey())
}

//取得组公钥（在秘密交换后有效）
func (n GroupNode) GetGroupPubKey() groupsig.Pubkey {
	return n.groupPubKey
}
