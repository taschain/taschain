package rand

import (
	"hash"
	"golang.org/x/crypto/sha3"
)

// Hash helpers

//生成多维字节数组的SHA3_256位哈希
func HashBytes(b ...[]byte) hash.Hash {
	d := sha3.New256()
	for _, bi := range b {
		d.Write(bi)
	}
	return d
}
