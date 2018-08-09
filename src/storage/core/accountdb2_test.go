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
	"testing"
	"storage/tasdb"
	"math/big"
	"fmt"
	"common"
)

func TestAccountDB_AddBalance(t *testing.T) {
	// Create an empty state database
	db, _ := tasdb.NewMemDatabase()
	state, _ := NewAccountDB(common.Hash{}, NewDatabase(db))
	state.SetBalance(common.BytesToAddress([]byte("1")), big.NewInt(1000000))
	state.AddBalance(common.BytesToAddress([]byte("1")), big.NewInt(1))
	state.SubBalance(common.BytesToAddress([]byte("1")), big.NewInt(2))
	balance := state.GetBalance(common.BytesToAddress([]byte("1")))
	fmt.Println(balance)
}

func TestAccountDB_SetData(t *testing.T) {
	// Create an empty state database
	//db, _ := tasdb.NewMemDatabase()
	db, _ := tasdb.NewLDBDatabase("/Users/Kaede/TasProject/work/test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := NewAccountDB(common.Hash{}, triedb)
	state.SetData(common.BytesToAddress([]byte("1")), "aa", []byte{1,2,3})

	state.SetData(common.BytesToAddress([]byte("1")), "bb", []byte{1})
	snapshot := state.Snapshot()
	state.SetData(common.BytesToAddress([]byte("1")), "bb", []byte{2})
	state.RevertToSnapshot(snapshot)
	state.SetData(common.BytesToAddress([]byte("2")), "cc", []byte{1,2})
	root, _ := state.Commit(false)
	fmt.Println(root.Hex())
	triedb.TrieDB().Commit(root, false)
}

func TestAccountDB_GetData(t *testing.T) {
	db, _ := tasdb.NewLDBDatabase("/Users/Kaede/TasProject/work/test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := NewAccountDB(common.HexToHash("0x8df8e749765d8bca4db3e957c66369bb058e64108a25c69f3513430ceba79eff"), triedb)
	//fmt.Println(string(state.Dump()))
	sta := state.GetData(common.BytesToAddress([]byte("1")), "aa")
	fmt.Println(sta)
	sta = state.GetData(common.BytesToAddress([]byte("1")), "bb")
	fmt.Println(sta)
	sta = state.GetData(common.BytesToAddress([]byte("2")), "cc")
	fmt.Println(sta)
}

func TestAccountDB_SetCode(t *testing.T) {
	db, _ := tasdb.NewLDBDatabase("/Users/Kaede/TasProject/work/test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := NewAccountDB(common.HexToHash("0xbd51564993c69858d5d2e181fc8d5e5dfdb4e1800f0ead7a1ebdd4a488f5e55f"), triedb)
	//fmt.Println(string(state.Dump()))
	state.SetCode(common.BytesToAddress([]byte("2")), []byte{1,2})
	root, _ := state.Commit(false)
	fmt.Println(root.Hex())
	triedb.TrieDB().Commit(root, false)
}

func TestAccountDB_GetCode(t *testing.T) {
	db, _ := tasdb.NewLDBDatabase("/Users/Kaede/TasProject/work/test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := NewAccountDB(common.HexToHash("0x43b1c4652bd927fce344607f46d61955100dc5b4f358baf2df4f4bfdf2016683"), triedb)
	//fmt.Println(string(state.Dump()))
	hash := state.GetCodeHash(common.BytesToAddress([]byte("2")))
	fmt.Println(hash)
	sta := state.GetCode(common.BytesToAddress([]byte("2")))
	fmt.Println(sta)
}