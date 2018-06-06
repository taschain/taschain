package common

import (
	"sync"
	"hash"
	"crypto/sha256"
)

var hasherPool = sync.Pool{
New: func() interface{} {
return sha256.New()
},
}

// 计算sha256
func Sha256(blockByte []byte) []byte {
hasher := hasherPool.Get().(hash.Hash)
hasher.Reset()
defer hasherPool.Put(hasher)

hasher.Write(blockByte)
return hasher.Sum(nil)

}