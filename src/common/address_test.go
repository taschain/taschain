package common

import (
	"math/rand"
	"testing"
	"time"
)

func TestSortAddresses(test *testing.T) {
	rand.Seed(time.Now().UnixNano()) //以当前时间值作为随机数种子
	n := rand.Intn(10)               //生成10个随机数？
	addresses, err := RandomAddresses(n)
	if err != nil {
		test.Fatal(err)
	}
	SortAddresses(addresses)
	for i := 0; i < n-1; i++ {
		if addresses[i].GetHexString() > addresses[i+1].GetHexString() {
			test.Fatal(addresses)
		}
	}
}
