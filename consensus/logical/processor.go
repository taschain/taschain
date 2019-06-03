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
	"github.com/taschain/taschain/consensus/groupsig"

	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/consensus/net"
	"github.com/taschain/taschain/core"
	"github.com/taschain/taschain/middleware/notify"
	"github.com/taschain/taschain/middleware/ticker"
	"github.com/taschain/taschain/middleware/time"
	"github.com/taschain/taschain/middleware/types"
	"strings"
	"sync/atomic"
)

var ProcTestMode bool

//见证人处理器
type Processor struct {
	joiningGroups *JoiningGroups //已加入未完成初始化的组(组初始化完成上链后，不再需要)。组内成员数据过程数据。

	blockContexts *castBlockContexts

	globalGroups *GlobalGroups //全网组静态信息（包括待完成组初始化的组，即还没有组ID只有DUMMY ID的组）

	groupManager *GroupManager

	//////和组无关的矿工信息
	mi *model.SelfMinerDO
	//////加入(成功)的组信息(矿工节点数据)
	belongGroups *BelongGroups //当前ID参与了哪些(已上链，可铸块的)组, 组id_str->组内私密数据（组外不可见或加速缓存）
	//////测试数据，代替屮逸的网络消息
	GroupProcs map[string]*Processor
	Ticker     *ticker.GlobalTicker //全局定时器, 组初始化完成后启动

	//futureBlockMsgs  *FutureMessageHolder //存储缺少父块的块
	futureVerifyMsgs *FutureMessageHolder //存储缺失前一块的验证消息
	futureRewardReqs *FutureMessageHolder //块仍未上链的分红交易签名请求

	proveChecker *proveChecker

	ready         bool //是否已初始化完成
	genesisMember bool

	//////链接口
	MainChain  core.BlockChain
	GroupChain *core.GroupChain

	minerReader *MinerPoolReader
	vrf         atomic.Value //vrfWorker

	NetServer net.NetworkServer
	conf      common.ConfManager

	ts time.TimeService
}

func (p Processor) getPrefix() string {
	return p.GetMinerID().ShortS()
}

//私密函数，用于测试，正式版本不提供
func (p Processor) getMinerInfo() *model.SelfMinerDO {
	return p.mi
}

func (p Processor) GetPubkeyInfo() model.PubKeyInfo {
	return model.NewPubKeyInfo(p.mi.GetMinerID(), p.mi.GetDefaultPubKey())
}

func (p *Processor) setProcs(gps map[string]*Processor) {
	p.GroupProcs = gps
}

//初始化矿工数据（和组无关）
func (p *Processor) Init(mi model.SelfMinerDO, conf common.ConfManager) bool {
	p.ready = false
	p.conf = conf
	//p.futureBlockMsgs = NewFutureMessageHolder()
	p.futureVerifyMsgs = NewFutureMessageHolder()
	p.futureRewardReqs = NewFutureMessageHolder()
	p.MainChain = core.BlockChainImpl
	p.GroupChain = core.GroupChainImpl
	p.mi = &mi
	p.globalGroups = NewGlobalGroups(p.GroupChain)
	p.joiningGroups = NewJoiningGroups()
	p.belongGroups = NewBelongGroups(p.genBelongGroupStoreFile(), p.getEncryptPrivateKey())
	p.blockContexts = newCastBlockContexts(p.MainChain)
	p.NetServer = net.NewNetworkServer()
	p.proveChecker = newProveChecker(p.MainChain)
	p.ts = time.TSInstance

	p.minerReader = newMinerPoolReader(core.MinerManagerImpl)
	pkPoolInit(p.minerReader)

	p.groupManager = NewGroupManager(p)
	p.Ticker = ticker.NewGlobalTicker("consensus")

	if stdLogger != nil {
		stdLogger.Debugf("proc(%v) inited 2.\n", p.getPrefix())
		consensusLogger.Infof("ProcessorId:%v", p.getPrefix())
	}

	notify.BUS.Subscribe(notify.BlockAddSucc, p.onBlockAddSuccess)
	notify.BUS.Subscribe(notify.GroupAddSucc, p.onGroupAddSuccess)
	//notify.BUS.Subscribe(notify.NewBlock, p.onNewBlockReceive)

	jgFile := conf.GetString(ConsensusConfSection, "joined_group_store", "")
	if strings.TrimSpace(jgFile) == "" {
		jgFile = "joined_group.config." + common.GlobalConf.GetString("instance", "index", "")
	}
	p.belongGroups.joinedGroup2DBIfConfigExists(jgFile)

	return true
}

//取得矿工ID（和组无关）
func (p Processor) GetMinerID() groupsig.ID {
	return p.mi.GetMinerID()
}

func (p Processor) GetMinerInfo() *model.MinerDO {
	return &p.mi.MinerDO
}

////验证块的组签名是否正确
//func (p *Processor) verifyGroupSign(msg *model.ConsensusBlockMessage, preBH *types.BlockHeader) bool {
//	b := &msg.Block
//	bh := b.Header
//	var gid groupsig.ID
//	if gid.Deserialize(bh.GroupID) != nil {
//		panic("verifyGroupSign: group id Deserialize failed.")
//	}
//
//	blog := newBizLog("verifyGroupSign")
//	groupInfo := p.GetGroup(gid)
//	if !groupInfo.GroupID.IsValid() {
//		blog.log("get group is nil!, gid=" + gid.ShortS())
//		return false
//	}
//
//	//blog.log("gpk %v, bh hash %v, sign %v, rand %v", groupInfo.GroupPK.ShortS(), bh.Hash.ShortS(), bh.Signature, bh.Random)
//	if !msg.VerifySig(groupInfo.GroupPK, preBH.Random) {
//		blog.log("verifyGroupSig fail")
//		return false
//	}
//	return true
//}

//检查铸块组是否合法
func (p *Processor) isCastLegal(bh *types.BlockHeader, preHeader *types.BlockHeader) (ok bool, group *StaticGroupInfo, err error) {
	castor := groupsig.DeserializeID(bh.Castor)
	minerDO := p.minerReader.getProposeMiner(castor)
	if minerDO == nil {
		err = fmt.Errorf("minerDO is nil, id=%v", castor.ShortS())
		return
	}
	if !minerDO.CanCastAt(bh.Height) {
		err = fmt.Errorf("miner can't cast at height, id=%v, height=%v(%v-%v)", castor.ShortS(), bh.Height, minerDO.ApplyHeight, minerDO.AbortHeight)
		return
	}
	totalStake := p.minerReader.getTotalStake(preHeader.Height, false)
	if ok2, err2 := vrfVerifyBlock(bh, preHeader, minerDO, totalStake); !ok2 {
		err = fmt.Errorf("vrf verify block fail, err=%v", err2)
		return
	}

	var gid = groupsig.DeserializeID(bh.GroupID)

	selectGroupIDFromCache := p.CalcVerifyGroupFromCache(preHeader, bh.Height)

	if selectGroupIDFromCache == nil {
		err = common.ErrSelectGroupNil
		stdLogger.Debugf("selectGroupId is nil")
		return
	}
	var verifyGid = *selectGroupIDFromCache

	if !selectGroupIDFromCache.IsEqual(gid) { //有可能组已经解散，需要再从链上取
		selectGroupIDFromChain := p.CalcVerifyGroupFromChain(preHeader, bh.Height)
		if selectGroupIDFromChain == nil {
			err = common.ErrSelectGroupNil
			return
		}
		//若内存与链不一致，则启动更新
		if !selectGroupIDFromChain.IsEqual(*selectGroupIDFromCache) {
			go p.updateGlobalGroups()
		}
		if !selectGroupIDFromChain.IsEqual(gid) {
			err = common.ErrSelectGroupInequal
			stdLogger.Debugf("selectGroupId from both cache and chain not equal, expect %v, receive %v", selectGroupIDFromChain.ShortS(), gid.ShortS())
			return
		}
		verifyGid = *selectGroupIDFromChain
	}

	group = p.GetGroup(verifyGid) //取得合法的铸块组
	if !group.GroupID.IsValid() {
		err = fmt.Errorf("selectedGroup is not valid, expect gid=%v, real gid=%v", verifyGid.ShortS(), group.GroupID.ShortS())
		return
	}

	ok = true
	return
}

func (p *Processor) getMinerPos(gid groupsig.ID, uid groupsig.ID) int32 {
	sgi := p.GetGroup(gid)
	return int32(sgi.GetMinerPos(uid))
}

///////////////////////////////////////////////////////////////////////////////
////取得自己参与的某个铸块组的公钥片段（聚合一个组所有成员的公钥片段，可以生成组公钥）
//func (p Processor) GetMinerPubKeyPieceForGroup(gid groupsig.ID) groupsig.Pubkey {
//	var pub_piece groupsig.Pubkey
//	gc := p.joiningGroups.GetGroup(gid)
//	node := gc.GetNode()
//	if node != nil {
//		pub_piece = node.GetSeedPubKey()
//	}
//	return pub_piece
//}
//
////取得自己参与的某个铸块组的私钥片段（聚合一个组所有成员的私钥片段，可以生成组私钥）
////用于测试目的，正式版对外不提供。
//func (p Processor) getMinerSecKeyPieceForGroup(gid groupsig.ID) groupsig.Seckey {
//	var secPiece groupsig.Seckey
//	gc := p.joiningGroups.GetGroup(gid)
//	node := gc.GetNode()
//	if node != nil {
//		secPiece = node.getSeedSecKey()
//	}
//	return secPiece
//}

//取得特定的组
func (p Processor) GetGroup(gid groupsig.ID) *StaticGroupInfo {
	if g, err := p.globalGroups.GetGroupByID(gid); err != nil {
		panic("GetSelfGroup failed.")
	} else {
		return g
	}
}

//取得一个铸块组的公钥(processer初始化时从链上加载)
func (p Processor) getGroupPubKey(gid groupsig.ID) groupsig.Pubkey {
	if g, err := p.globalGroups.GetGroupByID(gid); err != nil {
		panic("GetSelfGroup failed.")
	} else {
		return g.GetPubKey()
	}

}

func outputBlockHeaderAndSign(prefix string, bh *types.BlockHeader, si *model.SignData) {
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

func (p *Processor) ExistInGroup(gHash common.Hash) bool {
	//initingGroup := p.globalGroups.GetInitedGroup(gHash)
	//if initingGroup == nil {
	//	return false
	//}
	//return initingGroup.MemberExist(p.GetMinerID())
	return false
}

func (p *Processor) getEncryptPrivateKey() common.PrivateKey {
	seed := p.mi.SK.GetHexString() + p.mi.ID.GetHexString()
	encryptPrivateKey := common.GenerateKey(seed)
	return encryptPrivateKey
}

func (p *Processor) getDefaultSeckeyInfo() model.SecKeyInfo {
	return model.NewSecKeyInfo(p.GetMinerID(), p.mi.GetDefaultSecKey())
}

func (p *Processor) getInGroupSeckeyInfo(gid groupsig.ID) model.SecKeyInfo {
	return model.NewSecKeyInfo(p.GetMinerID(), p.getSignKey(gid))
}