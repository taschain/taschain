package logical

import (
	"consensus/groupsig"
	"fmt"
	"time"

	"common"
)


//组铸块最大允许时间=10s
const MAX_GROUP_BLOCK_TIME int32 = 10

//个人出块最大允许时间=2s
const MAX_USER_CAST_TIME int32 = 2

//计算当前距上一个铸块完成已经过去了几个铸块时间窗口（组间）
func getBlockTimeWindow(b time.Time) int32 {
	diff := time.Since(b).Seconds() //从上个铸块完成到现在的时间（秒）
	if diff >= 0 {
		return int32(diff) / MAX_GROUP_BLOCK_TIME
	} else {
		return -1
	}
}

//计算当前距上一个铸块完成已经过去了几个出块时间窗口（组内）
func getCastTimeWindow(b time.Time) int32 {
	diff := time.Since(b).Seconds() //从上个铸块完成到现在的时间（秒）
	if diff >= 0 {
		return int32(diff) / MAX_USER_CAST_TIME
	} else {
		return -1
	}
}

//见证人处理器
type Processer struct {
	bc   BlockContext    //铸块上下文
	gg   GlobalGroups    //全网组信息
	gid  groupsig.ID     //所属组ID
	uid  groupsig.ID     //个人ID
	gusk groupsig.Seckey //组成员签名私钥（片）
	usk  groupsig.Seckey //组个人私钥（用组内成员列表处保存的个人公钥可以验签）
}

func (p Processer) isBHCastLegal(bh BlockHeader, sd SignData) (result bool) {
	//to do : 检查是否基于链上最高块的出块
	gi := p.gg.GetCastGroup(bh.PreHash) //取得合法的铸块组
	if gi.GroupID == sd.SignMember {
		//检查组签名是否正确
		result = sd.VerifySign(gi.GroupPK)
	}
	return result
}

//收到组内成员的出块消息，铸块人用组分片密钥进行了签名
func (p Processer) OnMessageCast(ccm ConsensusCastMessage) {
	if !p.bc.IsCasting() { //当前没有在组铸块中
		fmt.Printf("processer::OnMessageCast failed, group not in cast.\n")
		return
	}
	cs := GenConsensusSummary(ccm.bh, ccm.si)
	n := p.bc.UserCasted(cs)
	fmt.Printf("processer:OnMessageCast UserCasted result=%v.\n", n)
	if n == CBMR_THRESHOLD_SUCCESS {
		b := p.bc.VerifyGroupSign(cs, p.GetSelfGroup().GroupPK)
		if b { //验证通过
			//to do: 鸠兹上链，小熊广播
		}
	}
	return
}

func (p Processer) OnMessageVerify(cvm ConsensusVerifyMessage) {
	cs := GenConsensusSummary(cvm.bh, cvm.si)
	n := p.bc.UserVerified(cs)
	fmt.Printf("processer::OnMessageVerify UserVerified result=%v.\n", n)
	if n == CBMR_THRESHOLD_SUCCESS {
		b := p.bc.VerifyGroupSign(cs, p.GetSelfGroup().GroupPK)
		if b {
			//to do: 鸠兹上链，小熊广播
		}
	}
	return
}

//收到铸块上链消息
func (p Processer) OnMessageBlock(cbm ConsensusBlockMessage) {
	if p.isBHCastLegal(cbm.bh, cbm.si) { //铸块头合法
		//to do : 鸠兹上链保存
		next_group, err := p.gg.SelectNextGroup(cbm.si.DataHash) //查找下一个铸块组
		if err == nil {
			if next_group == p.gid { //自身属于下一个铸块组
				p.bc.BeingCastGroup(cbm.bh.BlockHeight, cbm.bh.PreTime, cbm.si.DataHash)
				//to do : 屮逸组内广播
			}
		} else {
			panic("find next cast group failed.")
		}
	} else {
		//丢弃该块
		fmt.Printf("received invalid new block, height = %v.\n", cbm.bh.BlockHeight)
	}
}

//收到成为当前组消息
func (p Processer) OnMessageCurrent(ccm ConsensusCurrentMessage) {
	gi, err := p.gg.GetGroupByID(p.gid)
	if err == nil {
		ru, ok := gi.GetMember(ccm.si.SignMember) //检查发消息用户是否跟当前节点同组
		if ok {                                   //该用户和我是同一组
			if ccm.si.VerifySign(ru.pubkey) { //消息验签
				p.bc.BeingCastGroup(ccm.BlockHeight, ccm.PreTime, ccm.PreHash)
				//to do : 屮逸组内广播
				//检查当前节点是否铸块节点
				pos := p.GetSelfGroup().GetPosition(p.uid) //当前节点在组内位置
				if pos >= 0 && getCastTimeWindow(ccm.PreTime) == int32(pos) {
					//当前节点为铸块节点
					p.CastBlock() //启动铸块
				}
			}
		}
	} else {
		panic("OnMessageCrrent failed, invalid gid.")
	}
}

///////////////////////////////////////////////////////////////////////////////
//取得自身所在的组
func (p Processer) GetSelfGroup() StaticGroupInfo {
	g, err := p.gg.GetGroupByID(p.gid)
	if err != nil {
		panic("GetSelfGroup failed.")
	}
	return g
}

//当前节点成为KING，出块
func (p Processer) CastBlock() {
	var bh BlockHeader
	var hash []byte
	//to do : 鸠兹生成bh和哈希
	//给鸠兹的参数：QN, nonce，castor
	var si SignData
	si.DataHash = common.BytesToHash(hash)
	si.SignMember = p.uid
	si.DataSign = groupsig.Sign(p.gusk, si.DataHash.Bytes()) //对区块头签名
	if bh.BlockHeight > 0 && si.DataSign.IsValid() {
		fmt.Printf("success cast block, height= %v, castor= %v.\n", bh.BlockHeight, bh.Castor.GetHexString())
	}
	//个人铸块完成的同时也是个人验证完成（第一个验证者）
	//更新共识上下文
	return
}
