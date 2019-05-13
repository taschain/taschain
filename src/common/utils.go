package common

import (
	"github.com/hashicorp/golang-lru"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2019/5/8 上午11:37
**  Description: 
*/

func MustNewLRUCache(size int) *lru.Cache {
	cache, err := lru.New(size)
	if err != nil {
		panic(fmt.Errorf("new cache fail:%v", err))
	}
	return cache
}

func MustNewLRUCacheWithEvitCB(size int, cb func(k, v interface{})) *lru.Cache {
	cache, err := lru.NewWithEvict(size, cb)
	if err != nil {
		panic(fmt.Errorf("new cache fail:%v", err))
	}
	return cache
}