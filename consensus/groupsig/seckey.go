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
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/base"
	"github.com/taschain/taschain/consensus/groupsig/bn_curve"
	"log"
	"math/big"
)

// Curve and Field order
var curveOrder = bn_curve.Order //曲线整数域
var fieldOrder = bn_curve.P
var bitLength = curveOrder.BitLen()

// Seckey -- represented by a big.Int modulo curveOrder
//私钥对象，表现为一个大整数在曲线域上的求模？
type Seckey struct {
	value BnInt
}

//比较两个私钥是否相等
func (sec Seckey) IsEqual(rhs Seckey) bool {
	return sec.value.IsEqual(&rhs.value)
}

//MAP(地址->私钥)
type SeckeyMap map[common.Address]Seckey

// SeckeyMapInt -- a map from addresses to Seckey
//map(地址->私钥)
type SeckeyMapInt map[int]Seckey

type SeckeyMapID map[string]Seckey

//把私钥转换成字节切片（小端模式）
func (sec Seckey) Serialize() []byte {
	return sec.value.Serialize()
}

//把私钥转换成big.Int
func (sec Seckey) GetBigInt() (s *big.Int) {
	s = new(big.Int)
	s.Set(sec.value.GetBigInt())
	//s.SetString(sec.getHex(), 16)
	return s
}

func (sec Seckey) IsValid() bool {
	bi := sec.GetBigInt()
	return bi.Cmp(big.NewInt(0)) != 0
}

//返回十六进制字符串表示，不带前缀
func (sec Seckey) getHex() string {
	return sec.value.GetHexString()
}

//返回十六进制字符串表示，带前缀
func (sec Seckey) GetHexString() string {
	return sec.getHex()
}

func (sec *Seckey) ShortS() string {
	str := sec.GetHexString()
	return common.ShortHex12(str)
}

//由字节切片初始化私钥
func (sec *Seckey) Deserialize(b []byte) error {
	//to do : 对字节切片做检查
	return sec.value.Deserialize(b)
}

func (sec Seckey) MarshalJSON() ([]byte, error) {
	str := "\"" + sec.GetHexString() + "\""
	return []byte(str), nil
}

func (sec *Seckey) UnmarshalJSON(data []byte) error {
	str := string(data[:])
	if len(str) < 2 {
		return fmt.Errorf("data size less than min.")
	}
	str = str[1 : len(str)-1]
	return sec.SetHexString(str)
}

//由字节切片（小端模式）初始化私钥
func (sec *Seckey) SetLittleEndian(b []byte) error {
	return sec.value.Deserialize(b[:32])
}

//由不带前缀的十六进制字符串转换
func (sec *Seckey) setHex(s string) error {
	return sec.value.SetHexString(s)
}

//由带前缀的十六进制字符串转换
func (sec *Seckey) SetHexString(s string) error {
	////fmt.Printf("begin SecKey.SetHexString...\n")
	//if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
	//	return fmt.Errorf("arg failed")
	//}
	//buf := s[len(PREFIX):]
	return sec.setHex(s)
}

//由字节切片（小端模式）构建私钥
func NewSeckeyFromLittleEndian(b []byte) *Seckey {
	sec := new(Seckey)
	err := sec.SetLittleEndian(b)
	if err != nil {
		log.Printf("NewSeckeyFromLittleEndian %s\n", err)
		return nil
	}

	sec.value.Mod()
	return sec
}

//由随机数构建私钥
func NewSeckeyFromRand(seed base.Rand) *Seckey {
	//把随机数转换成字节切片（小端模式）后构建私钥
	return NewSeckeyFromLittleEndian(seed.Bytes())
}

//由大整数构建私钥
func NewSeckeyFromBigInt(b *big.Int) *Seckey {
	nb := &big.Int{}
	nb.Set(b)
	b.Mod(nb, curveOrder) //大整数在曲线域上求模

	sec := new(Seckey)
	sec.value.SetBigInt(b)

	return sec
}

//由int64构建私钥
func NewSeckeyFromInt64(i int64) *Seckey {
	return NewSeckeyFromBigInt(big.NewInt(i))
}

//由int32构建私钥
func NewSeckeyFromInt(i int) *Seckey {
	return NewSeckeyFromBigInt(big.NewInt(int64(i)))
}

//构建一个安全性要求不高的私钥
func TrivialSeckey() *Seckey {
	return NewSeckeyFromInt64(1) //以1作为跳频
}

//私钥聚合函数
func AggregateSeckeys(secs []Seckey) *Seckey {
	if len(secs) == 0 { //没有私钥要聚合
		log.Printf("AggregateSeckeys no secs")
		return nil
	}
	sec := new(Seckey) //创建一个新的私钥
	sec.value.SetBigInt(secs[0].value.GetBigInt())
	//sec.value = secs[0].value        //以第一个私钥作为基
	for i := 1; i < len(secs); i++ {
		sec.value.Add(&secs[i].value)
	}

	x := new(big.Int)
	x.Set(sec.value.GetBigInt())
	sec.value.SetBigInt(x.Mod(x, curveOrder))
	return sec
}

//用多项式替换生成特定于某个ID的签名私钥分片
//msec : master私钥切片
//id : 获得该分片的id
func ShareSeckey(msec []Seckey, id ID) *Seckey {
	secret := big.NewInt(0)
	k := len(msec) - 1

	// evaluate polynomial f(x) with coefficients c0, ..., ck
	secret.Set(msec[k].GetBigInt()) //最后一个master key的big.Int值放到secret
	x := id.GetBigInt()             //取得id的big.Int值
	new_b := &big.Int{}

	for j := k - 1; j >= 0; j-- { //从master key切片的尾部-1往前遍历
		new_b.Set(secret)
		secret.Mul(new_b, x) //乘上id的big.Int值，每一遍都需要乘，所以是指数？

		new_b.Set(secret)
		secret.Add(new_b, msec[j].GetBigInt()) //加法

		new_b.Set(secret)
		secret.Mod(new_b, curveOrder) //曲线域求模
	}

	return NewSeckeyFromBigInt(secret) //生成签名私钥
}

//由master私钥切片和TAS地址生成针对该地址的签名私钥分片
func ShareSeckeyByAddr(msec []Seckey, addr common.Address) *Seckey {
	id := NewIDFromAddress(addr)
	if id == nil {
		log.Printf("ShareSeckeyByAddr bad addr=%s\n", addr)
		return nil
	}
	return ShareSeckey(msec, *id)
}

//由master私钥切片和整数i生成签名私钥分片
func ShareSeckeyByInt(msec []Seckey, i int) *Seckey {
	return ShareSeckey(msec, *NewIDFromInt64(int64(i)))
}

//由master私钥切片和整数id，生成id+1者的签名私钥分片
func ShareSeckeyByMembershipNumber(msec []Seckey, id int) *Seckey {
	return ShareSeckey(msec, *NewIDFromInt64(int64(id + 1)))
}

//用（签名）私钥分片切片和id切片恢复出master私钥（通过拉格朗日插值法）
//私钥切片和ID切片的数量固定为门限值k
func RecoverSeckey(secs []Seckey, ids []ID) *Seckey {
	secret := big.NewInt(0) //组私钥
	k := len(secs)          //取得输出切片的大小，即门限值k
	//fmt.Println("k:", k)
	xs := make([]*big.Int, len(ids))
	for i := 0; i < len(xs); i++ {
		xs[i] = ids[i].GetBigInt() //把所有的id转化为big.Int，放到xs切片
	}
	// need len(ids) = k > 0
	for i := 0; i < k; i++ { //输入元素遍历
		// compute delta_i depending on ids only
		//为什么前面delta/num/den初始值是1，最后一个diff初始值是0？
		var delta, num, den, diff *big.Int = big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(0)
		for j := 0; j < k; j++ { //ID遍历
			if j != i { //不是自己
				num.Mul(num, xs[j])      //num值先乘上当前ID
				num.Mod(num, curveOrder) //然后对曲线域求模
				diff.Sub(xs[j], xs[i])   //diff=当前节点（内循环）-基节点（外循环）
				den.Mul(den, diff)       //den=den*diff
				den.Mod(den, curveOrder) //den对曲线域求模
			}
		}
		// delta = num / den
		den.ModInverse(den, curveOrder) //模逆
		delta.Mul(num, den)
		delta.Mod(delta, curveOrder)
		//最终需要的值是delta
		// apply delta to secs[i]
		delta.Mul(delta, secs[i].GetBigInt()) //delta=delta*当前节点私钥的big.Int
		// skip reducing delta modulo curveOrder here
		secret.Add(secret, delta)      //把delta加到组私钥（big.Int形式）
		secret.Mod(secret, curveOrder) //组私钥对曲线域求模（big.Int形式）
	}

	return NewSeckeyFromBigInt(secret)
}

//私钥恢复函数，m为map(地址->私钥)，k为门限值
func RecoverSeckeyByMap(m SeckeyMap, k int) *Seckey {
	ids := make([]ID, k)
	secs := make([]Seckey, k)
	i := 0
	for a, s := range m { //map遍历
		id := NewIDFromAddress(a) //提取地址对应的id
		if id == nil {
			log.Printf("RecoverSeckeyByMap bad Address %s\n", a)
			return nil
		}
		ids[i] = *id //组成员ID
		secs[i] = s  //组成员签名私钥
		i++
		if i >= k { //取到门限值
			break
		}
	}
	return RecoverSeckey(secs, ids) //调用私钥恢复函数
}

// RecoverSeckeyByMapInt --
//从签名私钥分片map中取k个（门限值）恢复出组私钥
func RecoverSeckeyByMapInt(m SeckeyMapInt, k int) *Seckey {
	ids := make([]ID, k)      //k个ID
	secs := make([]Seckey, k) //k个签名私钥分片
	i := 0
	//取map里的前k个签名私钥生成恢复基
	for a, s := range m {
		ids[i] = *NewIDFromInt64(int64(a))
		secs[i] = s
		i++
		if i >= k {
			break
		}
	}
	//恢复出组私钥
	return RecoverSeckey(secs, ids)
}

// Set --
func (sec *Seckey) Set(msk []Seckey, id *ID) error {
	// #nosec
	s := ShareSeckey(msk, *id)
	sec.Deserialize(s.Serialize())
	return nil
}

// Recover --
func (sec *Seckey) Recover(secVec []Seckey, idVec []ID) error {
	// #nosec
	s := RecoverSeckey(secVec, idVec)
	sec.Deserialize(s.Serialize())

	return nil
}
