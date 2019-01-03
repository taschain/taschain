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
	"fmt"
	//"fmt"
	"log"
	"golang.org/x/crypto/sha3"
	"bytes"
	"consensus/groupsig/bn_curve"
)

//用户公钥
type Pubkey struct {
	value bn_curve.G2
}


type PubkeyMap map[string]Pubkey

func (pub Pubkey) IsEmpty() bool {
	return pub.value.IsEmpty()
}

//判断两个公钥是否相同   
func (pub Pubkey) IsEqual(rhs Pubkey) bool {
	return  bytes.Equal(pub.value.Marshal(), rhs.value.Marshal())
}

//由字节切片初始化私钥  ToDoCheck
func (pub *Pubkey) Deserialize(b []byte) error {
	_, error := pub.value.Unmarshal(b)
	return error
}

//把公钥转换成字节切片（小端模式？）
func (pub Pubkey) Serialize() []byte {
	return pub.value.Marshal()
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

////把公钥转换成big.Int  ToDoCheck
//func (pub Pubkey) GetBigInt() *big.Int {
//	//x := new(big.Int)
//	//x.SetString(pub.value.GetHexString(), 16)
//	//return x
//
//	return nil
//}

func (pub Pubkey) IsValid() bool {
	return !pub.IsEmpty()
	//bi := pub.GetBigInt()
	//return bi.Cmp(big.NewInt(0)) != 0
}

//由公钥生成TAS地址
func (pub Pubkey) GetAddress() common.Address {
	h := sha3.Sum256(pub.Serialize())  //取得公钥的SHA3 256位哈希
	return common.BytesToAddress(h[:]) //由256位哈希生成TAS160位地址
}

//把公钥转换成十六进制字符串，不包含0x前缀   ToDoCheck
func (pub Pubkey) GetHexString() string {
	return PREFIX + common.Bytes2Hex(pub.value.Marshal())
}

func (pub *Pubkey) ShortS() string {
	str := pub.GetHexString()
	return common.ShortHex12(str)
}
//由十六进制字符串初始化公钥  ToDoCheck
func (pub *Pubkey) SetHexString(s string) error {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return fmt.Errorf("arg failed")
	}
	buf := s[len(PREFIX):]

	pub.value.Unmarshal(common.Hex2Bytes(buf))
	return nil
}

//由私钥构建公钥
func NewPubkeyFromSeckey(sec Seckey) *Pubkey {
	pub := new(Pubkey)
	pub.value.ScalarBaseMult(sec.value.GetBigInt())
	return pub
}

//构建一个安全性要求不高的公钥
func TrivialPubkey() *Pubkey {
	return NewPubkeyFromSeckey(*TrivialSeckey())
}

func (pub *Pubkey) Add(rhs *Pubkey) error {
	pa := &bn_curve.G2{}
	pb := &bn_curve.G2{}

	pa.Set(&pub.value)
	pb.Set(&rhs.value)

	pub.value.Add(pa, pb)
	return nil
}

//公钥聚合函数
func AggregatePubkeys(pubs []Pubkey) *Pubkey {
	if len(pubs) == 0 {
		log.Printf("AggregatePubkeys no pubs")
		return nil
	}

	pub := new(Pubkey)
	//pub.value.Unmarshal(pubs[0].value.Marshal())
	pub.value.Set(&pubs[0].value)

	//pub.value = pubs[0].value
	//for i := 0; i < len(pubs); i++ {
		//log.Println("", i)
		//log.Println("pub:", pubs[i].Serialize())
	//}

	//log.Println("len:", len(pubs))
	for i := 1; i < len(pubs); i++ {
		pub.Add(&pubs[i])
	}

	//log.Println("aggregatePK:", pub.Serialize())

	return pub
}

//公钥分片生成函数，用多项式替换生成特定于某个ID的公钥分片  
//mpub : master公钥切片
//id : 获得该分片的id
func SharePubkey(mpub []Pubkey, id ID) *Pubkey {
	pub := &Pubkey{}
	// degree of polynomial, need k >= 1, i.e. len(msec) >= 2
	k := len(mpub) - 1
	// msec = c_0, c_1, ..., c_k
	// evaluate polynomial f(x) with coefficients c0, ..., ck
	pub.Deserialize(mpub[k].Serialize())

	x := id.GetBigInt()             //取得id的big.Int值
	for j := k - 1; j >= 0; j-- {   //从master key切片的尾部-1往前遍历
		pub.value.ScalarMult(&pub.value, x)
		pub.value.Add(&pub.value, &mpub[j].value)
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

func DeserializePubkeyBytes(bytes []byte) Pubkey {
	var pk Pubkey
	if err := pk.Deserialize(bytes); err != nil {
		return Pubkey{}
	}
	return pk
}
