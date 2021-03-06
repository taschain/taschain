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
	"math/big"

	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/storage/account"
	"github.com/taschain/taschain/storage/serialize"
	"github.com/taschain/taschain/storage/trie"
)

var testTxAccount = []string{"0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103", "0xcad6d60fa8f6330f293f4f57893db78cf660e80d6a41718c7ad75e76795000d4",
	"0xca789a28069db6f1639b60a8bf1084333358672f65c6d6c2e6d58b69187fe402", "0x94bdb92d329dac69d7f107995a7b666d1092c63eadeae2dd495ab2e554bb155d",
	"0xb50eea221a1eb061dea7ca20f7b7508c2d9639e3558e69f758380e32624337b5", "0xce59fd5e1c6c99d9990b08ccf685260a2b3a03889de56e91b25878a4bf2f89e9",
	"0x5d9b2132ec1d2011f488648a8dc24f9b29ca40933ca89d8d19367280dff59a03", "0x5afb7e2617f1dd729ea3557096021e2f4eaa1a9c8fe48d8132b1f6cf13338a8f",
	"0x30c049d276610da3355f6c11de8623ec6b40fd2a73bb5d647df2ae83c30244bc", "0xa2b7bc555ca535745a7a9c55f9face88fc286a8b316352afc457ffafb40a7478"}

// IsTestTransaction is used for performance testing. We will not check the nonce if a transaction is sent from
// a testing account.
func IsTestTransaction(tx *types.Transaction) bool {
	if tx == nil || tx.Source == nil {
		return false
	}

	source := tx.Source.Hex()
	for _, testAccount := range testTxAccount {
		if source == testAccount {
			return true
		}
	}
	return false
}

func calcTxTree(txs []*types.Transaction) common.Hash {
	if nil == txs || 0 == len(txs) {
		return common.EmptyHash
	}

	buf := new(bytes.Buffer)

	for _, tx := range txs {
		buf.Write(tx.Hash.Bytes())
	}
	return common.BytesToHash(common.Sha256(buf.Bytes()))
}

func calcReceiptsTree(receipts types.Receipts) common.Hash {
	if nil == receipts || 0 == len(receipts) {
		return common.EmptyHash
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

func setupGenesisStateDB(stateDB *account.AccountDB, genesisInfo *types.GenesisInfo) {
	tenThousandTasBi := big.NewInt(0).SetUint64(common.TAS2RA(10000))

	//管理员账户
	stateDB.SetBalance(common.HexToAddress("0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd"), big.NewInt(0).SetUint64(common.TAS2RA(100000000)))
	stateDB.SetBalance(common.HexToAddress("0xb055a3ffdc9eeb0c5cf0c1f14507a40bdcbff98c03286b47b673c02d2efe727e"), big.NewInt(0).SetUint64(common.TAS2RA(100000000)))

	//创世账户
	for _, mem := range genesisInfo.Group.Members {
		addr := common.BytesToAddress(mem)
		stateDB.SetBalance(addr, tenThousandTasBi)
	}

	// 交易脚本账户
	for _, acc := range testTxAccount {
		stateDB.SetBalance(common.HexToAddress(acc), tenThousandTasBi)
	}
}
