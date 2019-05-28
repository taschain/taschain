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
package tasdb

import (
	"fmt"
	"github.com/taschain/taschain/utility"
	"testing"
)

func TestCreateLDB(t *testing.T) {
	// 创建ldb实例
	ds, err := NewDataSource("test")
	if err != nil {
		t.Fatal(err)
	}
	ldb, err := ds.NewPrefixDatabase("testldb")
	if err != nil {
		fmt.Printf("error to create ldb : %s\n", "testldb")
		return
	}
	defer ldb.Close()

	// 测试put
	err = ldb.Put([]byte("testkey"), []byte("testvalue"))
	if err != nil {
		fmt.Printf("failed to put key in testldb\n")
	}

	// 测试get
	result, err := ldb.Get([]byte("testkey"))
	if err != nil {
		fmt.Printf("failed to get key in testldb\n")
	}
	if result != nil {
		fmt.Printf("get key : testkey, value: %s \n", result)
	}

	// 测试has
	exist, err := ldb.Has([]byte("testkey"))
	if err != nil {
		fmt.Printf("error to check key : %s\n", "testkey")

	}
	if exist {
		fmt.Printf("get key : %s\n", "testkey")
	}

	// 测试delete
	err = ldb.Delete([]byte("testkey"))
	if err != nil {
		fmt.Printf("error to delete key : %s\n", "testkey")

	}

	// 测试get空
	// key不存在，会返回err
	result, err = ldb.Get([]byte("testkey"))
	if err != nil {
		fmt.Printf("failed to get key in testldb\n")
	}
	if result != nil {
		fmt.Printf("get key : testkey, value: %s \n", result)
	} else {
		fmt.Printf("get key : testkey, value: null")
	}

	if ldb != nil {
		ldb.Close()
	}

}

func TestLDBScan(t *testing.T) {
	//ldb, _ := NewLDBDatabase("/Users/Kaede/TasProject/test1",1,1)
	ds, err := NewDataSource("test")
	if err != nil {
		t.Fatal(err)
	}
	ldb, err := ds.NewPrefixDatabase("testldb")
	defer ldb.Close()
	key1 := []byte{0, 1, 1}
	key2 := []byte{0, 1, 2}
	key3 := []byte{0, 2, 1}
	ldb.Put(key1, key1)
	ldb.Put(key2, key2)
	ldb.Put(key3, key3)
	iter := ldb.NewIteratorWithPrefix([]byte{0, 1})
	for iter.Next() {
		fmt.Println(iter.Value())
	}
}

func TestLRUMemDatabase(t *testing.T) {
	mem, _ := NewLRUMemDatabase(10)
	for i := (byte)(0); i < 11; i++ {
		mem.Put([]byte{i}, []byte{i})
	}
	data, _ := mem.Get([]byte{0})
	if data != nil {
		t.Errorf("expected value nil")
	}
	data, _ = mem.Get([]byte{10})
	if data == nil {
		t.Errorf("expected value not nil")
	}
	data, _ = mem.Get([]byte{5})
	if data == nil {
		t.Errorf("expected value not nil")
	}
	mem.Delete([]byte{5})
	data, _ = mem.Get([]byte{5})
	if data != nil {
		t.Errorf("expected value nil")
	}
}

func TestClearLDB(t *testing.T) {
	// 创建ldb实例
	ds, err := NewDataSource("test")
	if err != nil {
		t.Fatal(err)
	}
	ldb, err := ds.NewPrefixDatabase("testldb")
	defer ldb.Close()
	if err != nil {
		t.Fatalf("error to create ldb : %s\n", "testldb")
		return
	}

	// 测试put
	err = ldb.Put([]byte("testkey"), []byte("testvalue"))
	if err != nil {
		t.Fatalf("failed to put key in testldb\n")
	}

	if err != nil {
		t.Fatalf("error to clear ldb : %s\n", "testldb")
		return
	}

	// 测试get，期待为null
	//result, err := ldb.Get([]byte("testkey"))
	//if result != nil {
	//	t.Fatalf("get key : testkey, value: %s \n", result)
	//
	//} else {
	//	fmt.Printf("get key : testkey, value: null")
	//}
}

func TestBatchPutVisiableBeforeWrite(t *testing.T) {
	ds, err := NewDataSource("test")
	if err != nil {
		t.Fatal(err)
	}
	ldb, err := ds.NewPrefixDatabase("testldb")
	defer ldb.Close()
	if err != nil {
		t.Fatalf("error to create ldb : %s\n", "testldb")
		return
	}

	key := []byte("test")
	batch := ldb.CreateLDBBatch()
	bs, err := ldb.Get(key)
	t.Log("before put:", string(bs), err)

	ldb.AddKv(batch, key, []byte("i am handsome"))
	bs, err = ldb.Get(key)
	t.Log("after put:", string(bs), err)

	err = batch.Write()
	if err != nil {
		t.Fatal("write fail", err)
	}
	bs, err = ldb.Get(key)
	t.Log("after write:", string(bs), err)

	ldb.AddKv(batch, key, nil)
	err = batch.Write()
	if err != nil {
		t.Fatal("write fail", err)
	}
	bs, err = ldb.Get(key)
	t.Log("after delete:", string(bs), err)
}

func TestIteratorWithPrefix(t *testing.T) {
	ds, err := NewDataSource("test")
	if err != nil {
		t.Fatal(err)
	}
	ldb, err := ds.NewPrefixDatabase("testldb")
	defer ldb.Close()
	if err != nil {
		t.Fatalf("error to create ldb : %s\n", "testldb")
		return
	}

	for i := 1; i < 13; i++ {
		ldb.Put(utility.UInt64ToByte(uint64(i)), utility.UInt64ToByte(uint64(i)))
	}

	for i := 20; i < 23; i++ {
		ldb.Put(utility.UInt64ToByte(uint64(i)), utility.UInt64ToByte(uint64(i)))
	}
	ldb.Put(utility.UInt64ToByte(uint64(1000)), utility.UInt64ToByte(uint64(1000)))
	ldb.Put(utility.UInt64ToByte(uint64(99999)), utility.UInt64ToByte(uint64(99999)))

	iter := ldb.NewIterator()
	//for iter.Next() {
	//	t.Log("next",utility.ByteToUInt64(iter.Key()), utility.ByteToUInt64(iter.Value()))
	//}
	if iter.Seek(utility.UInt64ToByte(uint64(20))) {
		t.Log("seek", utility.ByteToUInt64(iter.Key()), utility.ByteToUInt64(iter.Value()))
	}
	for iter.Next() {
		t.Log("iter", utility.ByteToUInt64(iter.Key()), utility.ByteToUInt64(iter.Value()))
	}

	iter.Release()
}

func TestIteratorWithPrefix2(t *testing.T) {
	ds, err := NewDataSource("test")
	if err != nil {
		t.Fatal(err)
	}
	ldb, err := ds.NewPrefixDatabase("testldb")
	defer ldb.Close()
	if err != nil {
		t.Fatalf("error to create ldb : %s\n", "testldb")
		return
	}

	ldb2, err := ds.NewPrefixDatabase("testldb")
	if err != nil {
		t.Fatalf("error to create ldb : %s\n", "testldb")
		return
	}

	for i := 1; i < 13; i++ {
		ldb2.Put(utility.UInt64ToByte(uint64(i)), utility.UInt64ToByte(uint64(i)))
	}

	for i := 1; i < 13; i++ {
		ldb.Put(utility.UInt64ToByte(uint64(i)), utility.UInt64ToByte(uint64(i)))
	}

	for i := 20; i < 23; i++ {
		ldb.Put(utility.UInt64ToByte(uint64(i)), utility.UInt64ToByte(uint64(i)))
	}
	ldb.Put(utility.UInt64ToByte(uint64(1000)), utility.UInt64ToByte(uint64(1000)))
	ldb.Put(utility.UInt64ToByte(uint64(99999)), utility.UInt64ToByte(uint64(99999)))

	iter := ldb2.NewIterator()
	//for iter.Next() {
	//	t.Log("ldb",utility.ByteToUInt64(iter.Key()), utility.ByteToUInt64(iter.Value()))
	//}
	//
	//iter2 := ldb2.NewIterator()
	//for iter2.Next() {
	//	t.Log("ldb2",utility.ByteToUInt64(iter2.Key()), utility.ByteToUInt64(iter2.Value()))
	//}

	if iter.Seek(utility.UInt64ToByte(uint64(12))) {
		t.Log("seek", utility.ByteToUInt64(iter.Key()), utility.ByteToUInt64(iter.Value()))
	}

	for iter.Next() {
		t.Log("iter", utility.ByteToUInt64(iter.Key()), utility.ByteToUInt64(iter.Value()))
	}

	iter.Release()
}

func TestGetAfter(t *testing.T) {
	ds, err := NewDataSource("test")
	if err != nil {
		t.Fatal(err)
	}
	ldb, err := ds.NewPrefixDatabase("testldb")
	defer ldb.Close()
	if err != nil {
		t.Fatalf("error to create ldb : %s\n", "testldb")
		return
	}
	//for i := 1; i < 13; i++ {
	//	ldb.Put(utility.UInt64ToByte(uint64(i)), utility.UInt64ToByte(uint64(i)))
	//}

	iter := ldb.NewIterator()
	if !iter.Seek(utility.UInt64ToByte(11)) {
		t.Logf("not found")
	}

	cnt := 10
	for cnt > 0 {
		t.Log(utility.ByteToUInt64(iter.Key()), utility.ByteToUInt64(iter.Value()))
		cnt--
		if !iter.Next() {
			break
		}
	}
}
