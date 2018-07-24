package state

import (
	"testing"
	"vm/ethdb"
	"vm/common"
	"math/big"
	"fmt"
)

func TestStateDB_AddBalance(t *testing.T) {
	// Create an empty state database
	db, _ := ethdb.NewMemDatabase()
	state, _ := New(common.Hash{}, NewDatabase(db))
	state.SetBalance(common.BytesToAddress([]byte("1")), big.NewInt(1000000))
	state.AddBalance(common.BytesToAddress([]byte("1")), big.NewInt(1))
	state.SubBalance(common.BytesToAddress([]byte("1")), big.NewInt(2))
	balance := state.GetBalance(common.BytesToAddress([]byte("1")))
	fmt.Println(balance)
}

func TestStateDB_SetState(t *testing.T) {
	// Create an empty state database
	//db, _ := ethdb.NewMemDatabase()
	db, _ := ethdb.NewLDBDatabase("/Users/Kaede/TasProject/work/test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := New(common.Hash{}, triedb)
	state.SetState(common.BytesToAddress([]byte("1")), "aa", []byte{1,2,3})
	state.SetState(common.BytesToAddress([]byte("1")), "bb", nil)
	state.SetState(common.BytesToAddress([]byte("2")), "cc", []byte{1,2})
	root, _ := state.Commit(false)
	fmt.Println(root.Hex())
	triedb.TrieDB().Commit(root, false)
}

func TestStateDB_GetState(t *testing.T) {
	db, _ := ethdb.NewLDBDatabase("/Users/Kaede/TasProject/work/test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := New(common.HexToHash("0x408e0245cbdec1ff16bb13030d053356cf3b5d75c3b1dfad9831b04a08ea239e"), triedb)
	fmt.Println(string(state.Dump()))
	sta := state.GetState(common.BytesToAddress([]byte("1")), "aa")
	fmt.Println(sta)
	sta = state.GetState(common.BytesToAddress([]byte("1")), "bb")
	fmt.Println(sta)
	sta = state.GetState(common.BytesToAddress([]byte("2")), "cc")
	fmt.Println(sta)
}

func TestStateDB_SetCode(t *testing.T) {
	db, _ := ethdb.NewLDBDatabase("/Users/Kaede/TasProject/work/test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := New(common.HexToHash("0x408e0245cbdec1ff16bb13030d053356cf3b5d75c3b1dfad9831b04a08ea239e"), triedb)
	fmt.Println(string(state.Dump()))
	state.SetCode(common.BytesToAddress([]byte("2")), []byte{1,2})
	root, _ := state.Commit(false)
	fmt.Println(root.Hex())
	triedb.TrieDB().Commit(root, false)
}

func TestStateDB_GetCode(t *testing.T) {
	db, _ := ethdb.NewLDBDatabase("/Users/Kaede/TasProject/work/test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := New(common.HexToHash("0x355a590eca935afe17bf722df599b2c41279665f7ad391900ac2b4bed7fe2403"), triedb)
	fmt.Println(string(state.Dump()))
	hash := state.GetCodeHash(common.BytesToAddress([]byte("2")))
	fmt.Println(hash)
	sta := state.GetCode(common.BytesToAddress([]byte("2")))
	fmt.Println(sta)
}

func TestStateDB_Snapshot(t *testing.T) {
	db, _ := ethdb.NewLDBDatabase("/Users/Kaede/TasProject/work/test", 0, 0)
	defer db.Close()
	triedb := NewDatabase(db)
	state, _ := New(common.HexToHash("0x355a590eca935afe17bf722df599b2c41279665f7ad391900ac2b4bed7fe2403"), triedb)
	fmt.Println(string(state.Dump()))
	snapshot := state.Snapshot()
	state.SetState(common.BytesToAddress([]byte("1")), "aa", []byte{1,1,1})
	state.RevertToSnapshot(snapshot)
	sta := state.GetState(common.BytesToAddress([]byte("1")), "aa")
	fmt.Println(sta)
}