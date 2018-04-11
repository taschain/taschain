package core

import (
	"common"
	"time"
	"consensus/groupsig"
	"sync"
	"crypto/sha256"
	"hash"
	"encoding/json"
)

//区块头结构
type BlockHeader struct {
	//Trans_Hash_List：交易集哈希列表
	//Trans_Root_Hash：默克尔树根哈希
	Hash         common.Hash   // 本块的hash，to do : 是对哪些数据的哈希
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
	ExtraData    []byte
}

func (bh BlockHeader) GenHash() common.Hash {
	var h common.Hash
	//to do ：鸠兹完成
	return h
}

type Block struct {
	Header       *BlockHeader
	Transactions []*Transaction
}

var hasherPool = sync.Pool{
	New: func() interface{} {
		return sha256.New()
	},
}

// 计算sha256
func Sha256(blockByte []byte) []byte {
	hasher := hasherPool.Get().(hash.Hash)
	hasher.Reset()
	defer hasherPool.Put(hasher)

	hasher.Write(blockByte)
	return hasher.Sum(nil)

}

// 创始块
func GenesisBlock() *Block {
	block := new(Block)

	block.Header = &BlockHeader{
		ExtraData: Sha256([]byte("tas")),
	}

	blockByte, _ := json.Marshal(block)
	block.Header.Hash = common.BytesToHash(Sha256(blockByte))

	return block
}
