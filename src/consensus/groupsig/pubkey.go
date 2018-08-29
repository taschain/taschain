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

package groupsig

import (
	"common"
	"consensus/bls"
	"fmt"
	//"fmt"
	"log"
	"math/big"
	"unsafe"

	"golang.org/x/crypto/sha3"
)

//用户公钥
type Pubkey struct {
	value bls.PublicKey
}

//MAP（地址->公钥）
type PubkeyMap map[common.Address]Pubkey

type PubkeyMapID map[string]Pubkey

//判断两个公钥是否相同
func (pub Pubkey) IsEqual(rhs Pubkey) bool {
	return pub.value.IsEqual(&rhs.value)
}

//由字节切片初始化私钥
func (pub *Pubkey) Deserialize(b []byte) error {
	return pub.value.Deserialize(b)
}

//把公钥转换成字节切片（小端模式？）
func (pub Pubkey) Serialize() []byte {
	return pub.value.Serialize()
}

func (pub Pubkey) MarshalJSON() ([]byte, error) {
	str := "\"" + pub.GetHexString() + "\""
	return []byte(str), nil
}

func (pub *Pubkey) UnmarshalJSON(data []byte) error {
	str := string(data[:])
	if len(str) < 2 {
		return fmt.Errorf("data size less than min.")
	}
	str = str[1:len(str)-1]
	return pub.SetHexString(str)
}

//把公钥转换成big.Int
func (pub Pubkey) GetBigInt() *big.Int {
	x := new(big.Int)
	x.SetString(pub.value.GetHexString(), 16)
	return x
}

func (pub Pubkey) IsValid() bool {
	bi := pub.GetBigInt()
	return bi.Cmp(big.NewInt(0)) != 0
}

//由公钥生成TAS地址
func (pub Pubkey) GetAddress() common.Address {
	h := sha3.Sum256(pub.Serialize())  //取得公钥的SHA3 256位哈希
	return common.BytesToAddress(h[:]) //由256位哈希生成TAS160位地址
}

//把公钥转换成十六进制字符串，不包含0x前缀
func (pub Pubkey) GetHexString() string {
	return PREFIX + pub.value.GetHexString()
	//return pub.value.GetHexString()
}

//由十六进制字符串初始化公钥
func (pub *Pubkey) SetHexString(s string) error {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return fmt.Errorf("arg failed")
	}
	buf := s[len(PREFIX):]
	return pub.value.SetHexString(buf)
	//return pub.value.SetHexString(s)
}

//由私钥构建公钥
func NewPubkeyFromSeckey(sec Seckey) *Pubkey {
	pub := new(Pubkey)
	pub.value = *sec.value.GetPublicKey()
	return pub
}

//构建一个安全性要求不高的公钥
func TrivialPubkey() *Pubkey {
	return NewPubkeyFromSeckey(*TrivialSeckey())
}

//公钥聚合函数，用bls曲线加法把多个公钥聚合成一个
func AggregatePubkeys(pubs []Pubkey) *Pubkey {
	if len(pubs) == 0 {
		log.Printf("AggregatePubkeys no pubs")
		return nil
	}
	pub := new(Pubkey)
	pub.value = pubs[0].value
	for i := 1; i < len(pubs); i++ {
		pub.value.Add(&pubs[i].value) //调用bls曲线的公钥相加函数
	}
	return pub
}

//公钥分片生成函数，用多项式替换生成特定于某个ID的公钥分片
//mpub : master公钥切片
//id : 获得该分片的id
func SharePubkey(mpub []Pubkey, id ID) *Pubkey {
	mpk := *(*[]bls.PublicKey)(unsafe.Pointer(&mpub))
	pub := new(Pubkey)
	err := pub.value.Set(mpk, &id.value) //用master公钥切片和id，调用bls曲线的（公钥）分片生成函数
	if err != nil {
		log.Printf("SharePubkey err=%s id=%s\n", err, id.GetHexString())
		return nil
	}
	return pub
}

//以i作为ID，调用公钥分片生成函数
func SharePubkeyByInt(mpub []Pubkey, i int) *Pubkey {
	return SharePubkey(mpub, *NewIDFromInt(i))
}

//以id+1作为ID，调用公钥分片生成函数
func SharePubkeyByMembershipNumber(mpub []Pubkey, id int) *Pubkey {
	return SharePubkey(mpub, *NewIDFromInt(id + 1))
}

func DeserializePubkeyBytes(bytes []byte) *Pubkey {
	var pk Pubkey
	if err := pk.Deserialize(bytes); err != nil {
		return nil
	}
	return &pk
}
