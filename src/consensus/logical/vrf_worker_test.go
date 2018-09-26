package logical

import (
	"testing"
	"math/big"
)

/*
**  Creator: pxf
**  Date: 2018/9/21 下午4:29
**  Description: 
*/

func TestBigInt(t *testing.T) {
	t.Log(max256, max256.String(), max256.FloatString(10))
}

func TestBigIntDiv(t *testing.T) {
	a, _ := new(big.Int).SetString("ffffffffffff", 16)
	b := new(big.Rat).SetInt(a)
	v := new(big.Rat).Quo(b, max256)
	t.Log(a, b, max256, v)
	t.Log(v.FloatString(5))

	a1 := new(big.Rat).SetInt64(10)
	a2 := new(big.Rat).SetInt64(30)
	v2 := a1.Quo(a1, a2)
	t.Log(v2.Float64())
	t.Log( v2.FloatString(5))
}