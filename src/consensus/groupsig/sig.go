package groupsig

import (
	"common"
	"consensus/bls"
	"log"
	"unsafe"
	"consensus/base"
)

// types

// Signature --
type Signature struct {
	value bls.Sign
}

//比较两个签名是否相同
func (sig Signature) IsEqual(rhs Signature) bool {
	return sig.value.IsEqual(&rhs.value)
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

//由字节切片初始化签名
func (sig *Signature) Deserialize(b []byte) error {
	return sig.value.Deserialize(b)
}

func DeserializeSign(b []byte) *Signature {
	var sign = &Signature{}
	if err := sign.Deserialize(b); err != nil {
		return nil
	}
	return sign
}

//把签名转换为字节切片
func (sig Signature) Serialize() []byte {
	return sig.value.Serialize()
}

func (sig Signature) IsValid() bool {
	s := sig.Serialize()
	return len(s) > 0
}

//把签名转换为十六进制字符串
func (sig Signature) GetHexString() string {
	//return PREFIX + sig.value.GetHexString()
	return sig.value.GetHexString()
}

//由十六进制字符串初始化签名
func (sig *Signature) SetHexString(s string) error {
	/*
		if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
			return fmt.Errorf("arg failed")
		}
		buf := s[len(PREFIX):]
		return sig.value.SetHexString(buf)
	*/
	return sig.value.SetHexString(s)
}

//签名函数。用私钥对明文（哈希）进行签名，返回签名对象
func Sign(sec Seckey, msg []byte) (sig Signature) {
	sig.value = *sec.value.Sign(string(msg)) //调用bls曲线的签名函数
	return sig
}

//验证函数。验证某个签名是否来自公钥对应的私钥。
func VerifySig(pub Pubkey, msg []byte, sig Signature) bool {
	return sig.value.Verify(&pub.value, string(msg)) //调用bls曲线的验证函数
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
	if n >= 1 {
		sig.value = sigs[0].value
		for i := 1; i < n; i++ {
			sig.value.Add(&sigs[i].value)
		}
	}
	return sig
}

//用签名切片和id切片恢复出master签名（通过拉格朗日插值法）
//RecoverXXX族函数的切片数量都固定是k（门限值）
func RecoverSignature(sigs []Signature, ids []ID) *Signature {
	signVec := *(*[]bls.Sign)(unsafe.Pointer(&sigs))
	idVec := *(*[]bls.ID)(unsafe.Pointer(&ids))
	sig := new(Signature)
	err := sig.value.Recover(signVec, idVec)
	if err != nil {
		log.Printf("RecoverSignature err=%s\n", err)
		return nil
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
