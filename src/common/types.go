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

package common

import (
	"common/secp256k1"
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"taslog"
	"utility"
)

const PREFIX = "0x"

func getDefaultCurve() elliptic.Curve {
	//return elliptic.P256()
	return secp256k1.S256()
}

const (
	//默认曲线相关参数开始：
	PubKeyLength = 65 //公钥字节长度，1 bytes curve, 64 bytes x,y。
	SecKeyLength = 97 //私钥字节长度，65 bytes pub, 32 bytes D。
	SignLength   = 65 //签名字节长度，32 bytes r & 32 bytes s & 1 byte recid.
	//默认曲线相关参数结束。
	AddressLength = 32 //地址字节长度(TAS, golang.SHA3，256位)
	HashLength    = 32 //哈希字节长度(golang.SHA3, 256位)。to do : 考虑废弃，直接使用golang的hash.Hash，直接为SHA3_256位，类型一样。
	GroupIdLength = 32
)

var DefaultLogger taslog.Logger

var (
	hashT               = reflect.TypeOf(Hash{})
	addressT            = reflect.TypeOf(Address{})
	BonusStorageAddress = BigToAddress(big.NewInt(0))
	LightDBAddress      = BigToAddress(big.NewInt(1))
	HeavyDBAddress      = BigToAddress(big.NewInt(2))
)

//160位地址
type Address [AddressLength]byte

func (a Address) MarshalJSON() ([]byte, error) {
	return []byte("\"" + a.GetHexString() + "\""), nil
}

//构造函数族
func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

func StringToAddress(s string) Address { return BytesToAddress([]byte(s)) }
func BigToAddress(b *big.Int) Address  { return BytesToAddress(b.Bytes()) }
func HexToAddress(s string) Address    { return BytesToAddress(FromHex(s)) }

//赋值函数，如b超出a的容量则截取后半部分
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[:], b[:])
}

func (a *Address) SetString(s string) {
	a.SetBytes([]byte(s))
}

func (a *Address) Set(other Address) {
	copy(a[:], other[:])
}


// MarshalText returns the hex representation of a.
//把地址编码成十六进制字符串
func (a Address) MarshalText() ([]byte, error) {
	return utility.Bytes(a[:]).MarshalText()
}

// UnmarshalText parses a hash in hex syntax.
//把十六进制字符串解码成地址
func (a *Address) UnmarshalText(input []byte) error {
	return utility.UnmarshalFixedText("Address", input, a[:])
}

// UnmarshalJSON parses a hash in hex syntax.
//把十六进制JSONG格式字符串解码成地址
func (a *Address) UnmarshalJSON(input []byte) error {
	return utility.UnmarshalFixedJSON(addressT, input, a[:])
}

//判断一个字符串是否能转成TAS地址格式
//支持三种类型的字符串，"0xFA10..."，"0XFA10..."和"FA10..."
func IsHexAddress(s string) bool {
	if len(s) == 2+2*AddressLength && IsHex(s) {
		return true
	}
	if len(s) == 2*AddressLength && IsHex("0x"+s) {
		return true
	}
	return false
}

//类型转换输出函数
func (a Address) Str() string          { return string(a[:]) }
func (a Address) Bytes() []byte        { return a[:] }
func (a Address) BigInteger() *big.Int { return new(big.Int).SetBytes(a[:]) }
func (a Address) Hash() Hash           { return BytesToHash(a[:]) }

func (a Address) IsValid() bool {
	return len(a.Bytes()) > 0
}

func (a Address) GetHexString() string {
	str := ToHex(a[:])
	return str
}

func (a Address) String() string {
	return a.GetHexString()
}

func HexStringToAddress(s string) (a Address) {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return
	}
	buf, _ := hex.DecodeString(s[len(PREFIX):])
	if len(buf) == AddressLength {
		a.SetBytes(buf)
	}
	return
}

/*
// Format implements fmt.Formatter, forcing the byte slice to be formatted as is,
// without going through the stringer interface used for logging.
func (a Address) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "%"+string(c), a[:])
}

// UnprefixedHash allows marshaling an Address without 0x prefix.
type UnprefixedAddress Address //无前缀地址

// UnmarshalText decodes the address from hex. The 0x prefix is optional.
//把十六进制字节数组解码成无前缀地址
func (a *UnprefixedAddress) UnmarshalText(input []byte) error {
	return utility.UnmarshalFixedUnprefixedText("UnprefixedAddress", input, a[:])
}

// MarshalText encodes the address as hex.
//把无前缀地址编码成十六进制字节数组
func (a UnprefixedAddress) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(a[:])), nil
}
*/
///////////////////////////////////////////////////////////////////////////////
//256位哈希
type Hash [HashLength]byte

func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}
func StringToHash(s string) Hash { return BytesToHash([]byte(s)) }
func BigToHash(b *big.Int) Hash  { return BytesToHash(b.Bytes()) }
func HexToHash(s string) Hash    { return BytesToHash(FromHex(s)) }

// Get the string representation of the underlying hash
func (h Hash) Str() string   { return string(h[:]) }
func (h Hash) Bytes() []byte { return h[:] }
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }
func (h Hash) Hex() string   { return utility.Encode(h[:]) }

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (h Hash) TerminalString() string {
	return fmt.Sprintf("%x…%x", h[:3], h[29:])
}

func (h Hash) IsValid() bool {
	return len(h.Bytes()) > 0
}

// String implements the stringer interface and is used also by the logger when
// doing full logging into a file.
func (h Hash) String() string {
	return h.Hex()
}

func (h Hash) ShortS() string {
	str := h.Hex()
	return ShortHex12(str)
}

// Format implements fmt.Formatter, forcing the byte slice to be formatted as is,
// without going through the stringer interface used for logging.
func (h Hash) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "%"+string(c), h[:])
}

// UnmarshalText parses a hash in hex syntax.
func (h *Hash) UnmarshalText(input []byte) error {
	return utility.UnmarshalFixedText("Hash", input, h[:])
}

// UnmarshalJSON parses a hash in hex syntax.
func (h *Hash) UnmarshalJSON(input []byte) error {
	return utility.UnmarshalFixedJSON(hashT, input, h[:])
}

// MarshalText returns the hex representation of h.
func (h Hash) MarshalText() ([]byte, error) {
	return utility.Bytes(h[:]).MarshalText()
}

// Sets the hash to the value of b. If b is larger than len(h), 'b' will be cropped (from the left).
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:] //截取右边部分
	}

	copy(h[HashLength-len(b):], b)
}

// Set string `s` to h. If s is larger than len(h) s will be cropped (from left) to fit.
func (h *Hash) SetString(s string) { h.SetBytes([]byte(s)) }

// Sets h from other
func (h *Hash) Set(other Hash) {
	for i, v := range other {
		h[i] = v
	}
}

// Generate implements testing/quick.Generator.
func (h Hash) Generate(rand *rand.Rand, size int) reflect.Value {
	m := rand.Intn(len(h)) //m为0-len(h)之间的伪随机数
	for i := len(h) - 1; i > m; i-- { //从高位到m之间进行遍历
		h[i] = byte(rand.Uint32()) //rand.Uint32为32位非负伪随机数
	}
	return reflect.ValueOf(h)
}

func EmptyHash(h Hash) bool {
	return h == Hash{}
}

// UnprefixedHash allows marshaling a Hash without 0x prefix.
type UnprefixedHash Hash

// UnmarshalText decodes the hash from hex. The 0x prefix is optional.
func (h *UnprefixedHash) UnmarshalText(input []byte) error {
	return utility.UnmarshalFixedUnprefixedText("UnprefixedHash", input, h[:])
}

// MarshalText encodes the hash as hex.
func (h UnprefixedHash) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h[:])), nil
}

type Hash256 Hash
type StorageSize float64

var (
	Big1   = big.NewInt(1)
	Big2   = big.NewInt(2)
	Big3   = big.NewInt(3)
	Big0   = big.NewInt(0)
	Big32  = big.NewInt(32)
	Big256 = big.NewInt(0xff)
	Big257 = big.NewInt(257)

	ErrSelectGroupNil = errors.New("selectGroupId is nil")
	ErrSelectGroupInequal = errors.New("selectGroupId not equal")
	ErrCreateBlockNil = errors.New("createBlock is nil")
)

const (
	// Integer limit values.
	MaxInt8   = 1<<7 - 1
	MinInt8   = -1 << 7
	MaxInt16  = 1<<15 - 1
	MinInt16  = -1 << 15
	MaxInt32  = 1<<31 - 1
	MinInt32  = -1 << 31
	MaxInt64  = 1<<63 - 1
	MinInt64  = -1 << 63
	MaxUint8  = 1<<8 - 1
	MaxUint16 = 1<<16 - 1
	MaxUint32 = 1<<32 - 1
	MaxUint64 = 1<<64 - 1
)
