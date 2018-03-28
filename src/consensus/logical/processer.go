package logical

import (
	"consensus/groupsig"
	"fmt"
	"hash"
	"math/big"
	"time"

	"common"
)

//组铸块最大允许时间=10s
const MAX_GROUP_BLOCK_TIME int32 = 10

//个人出块最大允许时间=2s
const MAX_USER_CAST_TIME int32 = 2

//铸币基础参数
type cast_base_info struct {
	prev_hash      hash.Hash //上个区块哈希
	prev_sign      big.Int   //上个区块的组签名值
	prev_timestamp time.Time //上个区块的生成时间
	cur_index      int64     //当前待铸块高度
}

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
type processer struct {
	cc   block_context   //铸块上下文
	gg   GlobalGroups    //全网组信息
	gid  groupsig.ID     //所属组ID
	uid  groupsig.ID     //个人ID
	gusk groupsig.Seckey //组成员签名私钥（片）
	usk  groupsig.Seckey //组个人私钥（用组内成员列表处保存的个人公钥可以验签）
}

func (p processer) isBHCastLegal(bh block_header, sd sign_data) (result bool) {
	//检查是否基于链上最高块的出块
	gi := p.gg.GetCastGroup(bh.pre_hash) //取得合法的铸块组
	if gi.group_id == sd.id {
		//检查组签名是否正确
		result = sd.VerifySign(gi.group_pk)
	}
	return result
}

//收到铸块上链消息
func (p processer) OnMessageBlock(cbm consensus_block_message) {
	if p.isBHCastLegal(cbm.bh, cbm.si) { //铸块头合法
		//to do : 鸠兹上链保存
		next_group, err := p.gg.SelectNextGroup(cbm.si.data_hash) //查找下一个铸块组
		if err == nil {
			if next_group == p.gid { //自身属于下一个铸块组
				p.cc.Begin_Cast(cbm.bh.block_height, cbm.bh.pre_time, cbm.si.data_hash)
				//to do : 屮逸组内广播
			}
		} else {
			panic("find next cast group failed.")
		}
	} else {
		//丢弃该块
		fmt.Printf("received invalid new block, height = %v.\n", cbm.bh.block_height)
	}
}

//收到成为当前组消息
func (p processer) OnMessageCurrent(ccm consensus_current_message) {
	gi, err := p.gg.GetGroupByID(p.gid)
	if err == nil {
		ru, ok := gi.GetMember(ccm.si.id) //检查发消息用户是否跟当前节点同组
		if ok {                           //该用户和我是同一组
			if ccm.si.VerifySign(ru.pubkey) { //消息验签
				p.cc.Begin_Cast(ccm.block_height, ccm.pre_time, ccm.pre_hash)
				//to do : 屮逸组内广播
				//检查当前节点是否铸块节点
				pos := p.GetSelfGroup().GetPosition(p.uid) //当前节点在组内位置
				if pos >= 0 && getCastTimeWindow(ccm.pre_time) == int32(pos) {
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
func (p processer) GetSelfGroup() StaticGroupInfo {
	g, err := p.gg.GetGroupByID(p.gid)
	if err != nil {
		panic("GetSelfGroup failed.")
	}
	return g
}

func (p processer) CastBlock() {
	var bh block_header
	var hash []byte
	//to do : 鸠兹生成bh和哈希
	//给鸠兹的参数：QN, nonce，castor
	var si sign_data
	si.data_hash = common.BytesToHash(hash)
	si.id = p.uid
	si.data_sign = groupsig.Sign(p.gusk, si.data_hash.Bytes()) //对区块头签名
	if bh.block_height > 0 && si.data_sign.GetHexString() != "" {
		fmt.Printf("success cast block, height= %v, castor= %v.\n", bh.block_height, bh.castor.GetHexString())
	}
	//个人铸块完成的同时也是个人验证完成（第一个验证者）
	//更新共识上下文
	return
}
