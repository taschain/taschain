package core

import (
	"bytes"
	"common"
	"encoding/json"
	"math/big"
	"time"
	"vm/core/state"
	vtypes "vm/core/types"
	"vm/rlp"
	"vm/trie"
	c "vm/common"
	"middleware/types"
)

var emptyHash = common.Hash{}

func calcTxTree(tx []*types.Transaction) common.Hash {
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

func calcReceiptsTree(receipts vtypes.Receipts) common.Hash {
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

// 创始块
func GenesisBlock(stateDB *state.StateDB, triedb *trie.Database) *types.Block {
	block := new(types.Block)

	block.Header = &types.BlockHeader{
		ExtraData:   common.Sha256([]byte("tas")),
		CurTime:     time.Date(2018, 6, 14, 10, 0, 0, 0, time.Local),
		QueueNumber: 0,
		TotalQN:     0,
	}

	blockByte, _ := json.Marshal(block)
	block.Header.Hash = common.BytesToHash(common.Sha256(blockByte))
	block.Header.Signature = common.Sha256([]byte("tas"))
	block.Header.RandSig = common.Sha256([]byte("tas genesis rand"))

	// 创始块账户创建
	stateDB.SetBalance(c.BytesToAddress(common.Sha256([]byte("1"))), big.NewInt(1000000))
	stateDB.SetBalance(c.BytesToAddress(common.Sha256([]byte("2"))), big.NewInt(2000000))
	stateDB.SetBalance(c.BytesToAddress(common.Sha256([]byte("3"))), big.NewInt(3000000))
	stateDB.SetBalance(c.BytesToAddress(common.Sha256([]byte("4"))), big.NewInt(1000000))
	stateDB.SetBalance(c.BytesToAddress(common.Sha256([]byte("5"))), big.NewInt(2000000))
	stateDB.SetBalance(c.BytesToAddress(common.Sha256([]byte("6"))), big.NewInt(3000000))
	stateDB.SetBalance(c.BytesToAddress(common.Sha256([]byte("7"))), big.NewInt(1000000))
	stateDB.SetBalance(c.BytesToAddress(common.Sha256([]byte("8"))), big.NewInt(2000000))
	stateDB.SetBalance(c.BytesToAddress(common.Sha256([]byte("9"))), big.NewInt(3000000))
	stateDB.SetBalance(c.BytesToAddress(common.Sha256([]byte("10"))), big.NewInt(1000000))
	stateDB.IntermediateRoot(false)
	root, _ := stateDB.Commit(false)
	triedb.Commit(root, false)
	block.Header.StateTree = common.BytesToHash(root.Bytes())

	return block
}
