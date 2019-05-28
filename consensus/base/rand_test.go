package base

import (
	"testing"
)

/*
**  Creator: pxf
**  Date: 2018/6/15 下午4:39
**  Description:
 */

func TestRandFromString(t *testing.T) {
	r := RandFromString("123")
	t.Log(r)
}

func TestNewRand(t *testing.T) {
	r := NewRand()
	t.Log(r.Bytes())
}

func TestRand_ModuloUint64(t *testing.T) {
	r := RandFromBytes([]byte("3456789"))

	for i := 0; i < 10000; i ++ {
		v := r.ModuloUint64(100)
		if v >= 100 {
			t.Fatalf("modulo uint64 error")
		}
	}
}

func BenchmarkRand_ModuloUint64(b *testing.B) {
	r := RandFromBytes([]byte("3456789"))
	for i := 0; i < b.N; i++ {
		r.ModuloUint64(100)
	}
}
