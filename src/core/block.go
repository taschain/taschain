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
	"math/big"
	"time"
	"storage/core"
	vtypes "storage/core/types"
	"storage/trie"
	"middleware/types"
	"storage/serialize"
	"github.com/vmihailenco/msgpack"
)

var emptyHash = common.Hash{}

func calcTxTree(tx []*types.Transaction) common.Hash {
	if nil == tx || 0 == len(tx) {
		return emptyHash
	}

	//keybuf := new(bytes.Buffer)
	//trie := new(trie.Trie)
	//Logger.Infof("calcTxTree transaction size:%d",len(tx))
	//for i := 0; i < len(tx); i++ {
	//	if tx[i] != nil {
	//		keybuf.Reset()
	//		serialize.Encode(keybuf, uint(i))
	//		//encode, _ := serialize.EncodeToBytes(tx[i])
	//		encode, _ := msgpack.Marshal(tx[i])
	//		len1 := -1
	//		if tx[i].Data != nil{
	//			len1 = len(tx[i].Data)
	//		}
	//		len2 := -1
	//		if tx[i].ExtraData != nil{
	//			len2 = len(tx[i].ExtraData)
	//		}
	//		Logger.Infof("calcTxTree %d len1:%d len2:%d source1:%s target1:%s %v",i,len1,len2,
	//			tx[i].Source.GetHexString(),tx[i].Target.GetHexString(),encode)
	//		trie.Update(keybuf.Bytes(), encode)
	//	} else {
	//		Logger.Error("calcTxTree exist empty transaction %d",i)
	//	}
	//}
	//hash := trie.Hash()
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
func GenesisBlock(stateDB *core.AccountDB, triedb *trie.Database,genesisInfo *types.GenesisInfo) *types.Block {
	block := new(types.Block)
	pv := big.NewInt(0)
	block.Header = &types.BlockHeader{
		ExtraData:  common.Sha256([]byte("tas")),
		CurTime:    time.Date(2018, 6, 14, 10, 0, 0, 0, time.Local),
		ProveValue: pv,
		TotalQN:    0,
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
	stateDB.SetBalance(common.HexStringToAddress("0xb26d797d6c29b60cd6a7f7eebf03c19a683f36ecb78643bd18318fbd1b739b09"), big.NewInt(1000000))
	stateDB.SetBalance(common.HexStringToAddress("0xa88ebed9c691f709788da55aa61548f23fad2f20e19f7c4cf8997894cd90662d"), big.NewInt(1000000))
	stateDB.SetBalance(common.HexStringToAddress("0x60113e78f3fec8482a23df56b1a49c11e6017e3c193fb42a4837585aa2cef9ac"), big.NewInt(1000000))
	stateDB.SetBalance(common.HexStringToAddress("0x31e59225ec0f5eb904899541ab91e23dbc73115509711901ee4d20f0d51f777a"), big.NewInt(1000000))
	stateDB.SetBalance(common.HexStringToAddress("0x273cf6a73494922bd207a40e02f4b4540586fa2b71fb05dada16e65bde62b51"), big.NewInt(1000000))
	stage := stateDB.IntermediateRoot(false)
	Logger.Debugf("GenesisBlock Stage1 Root:%s",stage.Hex())
	miners := make([]*types.Miner,0)
	for i,member := range genesisInfo.Group.Members{
		miner := &types.Miner{Id:member.Id,PublicKey:member.PubKey,VrfPublicKey:genesisInfo.VrfPKs[i],Stake:10}
		miners = append(miners,miner)
	}
	MinerManagerImpl.AddGenesesMiner(miners, stateDB)
	stage = stateDB.IntermediateRoot(false)
	Logger.Debugf("GenesisBlock Stage2 Root:%s",stage.Hex())
	stateDB.SetNonce(common.BonusStorageAddress,1)
	stateDB.SetNonce(common.HeavyDBAddress,1)
	stateDB.SetNonce(common.LightDBAddress,1)

	root, _ := stateDB.Commit(true)
	Logger.Debugf("GenesisBlock final Root:%s",root.Hex())
	triedb.Commit(root, false)
	block.Header.StateTree = common.BytesToHash(root.Bytes())
	return block
}
