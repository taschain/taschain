package logical

import (
	"common"
	"consensus/groupsig"
	"time"
)

type CONSENSUS_TYPE uint8

const (
	cast_block   CONSENSUS_TYPE = iota //铸块
	create_group                       //建组
)

//成为当前处理组消息 - 由第一个发现当前组成为铸块组的成员发出
type consensus_current_message struct {
	pre_hash       common.Hash    //上一块哈希
	pre_time       time.Time      //上一块完成时间
	consensus_type CONSENSUS_TYPE //共识类型
	block_height   uint64         //铸块高度
	instigator     groupsig.ID    //发起者（发现者）
	si             sign_data
}

//铸块消息 - 由成为KING的组成员发出
type consensus_cast_message struct {
	bh block_header
	si sign_data
}

//验证消息 - 由组内的验证人发出（对KING的铸块进行验证）
type consensus_verify_message struct {
	bh block_header
	si sign_data
}

//出块消息 - 该组成功完成了一个出块，由组内任意一个收集到k个签名的成员发出
//全量交易集在什么时候打到块内？铸块者还是广播出块者？
type consensus_block_message struct {
	bh block_header
	si sign_data
	//每个人在收到上一块后就知道下一块应该由哪个组铸出，提前可以从链上拿好下一个组的公钥，对msg_sign和msg_hash进行验证
	//交易集全量数据
}
