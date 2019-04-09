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
	"fmt"
	"math/big"
	"middleware/types"
	"storage/account"
	"storage/serialize"
	"storage/trie"
	"tvm"
)

var testTxAccount = []string{"0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103", "0xcad6d60fa8f6330f293f4f57893db78cf660e80d6a41718c7ad75e76795000d4",
	"0xca789a28069db6f1639b60a8bf1084333358672f65c6d6c2e6d58b69187fe402", "0x94bdb92d329dac69d7f107995a7b666d1092c63eadeae2dd495ab2e554bb155d",
	"0xb50eea221a1eb061dea7ca20f7b7508c2d9639e3558e69f758380e32624337b5", "0xce59fd5e1c6c99d9990b08ccf685260a2b3a03889de56e91b25878a4bf2f89e9",
	"0x5d9b2132ec1d2011f488648a8dc24f9b29ca40933ca89d8d19367280dff59a03", "0x5afb7e2617f1dd729ea3557096021e2f4eaa1a9c8fe48d8132b1f6cf13338a8f",
	"0x30c049d276610da3355f6c11de8623ec6b40fd2a73bb5d647df2ae83c30244bc", "0xa2b7bc555ca535745a7a9c55f9face88fc286a8b316352afc457ffafb40a7478"}

func IsTestTransaction(tx *types.Transaction) bool {
	if tx == nil || tx.Source == nil {
		return false
	}

	source := tx.Source.GetHexString()
	for _, testAccount := range testTxAccount {
		if source == testAccount {
			return true
		}
	}
	return false
}

var emptyHash = common.Hash{}

func calcTxTree(txs []*types.Transaction) common.Hash {
	if nil == txs || 0 == len(txs) {
		return emptyHash
	}

	buf := new(bytes.Buffer)
	//for i := 0; i < len(tx); i++ {
	//	encode, _ := msgpack.Marshal(tx[i])
	//	serialize.Encode(buf, encode)
	//	buf.Write(tx)
	//}
	for _, tx := range txs {
		buf.Write(tx.Hash.Bytes())
	}
	return common.BytesToHash(common.Sha256(buf.Bytes()))
}

func calcReceiptsTree(receipts types.Receipts) common.Hash {
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

func setupGenesisStateDB(stateDB *account.AccountDB, genesisInfo *types.GenesisInfo) {
	tenThousandTasBi := big.NewInt(0).SetUint64(common.TAS2RA(10000))

	//管理员账户
	stateDB.SetBalance(common.HexStringToAddress("0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd"), big.NewInt(0).SetUint64(common.TAS2RA(5000000)))
	stateDB.SetBalance(common.HexStringToAddress("0xb055a3ffdc9eeb0c5cf0c1f14507a40bdcbff98c03286b47b673c02d2efe727e"), big.NewInt(0).SetUint64(common.TAS2RA(5000000)))

	//创世账户
	for _, mem := range genesisInfo.Group.Members {
		addr := common.BytesToAddress(mem)
		stateDB.SetBalance(addr, tenThousandTasBi)
	}

	// 交易脚本账户
	for _, acc := range testTxAccount {
		stateDB.SetBalance(common.HexStringToAddress(acc), tenThousandTasBi)
	}

	setupTnsContract(stateDB)
}

func setupTnsContract(stateDB *account.AccountDB) {

	tnsManager :=common.HexStringToAddress("0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd")
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(tnsManager[:], common.Uint64ToByte(stateDB.GetNonce(tnsManager)))))
	code := tvm.Read0("./py/tns.py")
	contractData := tvm.Contract{code, "Tns", &contractAddr}
	jsonString, _ := json.Marshal(contractData)
	if len(jsonString) <= 0 {
		return
	}

	Logger.Debugf("tns contract address: %v", contractAddr)

	stateDB.CreateAccount(contractAddr)
	stateDB.SetCode(contractAddr, jsonString)
	stateDB.SetNonce(contractAddr, 1)

	controller := tvm.NewController(stateDB, nil, nil, nil, common.GlobalConf.GetString("tvm", "pylib", "lib"))
	controller.GasLeft = 1000000

	msg := tvm.Msg{Data: nil, Value: 0, Sender: tnsManager.GetHexString()}

	//部署tns合约
	errorCode, errorMsg := controller.DeployWithMsg(&tnsManager, &contractData, msg)
	if errorCode != 0 {
		Logger.Errorf("tns contract deploy error: %v", errorMsg)
		return
	}

	//设置地址
	abi := fmt.Sprintf(`{"FuncName": "set_short_account_address", "Args": ["tns", "%v"]}`, contractAddr)
	success, errorMsg := controller.ExecuteAbi(&tnsManager, &contractData, abi, msg)
	if !success  {
		Logger.Errorf("tns contract set_short_account_address error: %v", errorMsg)
		return
	}

	//获取account对应的地址
	abi = fmt.Sprintf(`{"FuncName": "get_address", "Args": ["tns"]}`)
	result := controller.ExecuteAbiResult(&tnsManager, &contractData, abi, msg)
	//if !success  {
		Logger.Debugf("tns contract get_address: %v", result.Content)
		//return
	//}
}

