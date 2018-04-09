package core

import (
	"common"
	"time"
	"consensus/groupsig"
)

//区块头结构
type BlockHeader struct {
	//Trans_Hash_List：交易集哈希列表
	//Trans_Root_Hash：默克尔树根哈希
	Hash         common.Hash   // 本块的hash
	Height       uint64        // 本块的高度
	PreHash      common.Hash   //上一块哈希
	PreTime      time.Time     //上一块铸块时间
	BlockHeight  uint64        //铸块高度
	QueueNumber  uint64        //轮转序号
	CurTime      time.Time     //当前铸块时间
	Castor       groupsig.ID   //铸块人(ID同时决定了铸块人的权重)
	Signature    common.Hash   // 组签名
	Nonce        uint64        //盐
	Transactions []common.Hash // 交易集哈希列表
	TxTree       common.Hash   // 交易默克尔树根hash
	ReceiptTree  common.Hash
	StateTree    common.Hash
	ExtraData    []int8
}

type Block struct {
	header       *BlockHeader
	transactions []*Transaction
}
