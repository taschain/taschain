package logical

import (
	"common"
	"consensus/groupsig"
	"time"
)

//区块头结构
type block_header struct {
	//trans_hash_list：交易集哈希列表
	//trans_root_hash：默克尔树根哈希
	pre_hash     common.Hash //上一块哈希
	pre_time     time.Time   //上一块铸块时间
	block_height uint64      //铸块高度
	queue_number uint32      //轮转序号
	cur_time     time.Time   //当前铸块时间
	castor       groupsig.ID //铸块人(ID同时决定了铸块人的权重)
	nonce        uint64      //盐
}

//数据签名结构
type SignData struct {
	data_hash common.Hash        //哈希值
	data_sign groupsig.Signature //签名
	id        groupsig.ID        //用户ID或组ID，看消息类型
}

func (sd SignData) VerifySign(pk groupsig.Pubkey) bool {
	return groupsig.VerifySig(pk, sd.data_hash.Bytes(), sd.data_sign)
}
