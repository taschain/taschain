package groupsig

import (
	"common"
	"consensus/bls"
	"consensus/rand"
	"fmt"
	"log"
	"math/big"
	"unsafe"
)

// Curve and Field order
var curveOrder = new(big.Int) //曲线整数域
var fieldOrder = new(big.Int)
var bitLength int

// Seckey -- represented by a big.Int modulo curveOrder
//私钥对象，表现为一个大整数在曲线域上的求模？
type Seckey struct {
	value bls.SecretKey
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

type SeckeyMapID map[ID]Seckey

//把私钥转换成字节切片（小端模式）
func (sec Seckey) Serialize() []byte {
	return sec.value.GetLittleEndian()
}

//把私钥转换成big.Int
func (sec Seckey) GetBigInt() (s *big.Int) {
	s = new(big.Int)
	s.SetString(sec.getHex(), 16)
	return s
}

func (sec Seckey) IsValid() bool {
	bi := sec.GetBigInt()
	return bi != big.NewInt(0)
}

//返回十六进制字符串表示，不带前缀
func (sec Seckey) getHex() string {
	return sec.value.GetHexString()
}

//返回十六进制字符串表示，带前缀
func (sec Seckey) GetHexString() string {
	return PREFIX + sec.getHex()
}

//由字节切片初始化私钥
func (sec *Seckey) Deserialize(b []byte) error {
	//to do : 对字节切片做检查
	return sec.value.SetLittleEndian(b)
}

//由字节切片（小端模式）初始化私钥
func (sec *Seckey) SetLittleEndian(b []byte) error {
	return sec.value.SetLittleEndian(b) //调用bls C库曲线函数
}

//由不带前缀的十六进制字符串转换
func (sec *Seckey) setHex(s string) error {
	return sec.value.SetHexString(s)
}

//由带前缀的十六进制字符串转换
func (sec *Seckey) SetHexString(s string) error {
	fmt.Printf("begin SecKey.SetHexString...\n")
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return fmt.Errorf("arg failed")
	}
	buf := s[len(PREFIX):]
	return sec.setHex(buf)
}

//由字节切片（小端模式）构建私钥
func NewSeckeyFromLittleEndian(b []byte) *Seckey {
	sec := new(Seckey)
	err := sec.SetLittleEndian(b)
	if err != nil {
		log.Printf("NewSeckeyFromLittleEndian %s\n", err)
		return nil
	}
	return sec
}

//由随机数构建私钥
func NewSeckeyFromRand(seed rand.Rand) *Seckey {
	//把随机数转换成字节切片（小端模式）后构建私钥
	return NewSeckeyFromLittleEndian(seed.Bytes())
}

//由大整数构建私钥
func NewSeckeyFromBigInt(b *big.Int) *Seckey {
	b.Mod(b, curveOrder) //大整数在曲线域上求模
	sec := new(Seckey)
	err := sec.value.SetDecString(b.Text(10)) //把模数转换为十进制字符串，然后构建私钥
	if err != nil {
		log.Printf("NewSeckeyFromBigInt %s\n", err)
		return nil
	}
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

//私钥聚合函数，用bls曲线加法把多个私钥聚合成一个
func AggregateSeckeys(secs []Seckey) *Seckey {
	if len(secs) == 0 { //没有私钥要聚合
		log.Printf("AggregateSeckeys no secs")
		return nil
	}
	sec := new(Seckey)               //创建一个新的私钥
	sec.value = secs[0].value        //以第一个私钥作为基
	for i := 1; i < len(secs); i++ { //把其后的私钥调用bls的add加到基私钥
		sec.value.Add(&secs[i].value) //调用bls曲线的私钥相加函数
	}
	return sec
}

//用多项式替换生成特定于某个ID的签名私钥分片
//msec : master私钥切片
//id : 获得该分片的id
func ShareSeckey(msec []Seckey, id ID) *Seckey {
	msk := *(*[]bls.SecretKey)(unsafe.Pointer(&msec))
	sec := new(Seckey)
	err := sec.value.Set(msk, &id.value) //用master私钥切片和id，调用bls曲线的（私钥）分片生成函数
	if err != nil {
		log.Printf("ShareSeckey err=%s id=%s\n", err, id.GetHexString())
		return nil
	}
	return sec
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
	//签名私钥分片向量
	secVec := *(*[]bls.SecretKey)(unsafe.Pointer(&secs))
	//ID向量
	idVec := *(*[]bls.ID)(unsafe.Pointer(&ids))
	sec := new(Seckey)
	err := sec.value.Recover(secVec, idVec) //调用bls曲线的（组）私钥恢复函数
	if err != nil {
		log.Printf("RecoverSeckey err=%s\n", err)
		return nil
	}
	return sec
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
