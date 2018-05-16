package core

import (
	"bytes"
	"common"
	"crypto/sha256"
	"encoding/json"
	"hash"
	"math/big"
	"sync"
	"time"
	"vm/core/state"
	"vm/core/types"
	"vm/rlp"
	"vm/trie"

	c "vm/common"
)

//区块头结构
type BlockHeader struct {
	Hash         common.Hash   // 本块的hash，to do : 是对哪些数据的哈希
	Height       uint64        // 本块的高度
	PreHash      common.Hash   //上一块哈希
	PreTime      time.Time     //上一块铸块时间
	QueueNumber  uint64        //轮转序号
	CurTime      time.Time     //当前铸块时间
	Castor       []byte        //出块人ID
	GroupId      []byte        //组ID，groupsig.ID的二进制表示
	Signature    []byte        // 组签名
	Nonce        uint64        //盐
	Transactions []common.Hash // 交易集哈希列表
	TxTree       common.Hash   // 交易默克尔树根hash
	ReceiptTree  common.Hash
	StateTree    common.Hash
	ExtraData    []byte
}

func (bh *BlockHeader) GenHash() common.Hash {
	sign := bh.Signature
	hash := bh.Hash

	bh.Signature = []byte{}
	bh.Hash = common.Hash{}
	blockByte, _ := json.Marshal(bh)
	result := common.BytesToHash(Sha256(blockByte))

	bh.Signature = sign
	bh.Hash = hash
	return result
}

var emptyHash = common.Hash{}

type Block struct {
	Header       *BlockHeader
	Transactions []*Transaction
}

func calcTxTree(tx []*Transaction) common.Hash {
	if nil == tx || 0 == len(tx) {
		return emptyHash
	}

	keybuf := new(bytes.Buffer)
	trie := new(trie.Trie)
	for i := 0; i < len(tx); i++ {
		keybuf.Reset()
		rlp.Encode(keybuf, uint(i))
		encode, _ := rlp.EncodeToBytes(tx[i])
		trie.Update(keybuf.Bytes(), encode)
	}
	hash := trie.Hash()

	return common.BytesToHash(hash.Bytes())
}

func calcReceiptsTree(receipts types.Receipts) common.Hash {
	if nil == receipts || 0 == len(receipts) {
		return emptyHash
	}

	keybuf := new(bytes.Buffer)
	trie := new(trie.Trie)
	for i := 0; i < len(receipts); i++ {
		keybuf.Reset()
		rlp.Encode(keybuf, uint(i))
		encode, _ := rlp.EncodeToBytes(receipts[i])
		trie.Update(keybuf.Bytes(), encode)
	}
	hash := trie.Hash()

	return common.BytesToHash(hash.Bytes())
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
func GenesisBlock(stateDB *state.StateDB, triedb *trie.Database) *Block {
	block := new(Block)

	block.Header = &BlockHeader{
		ExtraData: Sha256([]byte("tas")),
		CurTime:   time.Date(2018, 5, 16, 10, 0, 0, 0, time.Local),
	}

	blockByte, _ := json.Marshal(block)
	block.Header.Hash = common.BytesToHash(Sha256(blockByte))
	block.Header.Signature = Sha256([]byte("tas"))

	// 创始块账户创建
	stateDB.SetBalance(c.BytesToAddress(Sha256([]byte("1"))), big.NewInt(100))
	stateDB.SetBalance(c.BytesToAddress(Sha256([]byte("2"))), big.NewInt(200))
	stateDB.SetBalance(c.BytesToAddress(Sha256([]byte("3"))), big.NewInt(300))
	stateDB.IntermediateRoot(false)
	root, _ := stateDB.Commit(false)
	triedb.Commit(root, false)
	block.Header.StateTree = common.BytesToHash(root.Bytes())

	return block
}
