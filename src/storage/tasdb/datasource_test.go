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
	"testing"
	"fmt"
)

func TestCreateLDB(t *testing.T) {
	// 创建ldb实例
	ldb, err := NewDatabase("testldb")
	if err != nil {
		fmt.Printf("error to create ldb : %s\n", "testldb")
		return
	}

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
	ldb, err := NewDatabase("testldb")
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
