package groupsig

import (
	"consensus/bls"
	"consensus/rand"
	"fmt"
	"math/big"
	"testing"
	"time"
)

type Expect struct {
	bitLen int
	ok     []byte
}

//公钥测试函数
func testPubkey(t *testing.T) {
	t.Log("testPubkey")
	r := rand.NewRand() //生成随机数

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
}

//测试公钥，私钥，签名的聚合（相加）功能
func testComparison(t *testing.T) {
	t.Log("testComparison")
	var b = new(big.Int)
	b.SetString("16798108731015832284940804142231733909759579603404752749028378864165570215948", 10)
	sec := NewSeckeyFromBigInt(b) //从big.Int（固定的常量）生成原始私钥
	t.Log("sec.Hex: ", sec.GetHexString())
	t.Log("sec.DecimalString: ", sec.GetDecimalString())

	// Add Seckeys
	sum := AggregateSeckeys([]Seckey{*sec, *sec}) //同一个原始私钥相加，生成聚合私钥（bls底层算法）
	if sum == nil {
		t.Log("AggregateSeckeys")
	}

	// Pubkey
	pub := NewPubkeyFromSeckey(*sec) //从原始私钥萃取出公钥
	if pub == nil {
		t.Log("NewPubkeyFromSeckey")
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
}

//测试私钥
func testSeckey(t *testing.T) {
	t.Log("testSeckey")
	s := "401035055535747319451436327113007154621327258807739504261475863403006987855"
	var b = new(big.Int)
	b.SetString(s, 10)
	sec := NewSeckeyFromBigInt(b) //以固定的字符串常量构建私钥
	{
		var sec2 Seckey
		err := sec2.SetHexString(sec.GetHexString()) //测试私钥的十六进制字符串导出
		if err != nil || !sec.IsEqual(sec2) {        //检查字符串导入生成的私钥是否和之前的私钥相同
			t.Error("bad SetHexString")
		}
	}
	{
		var sec2 Seckey
		err := sec2.SetDecimalString(sec.GetDecimalString()) //测试私钥的十进制字符串导出
		if err != nil || !sec.IsEqual(sec2) {                //检查字符串导入生成的私钥是否和之前的私钥相同
			t.Error("bad DecimalString")
		}
		if s != sec.GetDecimalString() {
			t.Error("bad GetDecimalString")
		}
	}
	{
		var sec2 Seckey
		err := sec2.Deserialize(sec.Serialize()) //测试私钥的序列化
		if err != nil || !sec.IsEqual(sec2) {    //检查反序列化生成的私钥是否和之前的私钥相同
			t.Error("bad Serialize")
		}
	}
}

func testAggregation(t *testing.T) {
	t.Log("testAggregation")
	//    m := 5
	n := 3
	//    groupPubkeys := make([]Pubkey, m)
	r := rand.NewRand()                      //生成随机数基
	seckeyContributions := make([]Seckey, n) //私钥切片
	for i := 0; i < n; i++ {
		seckeyContributions[i] = *NewSeckeyFromRand(r.Deri(i)) //以r为基，i为递增量生成n个相关性私钥
	}
	groupSeckey := AggregateSeckeys(seckeyContributions) //对n个私钥聚合，生成组私钥（bls底层算法）
	groupPubkey := NewPubkeyFromSeckey(*groupSeckey)     //从组私钥萃取出组公钥
	t.Log("Group pubkey:", groupPubkey.GetHexString())
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

func testAggregateSeckeys(t *testing.T) {
	t.Log("testAggregateSeckeys")
	n := 100
	r := rand.NewRand() //创建随机数基r
	secs := make([]Seckey, n)
	// init secs
	for i := 0; i < n; i++ {
		secs[i] = *NewSeckeyFromRand(r.Deri(i)) //以基r和递增变量i生成随机数，创建私钥切片
	}
	s1 := AggregateSeckeysByBigInt(secs) //通过int加法和求模生成聚合私钥
	s2 := AggregateSeckeys(secs)         //通过bls底层库生成聚合私钥
	if !s1.value.IsEqual(&s2.value) {    //比较用简单加法求模生成的聚合私钥和bls底层库生成的聚合私钥是否不同
		t.Errorf("not same %s %s\n", s1.GetHexString(), s2.GetHexString())
	}
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

func testRecoverSeckey(t *testing.T) {
	t.Log("testRecoverSeckey")
	n := 50
	r := rand.NewRand() //生成随机数基

	secs := make([]Seckey, n) //私钥切片
	ids := make([]ID, n)      //ID切片
	for i := 0; i < n; i++ {
		ids[i] = *NewIDFromInt64(int64(i + 3))  //生成50个ID
		secs[i] = *NewSeckeyFromRand(r.Deri(i)) //以基r和累加值i，生成50个私钥
	}
	s1 := RecoverSeckey(secs, ids)         //调用bls的私钥恢复函数
	s2 := RecoverSeckeyByBigInt(secs, ids) //调用big.Int加法求模的私钥恢复函数
	if !s1.value.IsEqual(&s2.value) {      //检查两种方法恢复的私钥是否相同
		t.Errorf("Mismatch in recovered secret key:\n  %s\n  %s.", s1.GetHexString(), s2.GetHexString())
	}
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

//测试用bls生成的私钥分片和big.Int法生成的私钥分片是否相同
func testShareSeckey(t *testing.T) {
	t.Log("testShareSeckey")
	n := 100
	msec := make([]Seckey, n)
	r := rand.NewRand()
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
}

//测试ID的生成，序列化和反序列化
func testID(t *testing.T) {
	t.Log("testString")
	b := new(big.Int)
	b.SetString("1234567890abcdef", 16)
	id1 := NewIDFromBigInt(b) //从big.Int生成ID
	if id1 == nil {
		t.Error("NewIDFromBigInt")
	} else {
		buf := id1.Serialize()
		fmt.Printf("size of id = %v.\n", len(buf))
	}
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
}

func test(t *testing.T, c int) {
	fmt.Printf("call test, c =%v.\n", c)
	var tmp []byte
	fmt.Printf("len of empty []byte=%v.\n", len(tmp))
	var ti time.Time
	fmt.Printf("time zero=%v.\n", ti.IsZero())
	Init(c)
	testID(t)
	testSeckey(t)
	testPubkey(t)
	testAggregation(t)
	testComparison(t)
	testAggregateSeckeys(t)
	testRecoverSeckey(t)
	testShareSeckey(t)
}

func TestMain(t *testing.T) {
	t.Logf("GetMaxOpUnitSize() = %d\n", bls.GetMaxOpUnitSize())
	t.Log("CurveFp254BNb")
	test(t, bls.CurveFp254BNb)
	if bls.GetMaxOpUnitSize() == 6 {
		t.Log("CurveFp382_1")
		test(t, bls.CurveFp382_1)
		t.Log("CurveFp382_2")
		test(t, bls.CurveFp382_2)
	}
}
