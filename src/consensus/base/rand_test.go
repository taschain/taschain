package base

import (
	"testing"
	"math/big"
	"math/rand"
	"unsafe"
	"sync/atomic"
	"log"
	"math"
)

/*
**  Creator: pxf
**  Date: 2018/6/15 下午4:39
**  Description: 
*/

func TestRandSeed(t *testing.T) {
	b := big.NewInt(100000)

	r := NewFromSeed(b.Bytes())
	t.Log(r)

	r = NewFromSeed(b.Bytes())
	t.Log(r)

	r = NewFromSeed(b.Bytes())
	t.Log(r)
}

func TestMathRand(t *testing.T) {
	s := rand.NewSource(1000000)
	r := rand.New(s)


	t.Log(r.Uint64())
	t.Log(r.Uint64())
	t.Log(r.Uint64())
}

func TestMathRand2(t *testing.T) {
	rand.Seed(1000000)
	t.Log(rand.Uint64())
	t.Log(rand.Uint64())
	t.Log(rand.Uint64())
	t.Log(rand.Uint64())
}

func TestRandSeq(t *testing.T) {
	rand := RandFromBytes([]byte("abc"))
	t.Log(rand.RandomPerm(10, 3))
	t.Log(rand.RandomPerm(10, 3))
	t.Log(rand.RandomPerm(120, 3))
	t.Log(rand.RandomPerm(120, 15))
	t.Log(rand.RandomPerm(120, 15))
	t.Log(rand.RandomPerm(120, 16))
}

func TestAtomic(t *testing.T) {
	var b  = false
	pointer := unsafe.Pointer(&b)
	v := atomic.LoadPointer(&pointer)
	v1 := (*bool)(v)
	log.Println(*v1)

	n := true
	p2 := unsafe.Pointer(&b)
	atomic.StorePointer(&p2, unsafe.Pointer(&n))

	v = atomic.LoadPointer(&p2)
	v2 := (*bool)(v)
	log.Println(*v2)
}

func TestRand_RandomPermUint64(t *testing.T) {
	r := RandFromBytes([]byte("3"))
	t.Log(r.ModuloUint64(math.MaxUint64))
}

type aa interface {
	Func()
}
type parent struct {

}

func (p parent) Func() {
	log.Println("parent")
}

func (p parent) Test()  {
    var a aa
    a = &p
    a.Func()
}

type son1 struct {
	parent
}

func (son1) Func() {
	log.Println("son1")
}

type son2 struct {
	parent
}

func (son2) Func() {
	log.Println("son2")
}

func TestParent(t *testing.T) {
	p := &son1{}
	p.Test()
}