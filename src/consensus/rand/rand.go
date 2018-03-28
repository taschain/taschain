package rand

// TODO
// - rename String() to HexString()

import (
    "crypto/rand"
    "encoding/hex"
    "math/big"
)

/// Rand

// RandLength -- fixed length of all random values
// Everything is written for this length. To change it the selected hash functions have to be changed.
//随机数长度，跟算法有相关性
const RandLength = 32

// Rand - the type of our random values
//随机数类型，32个byte，即256位。
type Rand [RandLength]byte

// Constructors

// NewRand -- initialize and return a random value
//生成一个新的默认随机数
func NewRand() (r Rand) {
    b := make([]byte, RandLength)
    rand.Read(b)
    return RandFromBytes(b)
}

// RandFromBytes -- convert one or more byte arrays to a fixed length randomness through hashing
//把一个或多个字节数组转化为固定长度的随机数（通过哈希的方式）
func RandFromBytes(b ...[]byte) (r Rand) {
    HashBytes(b...).Sum(r[:0])
    return
}

// RandFromHex -- convert one or more hex strings to a fixed length randomness through hashing
//把一个或多个16进制字符串转化为固定长度的随机数（通过哈希的方式）
func RandFromHex(s ...string) (r Rand) {
    return RandFromBytes(MapHexToBytes(s)...)
}

// Getters

// Bytes -- Return Rand as []byte
//把一个随机数用字节数组的方式返回
func (r Rand) Bytes() []byte {
    return r[:]
}

// String -- Return Rand as hex string, not prefixed with 0x
//把一个随机数用16进制字符串的方式返回，不包含0x前缀
func (r Rand) String() string {
    return hex.EncodeToString(r[:])
}

// DerivedRand -- Derived Randomness hierarchically
// r.DerivedRand(x) := Rand(r,x) := H(r || x) converted to Rand
// r.DerivedRand(x1,x2) := Rand(Rand(r,x1),x2)
//随机衍生函数，以随机数r为基，以参数x为变量，生成衍生随机数。
func (r Rand) DerivedRand(x ...[]byte) Rand {
    // Keccak is not susceptible to length-extension-attacks, so we can use it as-is to implement an HMAC
    //HMAC:基于哈希的消息验证码
    //KECCAK是可以抵抗长度扩展攻击的，所以我们可以使用它来实现HMAC
    ri := r
    for _, xi := range x {
        HashBytes(ri.Bytes(),xi).Sum(ri[:0])
    }
    return ri
}

// Shortcuts to the derivation function

// Ders -- Derive randomness with indices given as strings
func (r Rand) Ders(s ...string) Rand {
    return r.DerivedRand(MapStringToBytes(s)...)
}

// Deri -- Derive randomness with indices given as ints
func (r Rand) Deri(vi ...int) Rand {
    return r.Ders(MapItoa(vi)...)
}

// Modulo -- Return Rand as integer modulo n
// Convert to a random integer from the interval [0,n-1].
func (r Rand) Modulo(n int) int {
    //var b big.Int
    b := big.NewInt(0)
    b.SetBytes(r.Bytes())
    b.Mod(b, big.NewInt(int64(n)))
    return int(b.Int64())
}

// RandomPerm -- A permutation deterministically derived from Rand
// Produces a sequence of k integers from the interval[0,n-1] without repetitions
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
