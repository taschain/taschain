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
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/base"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/model"
	"sync"
)

//数据接收池
type GroupInitPool struct {
	pool model.SharePieceMap
}

func newGroupInitPool() *GroupInitPool {
	return &GroupInitPool{
		pool: make(model.SharePieceMap),
	}
}

//接收数据
func (gmd *GroupInitPool) ReceiveData(id groupsig.ID, piece model.SharePiece) int {
	stdLogger.Debugf("GroupInitPool::ReceiveData, sender=%v, share=%v, pub=%v...\n", id.ShortS(), piece.Share.ShortS(), piece.Pub.ShortS())
	if _, ok := gmd.pool[id.GetHexString()]; !ok {
		gmd.pool[id.GetHexString()] = piece //没有收到过该成员消息
		return 0
	} else { //收到过
		return -1
	}
}

func (gmd *GroupInitPool) GetSize() int {
	return len(gmd.pool)
}

//生成组成员签名公钥列表（用于铸块相关消息的验签）
func (gmd GroupInitPool) GenMemberPubKeys() groupsig.PubkeyMap {
	pubs := make(groupsig.PubkeyMap, 0)
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
	//矿工属性
	minerInfo *model.SelfMinerDO //和组无关的矿工信息（本质上可以跨多个GroupNode共享）
	//组（相关）属性
	minerGroupSecret MinerGroupSecret //和组相关的矿工信息
	memberNum        int              //组成员数量

	groupInitPool     GroupInitPool   //组初始化消息池
	minerSignedSeckey groupsig.Seckey //输出：矿工签名私钥（由秘密共享接收池聚合而来）
	groupPubKey       groupsig.Pubkey //输出：组公钥（由矿工签名公钥接收池聚合而来）
	//memberPubKeys     groupsig.PubkeyMap //组成员签名公钥

	lock sync.RWMutex
}

func (n GroupNode) threshold() int {
	return model.Param.GetGroupK(n.memberNum)
}

func (n GroupNode) GenInnerGroup(ghash common.Hash) *JoinedGroup {
	return newJoindGroup(&n, ghash)
}

//矿工初始化(和组无关)
func (n *GroupNode) InitForMiner(mi *model.SelfMinerDO) {
	//log.Printf("begin GroupNode::InitForMiner...\n")
	n.minerInfo = mi
	return
}

//加入某个组初始化
func (n *GroupNode) InitForGroup(h common.Hash) {
	//log.Printf("begin GroupNode::InitForGroup...\n")
	n.minerGroupSecret = NewMinerGroupSecret(n.minerInfo.GenSecretForGroup(h)) //生成用户针对该组的私密种子
	n.groupInitPool = *newGroupInitPool()                                      //初始化秘密接收池
	n.minerSignedSeckey = groupsig.Seckey{}                                    //初始化
	n.groupPubKey = groupsig.Pubkey{}
	//n.memberPubKeys = make(groupsig.PubkeyMap, 0)
	return
}

//针对矿工的初始化(可以分两层，一个节点ID可以加入多个组)
//func (n *GroupNode) InitForMinerStr(id string, secret string, gis model.ConsensusGroupInitSummary) {
//	log.Printf("begin GroupNode::InitForMinerStr...\n")
//	n.minerInfo = model.NewSelfMinerDO(id, secret)
//	n.minerGroupSecret = NewMinerGroupSecret(n.minerInfo.GenSecretForGroup(gis.GenHash()))
//
//	n.groupInitPool = *newGroupInitPool()
//	n.minerSignedSeckey = groupsig.Seckey{}
//	n.groupPubKey = groupsig.Pubkey{}
//	return
//}

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

func (n *GroupNode) getAllPiece() bool {
	return n.groupInitPool.GetSize() == n.memberNum
}

//接收秘密共享
//返回：0正常接收，-1异常，1完成签名私钥聚合和组公钥聚合
func (n *GroupNode) SetInitPiece(id groupsig.ID, share *model.SharePiece) int {
	n.lock.Lock()
	defer n.lock.Unlock()

	//log.Printf("begin GroupNode::SetInitPiece...\n")
	if n.groupInitPool.ReceiveData(id, *share) == -1 {
		return -1
	}
	if n.getAllPiece() { //已经收到所有组内成员发送的秘密共享
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
//func (n *GroupNode) SetSignPKPiece(spkm *model.ConsensusSignPubKeyMessage) int {
//	//log.Printf("begin GroupNode::SetSignPKPiece...\n")
//	idHex := spkm.SI.SignMember.GetHexString()
//	signPk := spkm.SignPK
//
//	n.lock.Lock()
//	defer n.lock.Unlock()
//
//	if v, ok := n.memberPubKeys[idHex]; ok {
//		if v.IsEqual(signPk) {
//			return 0
//		} else {
//			return -1 //两次收到的数据不一致
//		}
//	} else {
//		n.memberPubKeys[idHex] = signPk
//		if len(n.memberPubKeys) == n.memberNum { //已经收到所有组内成员发送的签名公钥
//			return 1
//		}
//	}
//	return 0
//}

//成为有效矿工
func (n *GroupNode) beingValidMiner() bool {
	if !n.groupPubKey.IsValid() || !n.minerSignedSeckey.IsValid() {
		n.groupPubKey = *n.groupInitPool.GenGroupPubKey()           //生成组公钥
		n.minerSignedSeckey = *n.groupInitPool.GenMinerSignSecKey() //生成矿工签名私钥
	}
	return n.groupPubKey.IsValid() && n.minerSignedSeckey.IsValid()
}

//取得（和组相关的）私密私钥（这个函数在正式版本中不提供）
func (n GroupNode) getSeedSecKey() groupsig.Seckey {
	return n.minerGroupSecret.GenSecKey()
}

//取得签名私钥（这个函数在正式版本中不提供）
func (n GroupNode) getSignSecKey() groupsig.Seckey {
	return n.minerSignedSeckey
}

//取得（和组相关的）私密公钥
func (n GroupNode) GetSeedPubKey() groupsig.Pubkey {
	return *groupsig.NewPubkeyFromSeckey(n.getSeedSecKey())
}

//取得组公钥（在秘密交换后有效）
func (n GroupNode) GetGroupPubKey() groupsig.Pubkey {
	return n.groupPubKey
}

func (n *GroupNode) hasPiece(id groupsig.ID) bool {
	n.lock.RLock()
	defer n.lock.RUnlock()
	_, ok := n.groupInitPool.pool[id.GetHexString()]
	return ok
}
