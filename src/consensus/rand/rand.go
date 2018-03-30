package rand

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
)

//随机数长度=32*8=356位，跟使用的哈希函数有相关性
const RandLength = 32

type Rand [RandLength]byte

//以多维字节数组作为种子，进行SHA3哈希生成随机数（底层函数）
func RandFromBytes(b ...[]byte) (r Rand) {
	HashBytes(b...).Sum(r[:0])
	return
}

//对系统强随机种子（unix和windows系统的实现不同）进行SHA3哈希生成随机数
func NewRand() (r Rand) {
	b := make([]byte, RandLength)
	rand.Read(b)
	return RandFromBytes(b)
}

//把多维字符串转换成多维字节数组后，进行SHA3哈希生成随机数
func RandFromHex(s ...string) (r Rand) {
	return RandFromBytes(MapHexToBytes(s)...)
}

//把随机数转换成字节数组
func (r Rand) Bytes() []byte {
	return r[:]
}

//把随机数转换成16进制字符串（不包含0x前缀）
func (r Rand) GetHexString() string {
	return hex.EncodeToString(r[:])
}

// r.DerivedRand(x) := Rand(r,x) := H(r || x) converted to Rand
// r.DerivedRand(x1,x2) := Rand(Rand(r,x1),x2)
//哈希叠加函数。以随机数r为基，以多维字节数组x为变量进行SHA3处理，生成衍生随机数。
//硬计算量，没法做优化，具有良好的抗量子攻击。
func (r Rand) DerivedRand(x ...[]byte) Rand {
	ri := r
	for _, xi := range x { //遍历多维字节数组
		HashBytes(ri.Bytes(), xi).Sum(ri[:0]) //哈希叠加计算
	}
	return ri
}

//以多维字符串进行哈希叠加
func (r Rand) Ders(s ...string) Rand {
	return r.DerivedRand(MapStringToBytes(s)...)
}

//以多维整数进行哈希叠加
func (r Rand) Deri(vi ...int) Rand {
	return r.Ders(MapItoa(vi)...)
}

//随机数求模操作，返回0到n-1之间的一个值
func (r Rand) Modulo(n int) int {
	b := big.NewInt(0)
	b.SetBytes(r.Bytes())          //随机数转换成big.Int
	b.Mod(b, big.NewInt(int64(n))) //对n求模
	return int(b.Int64())
}

//从0到n-1区间中随机取k个数（以r为随机基），输出这个随机序列
func (r Rand) RandomPerm(n int, k int) []int {
	l := make([]int, n)
	for i := range l {
		l[i] = i
	}
	for i := 0; i < k; i++ {
		j := r.Deri(i).Modulo(n-i) + i
		l[i], l[j] = l[j], l[i]
	}
	return l[:k]
}
