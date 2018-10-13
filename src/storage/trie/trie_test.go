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

package trie

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/davecgh/go-spew/spew"
	"common"
	"storage/tasdb"

	"strconv"
)

func init() {
	spew.Config.Indent = "    "
	spew.Config.DisableMethods = false
}

// Used for testing
func newEmpty() *Trie {
	diskdb, _ := tasdb.NewMemDatabase()
	trie, _ := NewTrie(common.Hash{}, NewDatabase(diskdb))
	return trie
}

func TestEmptyTrie(t *testing.T) {
	var trie Trie
	res := trie.Hash()
	exp := emptyRoot
	if res != common.Hash(exp) {
		t.Errorf("expected %x got %x", exp, res)
	}
}



func TestNull(t *testing.T) {
	var trie Trie
	key := make([]byte, 32)
	value := []byte("test")
	trie.Update(key, value)
	if !bytes.Equal(trie.Get(key), value) {
		t.Fatal("wrong value")
	}
}


func TestExpandAll2(t *testing.T){
	diskdb, _ := tasdb.NewMemDatabase()
	triedb := NewDatabase(diskdb)
	trie, _ := NewTrie(common.Hash{}, triedb)//98
	mp := make(map[string]*[]byte)
	for i:=0;i<100;i++{
		//updateString(trie, strconv.Itoa(i), strconv.Itoa(i))
	}
	trie.Hash2(mp,true)
	root,_:=trie.Commit(nil)
	triedb.Commit(root,false)


	trie2, _ := NewTrie(root, triedb)//99
	for i:=200;i<300;i++{
		updateString(trie2, strconv.Itoa(i), strconv.Itoa(i))
	}
	trie2.Hash2(mp,true)
	root2,_:=trie2.Commit(nil)
	triedb.Commit(root2,false)


	//--------------------------------执行缺失交易-------------------------
	trie3, _ := NewTrie(root2, triedb)//100
	for i:=300;i<320;i++{
		updateString(trie3, strconv.Itoa(i), strconv.Itoa(1000+i))
	}
	trie3.Hash2(mp,true)
	root3,_:=trie3.Commit(nil)
	triedb.Commit(root3,false)
	fmt.Printf("false after1 100 tx:%v \n",root3)

	trie5, _ := NewTrie(root3, triedb)//100
	for i:=200;i<300;i++{
		updateString(trie5, strconv.Itoa(i), strconv.Itoa(1000+i))
	}
	root5,_:=trie5.Commit(nil)
	triedb.Commit(root5,false)
	fmt.Printf("false after2 100 tx:%v \n",root5)
	//--------------------------------执行缺失交易-------------------------

	trie4, _ := NewTrie(root2, triedb)//100
	for i:=200;i<320;i++{
		updateString(trie4, strconv.Itoa(i), strconv.Itoa(1000+i))
	}
	root4,_:=trie4.Commit(nil)
	triedb.Commit(root4,false)

	fmt.Printf("true after 100 tx:%v \n",root4)

	//--------------------------------execute-------------------------

	//-----------------------------------------------------------------------------------------------------------------------------
	diskdb2, _ := tasdb.NewMemDatabase()
	triedb2 := NewDatabase(diskdb2)
	for key,value := range mp{
		diskdb2.Put(([]byte)(key),*value)
	}
	trie33, _ := NewTrie(root3, triedb2)//100
	for i:=200;i<300;i++{
		updateString(trie33, strconv.Itoa(i), strconv.Itoa(1000+i))
	}
	root33,_:=trie33.Commit(nil)
	fmt.Printf("轻节点 after 100 tx:%v \n",root33)
}


func TestExpandAll(t *testing.T){
	fmt.Println("mock full chain:")
	diskdb, _ := tasdb.NewMemDatabase()
	triedb := NewDatabase(diskdb)
	mp := make(map[string]*[]byte)
	trie, _ := NewTrie(common.Hash{}, triedb)//init

	fmt.Println("--------------------------98 exe 1-----------------------------------")
	for i:=0;i<1;i++{
		updateString(trie, strconv.Itoa(i), strconv.Itoa(i))
	}
	root,_:=trie.Commit(nil)
	triedb.Commit(root,false)

	fmt.Printf("after -----------------------------------root= %x\n",root)

	fmt.Println("-------------------------- 99 exe tx2-----------------------------------")
	trie2, _ := NewTrie(root, triedb)//99
	for i:=2;i<3;i++{
		updateString(trie2, strconv.Itoa(i), strconv.Itoa(i))
	}
	root2,_:=trie2.Commit(nil)
	triedb.Commit(root2,false)
	fmt.Printf("after-----------------------------------root= %x\n",root2)

	fmt.Println("--------------------------99 get 2-----------------------------------")
	copyTrie2, _ := NewTrieWithMap(root2, triedb,mp)//99
	si := strconv.Itoa(2)
	copyTrie2.GetValueNode([]byte(si),mp)
	fmt.Printf("after-----------------------------------root= %x\n",root2)


	//------------------copy begin----------------------
	fmt.Println("--------------------------99 get 3-----------------------------------")
	copyTrie, _ := NewTrieWithMap(root2, triedb,mp)//99
	//put trie data
	for i:=3;i<4;i++{
		si := strconv.Itoa(i)
		copyTrie.GetValueNode([]byte(si),mp)
	}
	//------------------copy end----------------------
	fmt.Println("after-----------------------------------")

	fmt.Println("--------------------------100 exe 23-----------------------------------")
	trie3, _ := NewTrie(root2, triedb)//100
	for i:=2;i<4;i++{
		updateString(trie3, strconv.Itoa(i), strconv.Itoa(i + 1000))
	}
	root3,_:=trie3.Commit(nil)
	triedb.Commit(root3,false)
	fmt.Printf("after -----------------------------------root= %x\n",root3)
//-----------------------------------------------------------------------------------------------------------------------------
	fmt.Println("mock light chain:")
	diskdb2, _ := tasdb.NewMemDatabase()
	triedb2 := NewDatabase(diskdb2)
	for key,value := range mp{
		diskdb2.Put(([]byte)(key),*value)
	}
	trie22, _ := NewTrie(root2, triedb2)//99
	fmt.Println("--------------------------light 100 exe 23-----------------------------------")
	for i:=2;i<4;i++{
		updateString(trie22, strconv.Itoa(i), strconv.Itoa(i + 1000))
	}
	root33,_:=trie22.Commit(nil)
	fmt.Printf("after -----------------------------------root= %x\n",root33)

	if root3!= root33{
		t.Errorf("wrong error:old hash = %x,new hash= %x", root3,root33)
	}
}

func TestMissingRoot(t *testing.T) {
	diskdb, _ := tasdb.NewMemDatabase()
	trie, err := NewTrie(common.HexToHash("0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33"), NewDatabase(diskdb))
	if trie != nil {
		t.Error("NewTrie returned non-nil trie for invalid root")
	}
	if _, ok := err.(*MissingNodeError); !ok {
		t.Errorf("NewTrie returned wrong error: %v", err)
	}
}

func TestMissingNodeDisk(t *testing.T)    { testMissingNode(t, false) }
func TestMissingNodeMemonly(t *testing.T) { testMissingNode(t, true) }

func testMissingNode(t *testing.T, memonly bool) {
	diskdb, _ := tasdb.NewMemDatabase()
	triedb := NewDatabase(diskdb)

	trie, _ := NewTrie(common.Hash{}, triedb)
	updateString(trie, "120000", "qwerqwerqwerqwerqwerqwerqwerqwer")
	updateString(trie, "123456", "asdfasdfasdfasdfasdfasdfasdfasdf")
	root, _ := trie.Commit(nil)
	if !memonly {
		triedb.Commit(root, true)
	}

	trie, _ = NewTrie(root, triedb)
	_, err := trie.TryGet([]byte("120000"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	trie, _ = NewTrie(root, triedb)
	_, err = trie.TryGet([]byte("120099"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	trie, _ = NewTrie(root, triedb)
	_, err = trie.TryGet([]byte("123456"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	trie, _ = NewTrie(root, triedb)
	err = trie.TryUpdate([]byte("120099"), []byte("zxcvzxcvzxcvzxcvzxcvzxcvzxcvzxcv"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	trie, _ = NewTrie(root, triedb)
	err = trie.TryDelete([]byte("123456"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	hash := common.HexToHash("0xe1d943cc8f061a0c0b98162830b970395ac9315654824bf21b73b891365262f9")
	if memonly {
		delete(triedb.nodes, hash)
	} else {
		diskdb.Delete(hash[:])
	}

	trie, _ = NewTrie(root, triedb)
	_, err = trie.TryGet([]byte("120000"))
	if _, ok := err.(*MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}
	trie, _ = NewTrie(root, triedb)
	_, err = trie.TryGet([]byte("120099"))
	if _, ok := err.(*MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}
	trie, _ = NewTrie(root, triedb)
	_, err = trie.TryGet([]byte("123456"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	trie, _ = NewTrie(root, triedb)
	err = trie.TryUpdate([]byte("120099"), []byte("zxcv"))
	if _, ok := err.(*MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}
	trie, _ = NewTrie(root, triedb)
	err = trie.TryDelete([]byte("123456"))
	if _, ok := err.(*MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}
}

func TestInsert(t *testing.T) {
	diskdb, _ := tasdb.NewMemDatabase()
	//diskdb, _ := tasdb.NewLDBDatabase("/Volumes/Work/work/test2", 0, 0)
	triedb := NewDatabase(diskdb)
	trie, err := NewTrie(common.Hash{}, triedb)
	//trie,err:= NewTrie(common.HexToHash("0xe8ec13f3fc46b18ff08bb2beb207319377f03603bc9146374ca294d82c29c195"), triedb)
	//trie := newEmpty()
	if err != nil {
		panic(err)
	}
	updateString(trie, "doe", "reindeer")
	updateString(trie, "xog", "puppy")
	updateString(trie, "xogglesworth", "cat")
	updateString(trie, "togee", "cat11")
	updateString(trie, "pogef", "cat12")
	updateString(trie, "qogefff", "cat123")
	//trie.Update([]byte("dogeffa"), bytes.Repeat([]byte{'a'}, 33))
	//fmt.Println(string(trie.Get([]byte("doe"))))
	//exp := common.HexToHash("8aad789dff2f538bca5d8ea56e8abe10f4c7ba3a5dea95fea4cd6e7c3a1168d3")
	root := trie.Hash()
	//if root != exp {
	//	t.Errorf("exp %x got %x", exp, root)
	//}

	//trie = newEmpty()
	//updateString(trie, "A", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	//exp = common.HexToHash("d23786fb4a010da3ce639d66d5e904a11dbc02746d1ce25029e53290cabf28ab")
	root, _ = trie.Commit(nil)
	triedb.Commit(root, false)
	fmt.Println(root.Hex())
	//diskdb.Close()
	//diskdb, _ = tasdb.NewLDBDatabase("/Volumes/Work/work/test2", 0, 0)

	//if err != nil {
	//	t.Fatalf("commit error: %v", err)
	//}
	//if root != exp {
	//	t.Errorf("exp %x got %x", exp, root)
	//}

	trie2, _ := NewTrie(root, NewDatabase(diskdb))
	root,_ = trie2.Commit(nil)
	fmt.Println(root.Hex())
	fmt.Println(string(getString(trie2,"qogefff")))
	//diskdb.Close()
	//diskdb, _ = tasdb.NewLDBDatabase("/Volumes/Work/work/test2", 0, 0)
	//
	//trie3, _ := NewTrie(root, NewDatabase(diskdb))
	//updateString(trie3, "adsfdasf", "cat123")
	//root,_ = trie3.Commit(nil)
	//fmt.Println(root.Hex())
	//fmt.Println(string(trie2.Get([]byte("dogeffa"))))
}

func TestGet(t *testing.T) {
	trie := newEmpty()
	updateString(trie, "doe", "reindeer")
	updateString(trie, "dog", "puppy")
	updateString(trie, "dogglesworth", "cat")

	for i := 0; i < 2; i++ {
		res := getString(trie, "dog")
		if !bytes.Equal(res, []byte("puppy")) {
			t.Errorf("expected puppy got %x", res)
		}

		unknown := getString(trie, "unknown")
		if unknown != nil {
			t.Errorf("expected nil got %x", unknown)
		}

		if i == 1 {
			return
		}
		trie.Commit(nil)
	}
}

func TestDelete(t *testing.T) {
	trie := newEmpty()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
	}
	for _, val := range vals {
		if val.v != "" {
			updateString(trie, val.k, val.v)
		} else {
			deleteString(trie, val.k)
		}
	}

	hash := trie.Hash()
	exp := common.HexToHash("91f85b454903f1475fe2fc3067ae2d7ae5f1effc067907c17deba8029d307b8d")
	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestEmptyValues(t *testing.T) {
	trie := newEmpty()

	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
	}
	for _, val := range vals {
		updateString(trie, val.k, val.v)
	}

	hash := trie.Hash()
	exp := common.HexToHash("91f85b454903f1475fe2fc3067ae2d7ae5f1effc067907c17deba8029d307b8d")
	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestReplication(t *testing.T) {
	trie := newEmpty()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"dog", "puppy"},
		{"somethingveryoddindeedthis is", "myothernodedata"},
	}
	for _, val := range vals {
		updateString(trie, val.k, val.v)
	}
	exp, err := trie.Commit(nil)
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}

	// create a new trie on top of the database and check that lookups work.
	trie2, err := NewTrie(exp, trie.db)
	if err != nil {
		t.Fatalf("can't recreate trie at %x: %v", exp, err)
	}
	for _, kv := range vals {
		if string(getString(trie2, kv.k)) != kv.v {
			t.Errorf("trie2 doesn't have %q => %q", kv.k, kv.v)
		}
	}
	hash, err := trie2.Commit(nil)
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}
	if hash != exp {
		t.Errorf("root failure. expected %x got %x", exp, hash)
	}

	// perform some insertions on the new trie.
	vals2 := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		// {"shaman", "horse"},
		// {"doge", "coin"},
		// {"ether", ""},
		// {"dog", "puppy"},
		// {"somethingveryoddindeedthis is", "myothernodedata"},
		// {"shaman", ""},
	}
	for _, val := range vals2 {
		updateString(trie2, val.k, val.v)
	}
	if hash := trie2.Hash(); hash != exp {
		t.Errorf("root failure. expected %x got %x", exp, hash)
	}
}

func TestLargeValue(t *testing.T) {
	trie := newEmpty()
	trie.Update([]byte("key1"), []byte{99, 99, 99, 99})
	trie.Update([]byte("key2"), bytes.Repeat([]byte{1}, 32))
	trie.Hash()
}

type countingDB struct {
	tasdb.Database
	gets map[string]int
}

func (db *countingDB) Get(key []byte) ([]byte, error) {
	db.gets[string(key)]++
	return db.Database.Get(key)
}

// TestCacheUnload checks that decoded nodes are unloaded after a
// certain number of commit operations.
func TestCacheUnload(t *testing.T) {
	// Create test trie with two branches.
	trie := newEmpty()
	key1 := "---------------------------------"
	key2 := "---some other branch"
	updateString(trie, key1, "this is the branch of key1.")
	updateString(trie, key2, "this is the branch of key2.")

	root, _ := trie.Commit(nil)
	trie.db.Commit(root, true)

	// Commit the trie repeatedly and access key1.
	// The branch containing it is loaded from DB exactly two times:
	// in the 0th and 6th iteration.
	db := &countingDB{Database: trie.db.diskdb, gets: make(map[string]int)}
	trie, _ = NewTrie(root, NewDatabase(db))
	trie.SetCacheLimit(5)
	for i := 0; i < 12; i++ {
		getString(trie, key1)
		trie.Commit(nil)
	}
	// Check that it got loaded two times.
	for dbkey, count := range db.gets {
		if count != 2 {
			t.Errorf("db key %x loaded %d times, want %d times", []byte(dbkey), count, 2)
		}
	}
}

// randTest performs random trie operations.
// Instances of this test are created by Generate.
type randTest []randTestStep

type randTestStep struct {
	op    int
	key   []byte // for opUpdate, opDelete, opGet
	value []byte // for opUpdate
	err   error  // for debugging
}

const (
	opUpdate = iota
	opDelete
	opGet
	opCommit
	opHash
	opReset
	opItercheckhash
	opCheckCacheInvariant
	opMax // boundary value, not an actual op
)

func (randTest) Generate(r *rand.Rand, size int) reflect.Value {
	var allKeys [][]byte
	genKey := func() []byte {
		if len(allKeys) < 2 || r.Intn(100) < 10 {
			// new key
			key := make([]byte, r.Intn(50))
			r.Read(key)
			allKeys = append(allKeys, key)
			return key
		}
		// use existing key
		return allKeys[r.Intn(len(allKeys))]
	}

	var steps randTest
	for i := 0; i < size; i++ {
		step := randTestStep{op: r.Intn(opMax)}
		switch step.op {
		case opUpdate:
			step.key = genKey()
			step.value = make([]byte, 8)
			binary.BigEndian.PutUint64(step.value, uint64(i))
		case opGet, opDelete:
			step.key = genKey()
		}
		steps = append(steps, step)
	}
	return reflect.ValueOf(steps)
}

func runRandTest(rt randTest) bool {
	diskdb, _ := tasdb.NewMemDatabase()
	triedb := NewDatabase(diskdb)

	tr, _ := NewTrie(common.Hash{}, triedb)
	values := make(map[string]string) // tracks content of the trie

	for i, step := range rt {
		switch step.op {
		case opUpdate:
			tr.Update(step.key, step.value)
			values[string(step.key)] = string(step.value)
		case opDelete:
			tr.Delete(step.key)
			delete(values, string(step.key))
		case opGet:
			v := tr.Get(step.key)
			want := values[string(step.key)]
			if string(v) != want {
				rt[i].err = fmt.Errorf("mismatch for key 0x%x, got 0x%x want 0x%x", step.key, v, want)
			}
		case opCommit:
			_, rt[i].err = tr.Commit(nil)
		case opHash:
			tr.Hash()
		case opReset:
			hash, err := tr.Commit(nil)
			if err != nil {
				rt[i].err = err
				return false
			}
			newtr, err := NewTrie(hash, triedb)
			if err != nil {
				rt[i].err = err
				return false
			}
			tr = newtr
		//case opItercheckhash:
		//	checktr, _ := NewTrie(common.Hash{}, triedb)
		//	it := NewIterator(tr.NodeIterator(nil))
		//	for it.Next() {
		//		checktr.Update(it.Key, it.Value)
		//	}
		//	if tr.Hash() != checktr.Hash() {
		//		rt[i].err = fmt.Errorf("hash mismatch in opItercheckhash")
		//	}
		case opCheckCacheInvariant:
			rt[i].err = checkCacheInvariant(tr.RootNode, nil, tr.cachegen, false, 0)
		}
		// Abort the test on error.
		if rt[i].err != nil {
			return false
		}
	}
	return true
}

func checkCacheInvariant(n, parent node, parentCachegen uint16, parentDirty bool, depth int) error {
	var children []node
	var flag nodeFlag
	switch n := n.(type) {
	case *shortNode:
		flag = n.flags
		children = []node{n.Val}
	case *fullNode:
		flag = n.flags
		children = n.Children[:]
	default:
		return nil
	}

	errorf := func(format string, args ...interface{}) error {
		msg := fmt.Sprintf(format, args...)
		msg += fmt.Sprintf("\nat depth %d node %s", depth, spew.Sdump(n))
		msg += fmt.Sprintf("parent: %s", spew.Sdump(parent))
		return errors.New(msg)
	}
	if flag.gen > parentCachegen {
		return errorf("cache invariant violation: %d > %d\n", flag.gen, parentCachegen)
	}
	if depth > 0 && !parentDirty && flag.dirty {
		return errorf("cache invariant violation: %d > %d\n", flag.gen, parentCachegen)
	}
	for _, child := range children {
		if err := checkCacheInvariant(child, n, flag.gen, flag.dirty, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func TestRandom(t *testing.T) {
	if err := quick.Check(runRandTest, nil); err != nil {
		if cerr, ok := err.(*quick.CheckError); ok {
			t.Fatalf("random test iteration %d failed: %s", cerr.Count, spew.Sdump(cerr.In))
		}
		t.Fatal(err)
	}
}

func BenchmarkGet(b *testing.B)      { benchGet(b, false) }
func BenchmarkGetDB(b *testing.B)    { benchGet(b, true) }
func BenchmarkUpdateBE(b *testing.B) { benchUpdate(b, binary.BigEndian) }
func BenchmarkUpdateLE(b *testing.B) { benchUpdate(b, binary.LittleEndian) }

const benchElemCount = 20000

func benchGet(b *testing.B, commit bool) {
	trie := new(Trie)
	if commit {
		_, tmpdb := tempDB()
		trie, _ = NewTrie(common.Hash{}, tmpdb)
	}
	k := make([]byte, 32)
	for i := 0; i < benchElemCount; i++ {
		binary.LittleEndian.PutUint64(k, uint64(i))
		trie.Update(k, k)
	}
	binary.LittleEndian.PutUint64(k, benchElemCount/2)
	if commit {
		trie.Commit(nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Get(k)
	}
	b.StopTimer()

	if commit {
		ldb := trie.db.diskdb.(*tasdb.LDBDatabase)
		ldb.Close()
		os.RemoveAll(ldb.Path())
	}
}

func benchUpdate(b *testing.B, e binary.ByteOrder) *Trie {
	trie := newEmpty()
	k := make([]byte, 32)
	for i := 0; i < b.N; i++ {
		e.PutUint64(k, uint64(i))
		trie.Update(k, k)
	}
	return trie
}

// Benchmarks the trie hashing. Since the trie caches the result of any operation,
// we cannot use b.N as the number of hashing rouns, since all rounds apart from
// the first one will be NOOP. As such, we'll use b.N as the number of account to
// insert into the trie before measuring the hashing.
func BenchmarkHash(b *testing.B) {
	//// Make the random benchmark deterministic
	//random := rand.New(rand.NewSource(0))
	//
	//// Create a realistic account trie to hash
	//addresses := make([][20]byte, b.N)
	//for i := 0; i < len(addresses); i++ {
	//	for j := 0; j < len(addresses[i]); j++ {
	//		addresses[i][j] = byte(random.Intn(256))
	//	}
	//}
	//accounts := make([][]byte, len(addresses))
	//for i := 0; i < len(accounts); i++ {
	//	var (
	//		nonce   = uint64(random.Int63())
	//		balance = new(big.Int).Rand(random, new(big.Int).Exp(common.Big2, common.Big256, nil))
	//		root    = emptyRoot
	//		code    = sha3.Sum256(nil)
	//	)
	//	accounts[i], _ = serialize.EncodeToBytes([]interface{}{nonce, balance, root, code})
	//}
	//// Insert the accounts into the trie and hash it
	//trie := newEmpty()
	//for i := 0; i < len(addresses); i++ {
	//	trie.Update(crypto.Keccak256(addresses[i][:]), accounts[i])
	//}
	//b.ResetTimer()
	//b.ReportAllocs()
	//trie.Hash()
}

func tempDB() (string, *Database) {
	dir, err := ioutil.TempDir("", "trie-bench")
	if err != nil {
		panic(fmt.Sprintf("can't create temporary directory: %v", err))
	}
	diskdb, err := tasdb.NewLDBDatabase(dir, 256, 0)
	if err != nil {
		panic(fmt.Sprintf("can't create temporary database: %v", err))
	}
	return dir, NewDatabase(diskdb)
}

func getString(trie *Trie, k string) []byte {
	return trie.Get([]byte(k))
}

func updateString(trie *Trie, k, v string) {
	trie.Update([]byte(k), []byte(v))
}

func deleteString(trie *Trie, k string) {
	trie.Delete([]byte(k))
}
