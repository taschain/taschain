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
func GenesisBlock(stateDB *core.AccountDB, triedb *trie.Database, genesisInfo *types.GenesisInfo) *types.Block {
	block := new(types.Block)
	pv := big.NewInt(0)
	block.Header = &types.BlockHeader{
		Height:       0,
		ExtraData:    common.Sha256([]byte("tas")),
		CurTime:      time.Date(2018, 6, 14, 10, 0, 0, 0, time.Local),
		ProveValue:   pv,
		TotalQN:      0,
		Transactions: make([]common.Hash, 0), //important!!
	}

	//blockByte, _ := json.Marshal(block)
	//block.Header.Hash = common.BytesToHash(common.Sha256(blockByte))
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
	//小熊本地测试轻节点账户
	stateDB.SetBalance(common.HexStringToAddress("0xa88ebed9c691f709788da55aa61548f23fad2f20e19f7c4cf8997894cd90662d"), big.NewInt(1000000))
	stateDB.SetBalance(common.HexStringToAddress("0x60113e78f3fec8482a23df56b1a49c11e6017e3c193fb42a4837585aa2cef9ac"), big.NewInt(1000000))
	stateDB.SetBalance(common.HexStringToAddress("0x31e59225ec0f5eb904899541ab91e23dbc73115509711901ee4d20f0d51f777a"), big.NewInt(1000000))
	//阿里云账户
	stateDB.SetBalance(common.HexStringToAddress("0x1dd93e465350d356b873c9f41266bffd28e1b4125ac138e7393ee3c966c11a3c"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x5a664fe67e5a6d6eee691b5e7b9711e92ed20ae31ea6706db8c88da0cf7ab19c"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x51d8148462cef13d43aed048c670c1890fd24dac978dd68aa7a951db1256308e"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x9f4cd2913a88f008bfa601bc2495648e388563f1a78a20e5c349bf889f1b3d93"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x9911b24c4551d0e7dfea9ed72a5ba8fddfe48529c6656ce7840ebf0ff3e71fa7"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x5ab597f24ba3ad763aea01e5ca38942af20f12fa4cd21be12b55232422fc8ac6"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x1161f17d8ca9e4c0cb662ca9da1f57eb0e21b02fc483a8ceb907123ea80366d3"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xef1025e17a0c6a8f6f490b7d8fd45d6278d62f19ae17f8f0d09d3d553eed7160"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xb2e092adf117acdada6afa0ecf56bbda0191d1bbb0a7b569477773fe2a624cdb"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x5e6e2e73fb9e64d6f8afbfea818b4028545a920b3974969789abe5bddc8c4b76"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x325cf9d453f6d3957ed5a47c90a25d46a07b4cf6925b5be995057ace566d5ea5"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x85c423d8b8ac94d3fe9df757f51f8bbc7b30383df3d99bece24b883b8b2b00ad"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xa63688b2a23ae9684606af3db29f75bec656f238a751e4732953296a9d132be5"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x5f3e8c95eaed50cf19d71d7a7c46123d14571cb11665047e2f8f78b9df2db8aa"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x16cfc134a9b1080f28a8b36ac546f779c50f41b9d95bdc3f02ae931cc540fe2c"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x6619269bed0ddbca93563222ec6d6986b27e7ce839b8b58a7fe08db74aea4569"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x92732c644d0f9287900e74f1e393ac93486e1bdd25f9e77535bad675092f39b1"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x8c829396e9d818ef83334086105e904eafe921b2825f0c17717142af80f8fcda"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xd6f1a8de59ad8045abf5340660ffb3858bbc63d64e838dfae8e6191cd47a0ef0"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x9328c62a0850543120a6fceffbd6ece8fcebecbe4a9489d80f0f0159adaed25d"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x9fa3a3331c625148912ec596a39dea6ea31f1b51cebc503c9e6754ac36ea5c1c"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xeedab1c9b5b1a9b4e98e6283f4d0dcb44c1adfa3e55de0a0a11f7a8f7d7751a7"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x7956c963400cdedb6e4b2c73ead5f23fa7c007c85b0f3094f8a8d3235a3308b2"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xfb514bb7cd93f903524198c4e59a9ce01af1e31fcbbea59ed3b716ef93f7e6f7"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x118974825cf11368f5eba82e4748e81517fa8ab5fda5f573eaf5e75cddbad2d1"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x9eebe9bf02a82922cbd3c91116c5fa0d5fa46e94d4cbbfddc94e618c4053f67c"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xbbc0fbd4f77c046fc4dae0ee1770dd2dc5c80bc8c037c31c7739ee6ef607152e"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x8bc298a6f1eb709f408c2ca53c9c77193ab11a03d3e520c008eaf8af1816b919"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x2579e0644ff831be4861f89b556e3c28be41acdd29d1a6a386b53ef78d51a6ef"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x9b5cb3c9ca48b4be90ca0dab8f1a4ab71e0510463036c46a0762b7f4d8055307"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x3640e6ed4b85c59bc078f45d6b6b8ae96d3f632eaa5147e4b60064da5a5f42ba"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xf8309a56cb3fea97e4ab473335d4fd41762cf2515ae61dd895f4729d48b9f8bc"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x108f5c4b31b78c613f4eb4725440d97809cd65ac0ce3e71c41ef37be5c5f1277"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x5d705c5a7b5d41af26254ca43c1bd024d0a5b5d733e720b85d60cbd52a6e4f63"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xb949785bf29b4e08fb85806ed027130843c7c79d5d789d182bb164567173a6e7"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x4af1184a151bde9626030ad9796b4e923ca65c6f273624efaa2027c0046caa64"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xbaf454a07f8e7c8aeacfd0f6c47e488a352d19eef651ff6050aa28b8e4050d8e"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xe0312f64a0cd7329c3109901fa3a4cb89fdad566abbd5e58e6fa3ccc31bb5e1d"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x9712520461d960151b65d5171c2eca87cb82896234e16e7b1f0e178b62014553"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x90ad4cebd328480baa6833a76f7ef07a72348be7af5d9c79729247c1b2f25980"), big.NewInt(10000))
	stage := stateDB.IntermediateRoot(false)
	Logger.Debugf("GenesisBlock Stage1 Root:%s", stage.Hex())
	miners := make([]*types.Miner, 0)
	for i, member := range genesisInfo.Group.Members {
		miner := &types.Miner{Id: member.Id, PublicKey: member.PubKey, VrfPublicKey: genesisInfo.VrfPKs[i], Stake: 10}
		miners = append(miners, miner)
	}
	MinerManagerImpl.AddGenesesMiner(miners, stateDB)
	stage = stateDB.IntermediateRoot(false)
	Logger.Debugf("GenesisBlock Stage2 Root:%s", stage.Hex())
	stateDB.SetNonce(common.BonusStorageAddress, 1)
	stateDB.SetNonce(common.HeavyDBAddress, 1)
	stateDB.SetNonce(common.LightDBAddress, 1)

	root, _ := stateDB.Commit(true)
	Logger.Debugf("GenesisBlock final Root:%s", root.Hex())
	triedb.Commit(root, false)
	block.Header.StateTree = common.BytesToHash(root.Bytes())
	block.Header.Hash = block.Header.GenHash()
	//block.Transactions = make([]*types.Transaction, 0)
	return block
}
