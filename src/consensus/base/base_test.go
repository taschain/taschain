package base

import (
	"testing"
	"sync/atomic"
)

/*
**  Creator: pxf
**  Date: 2018/9/6 下午1:39
**  Description: 
*/

type something struct {
	ptr atomic.Value
}

func (st *something) setPrt(v *int)  {
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

