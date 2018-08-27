package groupsig

import (
	"common"
	"log"
	"bytes"
	"fmt"
	"math/big"
	"consensus/groupsig/bn_curve"
	"consensus/base"
)

// types

// Signature --
type Signature struct {
	value bn_curve.G1
}

func (sig *Signature) Add(sig1 *Signature) error {
	new_sig := &Signature{}
	new_sig.value.Set(&sig.value)
	sig.value.Add(&new_sig.value, &sig1.value)

	return nil
}

func (sig *Signature) Mul(bi *big.Int) error {
	g1 := new(bn_curve.G1)
	g1.Set(&sig.value)
	sig.value.ScalarMult(g1, bi)
	return nil
}

//比较两个签名是否相同
func (sig Signature) IsEqual(rhs Signature) bool {
	fmt.Println("IsEqual rhs:", rhs.value.Marshal())
	fmt.Println("IsEqual sig:", sig.value.Marshal())

	return bytes.Equal(sig.value.Marshal(), rhs.value.Marshal())
	//return sig.value.IsEqual(&rhs.value)
}

//MAP(地址->签名)
type SignatureAMap map[common.Address]Signature
type SignatureIMap map[string]Signature

// Conversion

func (sig Signature) GetHash() common.Hash {
	buf := sig.Serialize()
	return base.Data2CommonHash(buf)
}

//由签名生成随机数
func (sig Signature) GetRand() base.Rand {
	//先取得签名的字节切片（序列化），然后以字节切片为基生成随机数
	return base.RandFromBytes(sig.Serialize())
}

func DeserializeSign(b [] byte)  * Signature {
	sig := &Signature{}
	sig.Deserialize(b)
	return sig
}

//由字节切片初始化签名
func (sig *Signature) Deserialize(b []byte) error {
	sig.value.Unmarshal(b)
	return nil
}

//把签名转换为字节切片
func (sig Signature) Serialize() []byte {
	return sig.value.Marshal()
}

func (sig Signature) IsValid() bool {
	s := sig.Serialize()
	return len(s) > 0
}

//把签名转换为十六进制字符串 ToDoCheck
func (sig Signature) GetHexString() string {
	//return PREFIX + sig.value.GetHexString()
	//return sig.value.GetHexString()

	return ""
}

//由十六进制字符串初始化签名 ToDoCheck
func (sig *Signature) SetHexString(s string) error {
	/*
		if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
			return fmt.Errorf("arg failed")
		}
		buf := s[len(PREFIX):]
		return sig.value.SetHexString(buf)
	*/

	return nil
}

//签名函数。用私钥对明文（哈希）进行签名，返回签名对象
func Sign(sec Seckey, msg []byte) (sig Signature) {
	bg := HashToG1(string(msg))
	sig.value.ScalarMult(bg, sec.GetBigInt())
	//sig.value = *sec.value.Sign(string(msg)) //调用bls曲线的签名函数
	return sig
}

//验证函数。验证某个签名是否来自公钥对应的私钥。 ToDoCheck
func VerifySig(pub Pubkey, msg []byte, sig Signature) bool {
	bQ := bn_curve.GetG2Base()
	p1 := bn_curve.Pair(&sig.value, bQ)

	Hm := HashToG1(string(msg))
	p2 := bn_curve.Pair(Hm, &pub.value)

	return bn_curve.PairIsEuqal(p1, p2)
	//return bytes.Equal(p1.Marshal(), p2.Marshal())
}

//分片合并验证函数。先把公钥切片合并，然后验证该签名是否来自公钥对应的私钥。
func VerifyAggregateSig(pubs []Pubkey, msg []byte, asig Signature) bool {
	pub := AggregatePubkeys(pubs) //用公钥切片合并出组公钥（全部公钥切片而不只是k个）
	if pub == nil {
		return false
	}
	return VerifySig(*pub, msg, asig) //调用验证函数
}

//批量验证函数。
func BatchVerify(pubs []Pubkey, msg []byte, sigs []Signature) bool {
	//把签名切片合并成一个，把公钥签名合并成一个，然后调用签名验证函数。
	return VerifyAggregateSig(pubs, msg, AggregateSigs(sigs))
}

// AggregateXXX函数族是把全部切片相加，而不是k个相加。
//签名聚合函数。用bls曲线加法把多个签名聚合成一个。
func AggregateSigs(sigs []Signature) (sig Signature) {
	n := len(sigs)
	sig = Signature{}
	if n >= 1 {
		sig.value.Set(&sigs[0].value)
		for i := 1; i < n; i++ {
			newsig := &Signature{}
			newsig.value.Set(&sig.value)
			sig.value.Add(&newsig.value, &sigs[i].value)
		}
	}
	return sig
}

//用签名切片和id切片恢复出master签名（通过拉格朗日插值法） 
//RecoverXXX族函数的切片数量都固定是k（门限值）
func RecoverSignature(sigs []Signature, ids []ID) *Signature {
	//secret := big.NewInt(0) //组私钥
	k := len(sigs)          //取得输出切片的大小，即门限值k
	xs := make([]*big.Int, len(ids))
	for i := 0; i < len(xs); i++ {
		xs[i] = ids[i].GetBigInt() //把所有的id转化为big.Int，放到xs切片
	}
	// need len(ids) = k > 0
	sig := &Signature{}
	new_sig := &Signature{}
	for i := 0; i < k; i++ { //输入元素遍历
		// compute delta_i depending on ids only
		//为什么前面delta/num/den初始值是1，最后一个diff初始值是0？
		var delta, num, den, diff *big.Int = big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(0)
		for j := 0; j < k; j++ { //ID遍历
			if j != i { //不是自己
				num.Mul(num, xs[j])    //num值先乘上当前ID
				num.Mod(num, curveOrder)    //然后对曲线域求模
				diff.Sub(xs[j], xs[i]) //diff=当前节点（内循环）-基节点（外循环）
				den.Mul(den, diff)     //den=den*diff
				den.Mod(den, curveOrder)    //den对曲线域求模
			}
		}
		// delta = num / den
		den.ModInverse(den, curveOrder) //模逆
		delta.Mul(num, den)
		delta.Mod(delta, curveOrder)

		//最终需要的值是delta
		new_sig.value.Set(&sigs[i].value)
		new_sig.Mul(delta)

		if i == 0 {
			sig.value.Set(&new_sig.value)
		} else {
			sig.Add(new_sig)
		}
	}
	return sig
}

//签名恢复函数，m为map(ID->签名)，k为门限值
func RecoverSignatureByMapI(m SignatureIMap, k int) *Signature {
	ids := make([]ID, k)
	sigs := make([]Signature, k)
	i := 0
	for s_id, si := range m { //map遍历
		var id ID
		id.SetHexString(s_id)
		ids[i] = id  //组成员ID值
		sigs[i] = si //组成员签名
		i++
		if i >= k {
			break
		}
	}
	return RecoverSignature(sigs, ids) //调用签名恢复函数
}

//签名恢复函数，m为map(地址->签名)，k为门限值
func RecoverSignatureByMapA(m SignatureAMap, k int) *Signature {
	ids := make([]ID, k)
	sigs := make([]Signature, k)
	i := 0
	for a, s := range m { //map遍历
		id := NewIDFromAddress(a) //取得地址对应的ID
		if id == nil {
			log.Printf("RecoverSignatureByMap bad address %s\n", a)
			return nil
		}
		ids[i] = *id //组成员ID值
		sigs[i] = s  //组成员签名
		i++
		if i >= k {
			break
		}
	}
	return RecoverSignature(sigs, ids) //调用签名恢复函数
}

// Recover --
func (sign *Signature) Recover(signVec []Signature, idVec []ID) error {

	return nil
}