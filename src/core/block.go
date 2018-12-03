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

package core

import (
	"bytes"
	"common"
	"encoding/json"
	"github.com/vmihailenco/msgpack"
	"math/big"
	"time"
	"storage/core"
	vtypes "storage/core/types"
	"storage/trie"
	"middleware/types"
	"storage/serialize"
)

var emptyHash = common.Hash{}

func calcTxTree(tx []*types.Transaction) common.Hash {
	if nil == tx || 0 == len(tx) {
		return emptyHash
	}

	//keybuf := new(bytes.Buffer)
	//trie := new(trie.Trie)
	//for i := 0; i < len(tx); i++ {
	//	if tx[i] != nil {
	//		keybuf.Reset()
	//		serialize.Encode(keybuf, uint(i))
	//		encode, _ := serialize.EncodeToBytes(tx[i])
	//		trie.Update(keybuf.Bytes(), encode)
	//	}
	//}
	//hash := trie.Hash()
	//
	//return common.BytesToHash(hash.Bytes())

	buf := new(bytes.Buffer)
	for i := 0; i < len(tx); i++ {
		encode, _ := msgpack.Marshal(tx[i])
		serialize.Encode(buf, encode)
	}
	return common.BytesToHash(common.Sha256(buf.Bytes()))
}

func calcReceiptsTree(receipts vtypes.Receipts) common.Hash {
	if nil == receipts || 0 == len(receipts) {
		return emptyHash
	}

	keybuf := new(bytes.Buffer)
	trie := new(trie.Trie)
	for i := 0; i < len(receipts); i++ {
		if receipts[i] != nil {
			keybuf.Reset()
			serialize.Encode(keybuf, uint(i))
			encode, _ := serialize.EncodeToBytes(receipts[i])
			trie.Update(keybuf.Bytes(), encode)
		}
	}
	hash := trie.Hash()

	return common.BytesToHash(hash.Bytes())
}

// 创始块
func GenesisBlock(stateDB *core.AccountDB, triedb *trie.Database) *types.Block {
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
	block.Header.Random = common.Sha256([]byte("tas_initial_random"))

	// 创始块账户创建
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("1"))), big.NewInt(1000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("2"))), big.NewInt(2000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("3"))), big.NewInt(3000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("4"))), big.NewInt(1000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("5"))), big.NewInt(2000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("6"))), big.NewInt(3000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("7"))), big.NewInt(1000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("8"))), big.NewInt(2000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("9"))), big.NewInt(3000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("10"))), big.NewInt(1000000))
	stateDB.IntermediateRoot(false)
	root, _ := stateDB.Commit(false)
	triedb.Commit(root, false)
	block.Header.StateTree = common.BytesToHash(root.Bytes())

	return block
}
