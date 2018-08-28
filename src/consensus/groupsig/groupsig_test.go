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
	"consensus/bls"
	"fmt"
	"math/big"
	"testing"
	"time"
	"unsafe"
	"consensus/base"
)

type Expect struct {
	bitLen int
	ok     []byte
}

//测试用衍生随机数生成私钥，从私钥萃取公钥，以及公钥的序列化
func testPubkey(t *testing.T) {
	fmt.Printf("\nbegin test pub key...\n")
	t.Log("testPubkey")
	r := base.NewRand() //生成随机数

	fmt.Printf("size of rand = %v\n.", len(r))
	sec := NewSeckeyFromRand(r.Deri(1)) //以r的衍生随机数生成私钥
	if sec == nil {
		t.Fatal("NewSeckeyFromRand")
	}
	pub := NewPubkeyFromSeckey(*sec) //从私钥萃取出公钥
	if pub == nil {
		t.Log("NewPubkeyFromSeckey")
	}
	{
		var pub2 Pubkey
		err := pub2.SetHexString(pub.GetHexString()) //测试公钥的字符串导出
		if err != nil || !pub.IsEqual(pub2) {        //检查字符串导入生成的公钥是否跟之前的公钥相同
			t.Log("pub != pub2")
		}
	}
	{
		var pub2 Pubkey
		err := pub2.Deserialize(pub.Serialize()) //测试公钥的序列化
		if err != nil || !pub.IsEqual(pub2) {    //检查反序列化生成的公钥是否跟之前的公钥相同
			t.Log("pub != pub2")
		}
	}
	fmt.Printf("\nend test pub key.\n")
}

//用big.Int生成私钥，取得公钥和签名。然后对私钥、公钥和签名各复制一份后测试加法后的验证是否正确。
//同时测试签名的序列化。
func testComparison(t *testing.T) {
	fmt.Printf("\nbegin test Comparison...\n")
	t.Log("begin testComparison")
	var b = new(big.Int)
	b.SetString("16798108731015832284940804142231733909759579603404752749028378864165570215948", 10)
	sec := NewSeckeyFromBigInt(b) //从big.Int（固定的常量）生成原始私钥
	t.Log("sec.Hex: ", sec.GetHexString())

	// Add Seckeys
	sum := AggregateSeckeys([]Seckey{*sec, *sec}) //同一个原始私钥相加，生成聚合私钥（bls底层算法）
	if sum == nil {
		t.Error("AggregateSeckeys failed.")
	}

	// Pubkey
	pub := NewPubkeyFromSeckey(*sec) //从原始私钥萃取出公钥
	if pub == nil {
		t.Error("NewPubkeyFromSeckey failed.")
	} else {
		fmt.Printf("size of pub key = %v.\n", len(pub.Serialize()))
	}

	// Sig
	sig := Sign(*sec, []byte("hi")) //以原始私钥对明文签名，生成原始签名
	fmt.Printf("size of sign = %v\n.", len(sig.Serialize()))
	asig := AggregateSigs([]Signature{sig, sig})                       //同一个原始签名相加，生成聚合签名
	if !VerifyAggregateSig([]Pubkey{*pub, *pub}, []byte("hi"), asig) { //对同一个原始公钥进行聚合后（生成聚合公钥），去验证聚合签名
		t.Error("Aggregated signature does not verify")
	}
	{
		var sig2 Signature
		err := sig2.SetHexString(sig.GetHexString()) //测试原始签名的字符串导出
		if err != nil || !sig.IsEqual(sig2) {        //检查字符串导入生成的签名是否和之前的签名相同
			t.Error("sig2.SetHexString")
		}
	}
	{
		var sig2 Signature
		err := sig2.Deserialize(sig.Serialize()) //测试原始签名的序列化
		if err != nil || !sig.IsEqual(sig2) {    //检查反序列化生成的签名是否跟之前的签名相同
			t.Error("sig2.Deserialize")
		}
	}
	t.Log("end testComparison")
	fmt.Printf("\nend test Comparison.\n")
}

//测试从big.Int生成私钥，以及私钥的序列化
func testSeckey(t *testing.T) {
	fmt.Printf("\nbegin test sec key...\n")
	t.Log("testSeckey")
	s := "401035055535747319451436327113007154621327258807739504261475863403006987855"
	var b = new(big.Int)
	b.SetString(s, 10)
	sec := NewSeckeyFromBigInt(b) //以固定的字符串常量构建私钥
	str := sec.GetHexString()
	fmt.Printf("sec key export, len=%v, data=%v.\n", len(str), str)
	{
		var sec2 Seckey
		err := sec2.SetHexString(str)         //测试私钥的十六进制字符串导出
		if err != nil || !sec.IsEqual(sec2) { //检查字符串导入生成的私钥是否和之前的私钥相同
			t.Error("bad SetHexString")
		}
		str = sec2.GetHexString()
		fmt.Printf("sec key import and export again, len=%v, data=%v.\n", len(str), str)
	}
	{
		var sec2 Seckey
		err := sec2.Deserialize(sec.Serialize()) //测试私钥的序列化
		if err != nil || !sec.IsEqual(sec2) {    //检查反序列化生成的私钥是否和之前的私钥相同
			t.Error("bad Serialize")
		}
	}
	fmt.Printf("end test sec key.\n")
}

//生成n个衍生随机数私钥，对这n个衍生私钥进行聚合生成组私钥，然后萃取出组公钥
func testAggregation(t *testing.T) {
	fmt.Printf("\nbegin test Aggregation...\n")
	t.Log("testAggregation")
	//    m := 5
	n := 3
	//    groupPubkeys := make([]Pubkey, m)
	r := base.NewRand()                      //生成随机数基
	seckeyContributions := make([]Seckey, n) //私钥切片
	for i := 0; i < n; i++ {
		seckeyContributions[i] = *NewSeckeyFromRand(r.Deri(i)) //以r为基，i为递增量生成n个相关性私钥
	}
	groupSeckey := AggregateSeckeys(seckeyContributions) //对n个私钥聚合，生成组私钥（bls底层算法）
	groupPubkey := NewPubkeyFromSeckey(*groupSeckey)     //从组私钥萃取出组公钥
	t.Log("Group pubkey:", groupPubkey.GetHexString())
	fmt.Printf("end test Aggregation.\n")
}

//把secs切片的每个私钥都转化成big.Int，累加后对曲线域求模，以求模后的big.Int作为参数构建一个新的私钥
func AggregateSeckeysByBigInt(secs []Seckey) *Seckey {
	secret := big.NewInt(0)
	for _, s := range secs {
		secret.Add(secret, s.GetBigInt())
	}
	secret.Mod(secret, curveOrder) //为什么不是每一步都求模，而是全部累加后求模？
	return NewSeckeyFromBigInt(secret)
}

//生成n个衍生随机数私钥，对这n个衍生私钥用bls聚合法和big.Int聚合法生成聚合私钥，比较2个聚合私钥是否一致。
func testAggregateSeckeys(t *testing.T) {
	fmt.Printf("\nbegin testAggregateSeckeys...\n")
	t.Log("begin testAggregateSeckeys")
	n := 100
	r := base.NewRand() //创建随机数基r
	secs := make([]Seckey, n)
	fmt.Printf("begin init 100 sec key...\n")
	for i := 0; i < n; i++ {
		secs[i] = *NewSeckeyFromRand(r.Deri(i)) //以基r和递增变量i生成随机数，创建私钥切片
	}
	fmt.Printf("begin aggr sec key with bigint...\n")
	s1 := AggregateSeckeysByBigInt(secs) //通过int加法和求模生成聚合私钥
	fmt.Printf("begin aggr sec key with bls...\n")
	s2 := AggregateSeckeys(secs) //通过bls底层库生成聚合私钥
	fmt.Printf("sec aggred with int, data=%v.\n", s1.GetHexString())
	fmt.Printf("sec aggred with bls, data=%v.\n", s2.GetHexString())
	if !s1.value.IsEqual(&s2.value) { //比较用简单加法求模生成的聚合私钥和bls底层库生成的聚合私钥是否不同
		t.Errorf("not same int(%v) VS bls(%v).\n", s1.GetHexString(), s2.GetHexString())
	}
	t.Log("end testAggregateSeckeys")
	fmt.Printf("end testAggregateSeckeys.\n")
}

//big.Int处理法：以私钥切片和ID切片恢复出组私钥(私钥切片和ID切片的大小都为门限值k)
func RecoverSeckeyByBigInt(secs []Seckey, ids []ID) *Seckey {
	secret := big.NewInt(0) //组私钥
	k := len(secs)          //取得输出切片的大小，即门限值k
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
	return NewSeckeyFromBigInt(secret) //用big.Int数生成真正的bls私钥
}

//生成n个ID和n个衍生随机数私钥，然后调用bls恢复法和bls.Int恢复法，比较2个恢复的私钥是否一致。
func testRecoverSeckey(t *testing.T) {
	fmt.Printf("\nbegin testRecoverSeckey...\n")
	t.Log("testRecoverSeckey")
	n := 50
	r := base.NewRand() //生成随机数基

	secs := make([]Seckey, n) //私钥切片
	ids := make([]ID, n)      //ID切片
	for i := 0; i < n; i++ {
		ids[i] = *NewIDFromInt64(int64(i + 3))  //生成50个ID
		secs[i] = *NewSeckeyFromRand(r.Deri(i)) //以基r和累加值i，生成50个私钥
	}
	s1 := RecoverSeckey(secs, ids)         //调用bls的私钥恢复函数（门限值取100%）
	s2 := RecoverSeckeyByBigInt(secs, ids) //调用big.Int加法求模的私钥恢复函数
	if !s1.value.IsEqual(&s2.value) {      //检查两种方法恢复的私钥是否相同
		t.Errorf("Mismatch in recovered secret key:\n  %s\n  %s.", s1.GetHexString(), s2.GetHexString())
	}
	fmt.Printf("end testRecoverSeckey.\n")
}

//big.Int处理法：以master key切片和ID生成属于该ID的（签名）私钥
func ShareSeckeyByBigInt(msec []Seckey, id ID) *Seckey {
	secret := big.NewInt(0)
	// degree of polynomial, need k >= 1, i.e. len(msec) >= 2
	k := len(msec) - 1
	// msec = c_0, c_1, ..., c_k
	// evaluate polynomial f(x) with coefficients c0, ..., ck
	secret.Set(msec[k].GetBigInt()) //最后一个master key的big.Int值放到secret
	x := id.GetBigInt()             //取得id的big.Int值
	for j := k - 1; j >= 0; j-- {   //从master key切片的尾部-1往前遍历
		secret.Mul(secret, x) //乘上id的big.Int值，每一遍都需要乘，所以是指数？
		//sec.secret.Mod(&sec.secret, curveOrder)
		secret.Add(secret, msec[j].GetBigInt()) //加法
		secret.Mod(secret, curveOrder)          //曲线域求模
	}
	return NewSeckeyFromBigInt(secret) //生成签名私钥
}

//调用bls生成n个衍生随机数私钥，然后针对一个特定的ID生成bls分享片段和big.Int分享片段，比较2个分享片段是否一致。
func testShareSeckey(t *testing.T) {
	fmt.Printf("\nbegin testShareSeckey...\n")
	t.Log("testShareSeckey")
	n := 100
	msec := make([]Seckey, n)
	r := base.NewRand()
	for i := 0; i < n; i++ {
		msec[i] = *NewSeckeyFromRand(r.Deri(i)) //生成100个随机私钥（bls库初始化函数）
	}
	id := *NewIDFromInt64(123)          //随机生成一个ID
	s1 := ShareSeckeyByBigInt(msec, id) //简单加法分享函数
	s2 := ShareSeckey(msec, id)         //bls库分享函数
	if !s1.value.IsEqual(&s2.value) {   //比较2者是否相同
		t.Errorf("bad sec\n%s\n%s", s1.GetHexString(), s2.GetHexString())
	} else {
		buf := s2.Serialize()
		fmt.Printf("size of seckey = %v.\n", len(buf))
	}
	fmt.Printf("end testShareSeckey.\n")
}

//测试从big.Int生成ID，以及ID的序列化
func testID(t *testing.T) {
	t.Log("testString")
	fmt.Printf("\nbegin test ID...\n")
	b := new(big.Int)
	b.SetString("1234567890abcdef", 16)
	id1 := NewIDFromBigInt(b) //从big.Int生成ID
	if id1 == nil {
		t.Error("NewIDFromBigInt")
	} else {
		buf := id1.Serialize()
		fmt.Printf("id Serialize, len=%v, data=%v.\n", len(buf), buf)
	}
	str := id1.GetHexString()
	fmt.Printf("ID export, len=%v, data=%v.\n", len(str), str)
	{
		var id2 ID
		err := id2.SetHexString(id1.GetHexString()) //测试ID的十六进制导出和导入功能
		if err != nil || !id1.IsEqual(id2) {
			t.Errorf("not same\n%s\n%s", id1.GetHexString(), id2.GetHexString())
		}
	}
	{
		var id2 ID
		err := id2.Deserialize(id1.Serialize()) //测试ID的序列化和反序列化
		if err != nil || !id1.IsEqual(id2) {
			t.Errorf("not same\n%s\n%s", id1.GetHexString(), id2.GetHexString())
		}
	}
	fmt.Printf("end test ID.\n")
}

func test(t *testing.T, c int) {
	fmt.Printf("call test, c =%v.\n", c)
	var tmp []byte
	fmt.Printf("len of empty []byte=%v.\n", len(tmp))
	var ti time.Time
	fmt.Printf("time zero=%v.\n", ti.IsZero())
	var tmp_i int = 456
	fmt.Printf("sizeof(int) =%v.\n", unsafe.Sizeof(tmp_i))
	Init(c)            //初始化bls底层C库
	testID(t)          //测试从big.Int生成ID，以及ID的序列化
	testSeckey(t)      //测试从big.Int生成私钥，以及私钥的序列化
	testPubkey(t)      //测试用衍生随机数生成私钥，从私钥萃取公钥，以及公钥的序列化
	testAggregation(t) //生成n个衍生随机数私钥，对这n个衍生私钥进行聚合生成组私钥，然后萃取出组公钥
	//用big.Int生成私钥，取得公钥和签名。然后对私钥、公钥和签名各复制一份后测试加法后的验证是否正确。
	//同时测试签名的序列化。
	testComparison(t)
	//生成n个衍生随机数私钥，对这n个衍生私钥用bls聚合法和big.Int聚合法生成聚合私钥，比较2个聚合私钥是否一致。
	//全量聚合，用于生成组成员签名私钥（对收到的秘密分片聚合）和组公钥（由组成员签名私钥萃取出公钥，然后在组内广播，任何一个成员收到全量公钥后聚合即生成组公钥）
	testAggregateSeckeys(t)
	//生成n个ID和n个衍生随机数私钥，然后调用bls恢复法和bls.Int恢复法，比较2个恢复的私钥是否一致。
	//秘密分享恢复函数，门限值取了100%。
	testRecoverSeckey(t)
	//调用bls生成n个衍生随机数私钥，然后针对一个特定的ID生成bls分享片段和big.Int分享片段，比较2个分享片段是否一致。
	//秘密分享，把自己的秘密分片发送给组内不同的成员（对不同成员生成不同的秘密分片）
	testShareSeckey(t)
}


func TestIDStringConvert(t *testing.T){
	Init(1)
	str := "QmWJZdSV23by4xzYSz8SEmcfdo38N27WgxSefoy179pnoK"
	id := NewIDFromString(str)
	s:= id.String()
	fmt.Printf("id str:%s\n",s)
	fmt.Printf("id str compare resylt:%t\n",str==s)
}
func TestMain1(t *testing.T) {
	fmt.Printf("begin TestMain...\n")
	t.Logf("GetMaxOpUnitSize() = %d\n", bls.GetMaxOpUnitSize())
	t.Log("CurveFp254BNb")
	fmt.Printf("\ncall test with curve(CurveFp254BNb)=%v...\n", bls.CurveFp254BNb)
	test(t, bls.CurveFp254BNb)
	if bls.GetMaxOpUnitSize() == 6 {
		t.Log("CurveFp382_1")
		fmt.Printf("\ncall test with curve(CurveFp382_1)=%v...\n", bls.CurveFp382_1)
		test(t, bls.CurveFp382_1)
		t.Log("CurveFp382_2")
		fmt.Printf("\ncall test with curve(CurveFp382_2)=%v...\n", bls.CurveFp382_2)
		test(t, bls.CurveFp382_2)
	}
	return
}

func TestID_Deserialize(t *testing.T) {
	Init(1)
	s := "abc"
	id1 := DeserializeId([]byte(s))
	id2 := NewIDFromString(s)
	t.Log(id1.GetHexString(), id2.GetHexString(), id1.IsEqual(*id2))

	t.Log([]byte(s))
	t.Log(id1.Serialize(), id2.Serialize())
	t.Log(id1.String(), id2.String())

	b := id2.Serialize()
	id3 := DeserializeId(b)
	t.Log(id3.GetHexString())
}

func TestNewIDFromString(t *testing.T) {
	Init(1)
	id := NewIDFromString("0x123abc")
	t.Log(id.String(), ",===", id.GetHexString())

	//bi := new(big.Int).SetBytes([]byte("abc"))
	//id2 := NewIDFromBigInt(bi)
	//
	//var id3 ID
	//id3.SetHexString(PREFIX + bi.Text(16))

	//t.Log(id2.GetHexString(), id3.GetHexString())
	id2 := NewIDFromInt(23)
	t.Log(id2.String(), id2.GetHexString())

}
