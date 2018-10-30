package base

import (
	"testing"
	"sync/atomic"
	"regexp"
	"fmt"
	"strings"
	"math/big"
)

/*
**  Creator: pxf
**  Date: 2018/9/6 下午1:39
**  Description: 
*/

type something struct {
	ptr atomic.Value
}

func (st *something) setPrt(v *int) {
	//atomic.StorePointer(&st.ptr, unsafe.Pointer(&v))
	st.ptr.Store(v)
}

func (st *something) getPrt() *int {
	//return (*int)(atomic.LoadPointer(&st.ptr))
	return st.ptr.Load().(*int)
}

func TestAtomicPtr(t *testing.T) {
	sth := &something{}
	a := 100
	sth.setPrt(&a)
	l := sth.getPrt()
	t.Log(sth.ptr, *l)

	sth.setPrt(nil)
	l = sth.getPrt()
	if l != nil {
		t.Log(*sth.getPrt())
	} else {
		t.Log("nil")
	}
}

func TestRegex(t *testing.T) {
	prefix := "GetIDPrefix"
	data := prefix + "()"
	re2, _ := regexp.Compile(prefix + "\\((.*?)\\)")

	//FindSubmatch查找子匹配项
	sub := re2.FindSubmatch([]byte(data))
	//第一个匹配的是全部元素
	fmt.Println(string(sub[0]))
	//第二个匹配的是第一个()里面的
	fmt.Println(string(sub[1]))

	s := strings.Replace(data, data, string(sub[1])+".ShortS()", 1)
	fmt.Printf(s)

}

func TestVRF_prove(t *testing.T) {
	//total := new(big.Int)
	pk, sk, _ := VRF_GenerateKey(nil)
	for i := 0; i < 1000000000; i ++ {
		pi, _ := VRF_prove(pk, sk, NewRand().Bytes())
		bi := new(big.Int).SetBytes(pi)
		if bi.Sign() < 0 {
			fmt.Errorf("error bi %d", bi )
			break
		}
		//total = total.Add(total, bi)
		//if total.Sign() < 0 {
		//	fmt.Errorf("error total %d", total)
		//	break
		//}
		if i%10000 == 0 {
			fmt.Printf("%d total: %d\n", i, bi)
		}
	}
}
