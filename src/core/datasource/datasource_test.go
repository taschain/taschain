package datasource

import (
	"testing"
	"fmt"
)

func TestCreateLDB(t *testing.T) {
	// 创建ldb实例
	ldb, err := NewLDBDatabase("testldb", 20, 20)
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
