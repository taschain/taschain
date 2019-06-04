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
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
	time2 "github.com/taschain/taschain/middleware/time"
	"github.com/taschain/taschain/middleware/types"
	"github.com/vmihailenco/msgpack"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

/*
**  Creator: pxf
**  Date: 2019/3/20 下午8:56
**  Description:
 */
func genTx(source string, target string) *types.Transaction {
	var sourceAddr, targetAddr *common.Address

	sourcebyte := common.HexToAddress(source)
	sourceAddr = &sourcebyte
	if target == "" {
		targetAddr = nil
	} else {
		targetbyte := common.HexToAddress(target)
		targetAddr = &targetbyte
	}

	tx := &types.Transaction{
		Data:          []byte{13, 23},
		GasPrice:      1,
		Source:        sourceAddr,
		Target:        targetAddr,
		Nonce:         rand.Uint64(),
		Value:         rand.Uint64(),
		ExtraData:     []byte{2, 3, 4},
		ExtraDataType: 1,
		GasLimit:      10000000,
		Type:          1,
	}
	tx.Hash = tx.GenHash()
	return tx
}

func genBlockHeader() *types.BlockHeader {
	castor := groupsig.ID{}
	castor.SetBigInt(big.NewInt(1000))
	bh := &types.BlockHeader{
		CurTime:    time2.TimeToTimeStamp(time.Now()),
		Height:     rand.Uint64(),
		ProveValue: []byte{},
		Castor:     castor.Serialize(),
		GroupId:    castor.Serialize(),
		TotalQN:    rand.Uint64(),
		StateTree:  common.Hash{},
	}
	return bh
}

func genBlock(txNum int) *types.Block {
	bh := genBlockHeader()
	txs := make([]*types.Transaction, 0)
	txHashs := make([]common.Hash, 0)
	for i := 0; i < txNum; i++ {
		tx := genTx("0x123", "0x234")
		txs = append(txs, tx)
		txHashs = append(txHashs, tx.Hash)
	}
	return &types.Block{
		Header:       bh,
		Transactions: txs,
	}
}

func TestEncodeTransaction(t *testing.T) {
	b := genBlock(10)
	bs, err := encodeBlockTransactions(b)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(bs)
}
func TestDecodeBlockTransactionWithNoTransaction(t *testing.T) {
	b := genBlock(0)
	t.Logf("block is %+v", b.Header)
	bs, err := encodeBlockTransactions(b)
	if err != nil {
		t.Fatal(err)
	}

	txs, err := decodeBlockTransactions(bs)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("txs %v", txs)

}

func TestDecodeBlockTransactionWithTransactions(t *testing.T) {
	b := genBlock(10)
	t.Logf("block is %+v", b.Header)
	for i, tx := range b.Transactions {
		t.Logf("before %v: %+v", i, tx)
	}
	bs, err := encodeBlockTransactions(b)
	if err != nil {
		t.Fatal(err)
	}

	txs, err := decodeBlockTransactions(bs)
	if err != nil {
		t.Fatal(err)
	}
	for i, tx := range txs {
		if tx.Hash != b.Transactions[i].Hash {
			t.Fatal("tx hash error")
		}
	}
}

func TestDecodeTransactionByHash(t *testing.T) {
	b := genBlock(11)
	t.Logf("block is %+v", b.Header)
	var testHash common.Hash
	var testIndex int
	r := rand.Intn(len(b.Transactions))

	for i, tx := range b.Transactions {
		if i == r {
			testHash = tx.Hash
			testIndex = i
			t.Log("test hash", i, testHash.Hex())
		}
	}
	bs, err := encodeBlockTransactions(b)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := decodeTransaction(testIndex, testHash, bs)
	if err != nil {
		t.Fatal(err)
	}
	if tx.Hash != tx.GenHash() {
		t.Fatal("gen hash diff")
	}
	if tx.Hash != testHash {
		t.Fatal("hash diff")
	}
	t.Log("success")
}

func TestMarshalSign(t *testing.T) {
	s := common.HexToSign("0x220ee8a9b1f85445ef27e1ae82f985087fe40854ccc3f8a6c6a5d47116420dc6000000000000000000000000000000000000000000000000000000000000000000")
	bs, err := msgpack.Marshal(s)
	t.Log(bs, err)

	var sign *common.Sign
	err = msgpack.Unmarshal(bs, &sign)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(sign.Hex())
}

func TestMarshalTx(t *testing.T) {
	tx := genTx("0x123", "0x2343")
	bs, err := marshalTx(tx)
	t.Log(tx, tx.Source)
	if err != nil {
		t.Fatal(err)
	}

	tx1, err := unmarshalTx(bs)
	t.Log(tx1, tx1.Source)
}
