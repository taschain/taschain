package common

import (
	"sync"
	"hash"
	"crypto/sha256"
	"bytes"
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

//BytesCombine 多个[]byte数组合并成一个[]byte
func BytesCombine(pBytes ...[]byte) []byte {
	len := len(pBytes)
	s := make([][]byte, len)
	for index := 0; index < len; index++ {
		s[index] = pBytes[index]
	}
	sep := []byte("")
	return bytes.Join(s, sep)
}