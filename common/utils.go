package common

import (
	"fmt"
	"github.com/hashicorp/golang-lru"
)

/*
**  Creator: pxf
**  Date: 2019/5/8 上午11:37
**  Description:
 */

// MustNewLRUCache creates a new lru cache.
// Caution: if fail, the function will cause panic
func MustNewLRUCache(size int) *lru.Cache {
	cache, err := lru.New(size)
	if err != nil {
		panic(fmt.Errorf("new cache fail:%v", err))
	}
	return cache
}

// MustNewLRUCacheWithEvictCB creates a new lru cache with buffer eviction
// Caution: if fail, the function will cause panic.
func MustNewLRUCacheWithEvictCB(size int, cb func(k, v interface{})) *lru.Cache {
	cache, err := lru.NewWithEvict(size, cb)
	if err != nil {
		panic(fmt.Errorf("new cache fail:%v", err))
	}
	return cache
}
