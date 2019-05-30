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

package base

import (
	"fmt"
	"github.com/taschain/taschain/common"
	"math/big"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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
	for i := 0; i < 1000000000; i++ {
		pi, _ := VRF_prove(pk, sk, NewRand().Bytes())
		bi := new(big.Int).SetBytes(pi)
		if bi.Sign() < 0 {
			fmt.Errorf("error bi %d", bi)
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

func TestTimeAdd(t *testing.T) {
	now := time.Now()
	b := now.Add(-time.Second * time.Duration(10))
	t.Log(b)
}

func TestHashEqual(t *testing.T) {
	h := common.BytesToHash([]byte("123"))
	h2 := common.BytesToHash([]byte("123"))
	h3 := common.BytesToHash([]byte("234"))

	t.Log(h == h2, h == h3)
}
