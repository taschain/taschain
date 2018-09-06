// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bn_curve

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ethereum/go-ethereum/common/bitutil"
)

func TestExamplePair(t *testing.T) {
	// This implements the tripartite Diffie-Hellman algorithm from "A One
	// Round Protocol for Tripartite Diffie-Hellman", A. Joux.
	// http://www.springerlink.com/content/cddc57yyva0hburb/fulltext.pdf

	// Each of three parties, a, b and c, generate a private value.
	a, _ := rand.Int(rand.Reader, Order)
	b, _ := rand.Int(rand.Reader, Order)
	c, _ := rand.Int(rand.Reader, Order)

	// Then each party calculates g₁ and g₂ times their private value.
	pa := new(G1).ScalarBaseMult(a)
	qa := new(G2).ScalarBaseMult(a)

	pb := new(G1).ScalarBaseMult(b)
	qb := new(G2).ScalarBaseMult(b)

	pc := new(G1).ScalarBaseMult(c)
	qc := new(G2).ScalarBaseMult(c)

	// Now each party exchanges its public values with the other two and
	// all parties can calculate the shared key.
	k1 := Pair(pb, qc)
	k1.ScalarMult(k1, a)

	k2 := Pair(pc, qa)
	k2.ScalarMult(k2, b)

	k3 := Pair(pa, qb)
	k3.ScalarMult(k3, c)

	// k1, k2 and k3 will all be equal.

	require.Equal(t, k1, k2)
	require.Equal(t, k1, k3)

	require.Equal(t, len(np), 4) //Avoid gometalinter varcheck err on np
}


//同态加密测试
func TestHomoEncrypt(t *testing.T) {
	//------------1.初始化-----------
	//生成A B的公私钥对<s1, P1> <s2, P2>
	s1,_ := randomK(rand.Reader)
	s2,_ := randomK(rand.Reader)
	P1, P2 := &G1{}, &G1{}
	P1.ScalarBaseMult(s1)
	P2.ScalarBaseMult(s2)

	//------------2.加密-----------
	//得到随机M
	M := make([]byte, 32)
	rand.Read(M)
	t.Log("M:", M)

	//加密M， 得到<C1, R1>
	r1,_ := randomK(rand.Reader)

	R1 := &G1{}
	R1.ScalarBaseMult(r1)

	S1 := &G1{}
	S1.ScalarMult(P1, r1)
	C1 := make([]byte, 32)
	bitutil.XORBytes(C1, M, S1.Marshal())

	//再次加密<C1, R1>， 得到<C2, R1, R2>
	r2,_ := randomK(rand.Reader)
	R2 := &G1{}
	R2.ScalarBaseMult(r2)

	S2 := &G1{}
	S2.ScalarMult(P2, r2)
	C2 := make([]byte, 32)
	bitutil.XORBytes(C2, C1, S2.Marshal())

	//------------3.解密-----------
	SS1, SS2 := &G1{}, &G1{}
	SS1 = R1.ScalarMult(R1, s1)  //A计算出
	SS2 = R2.ScalarMult(R2, s2)  //B计算出

	//A 先解密，B后解密
	D1, D2 := make([]byte, 32), make([]byte, 32)
	bitutil.XORBytes(D1, C2, SS1.Marshal())      //A计算出D1
	bitutil.XORBytes(D2, D1, SS2.Marshal())      //B计算出D2
	t.Log("D2:", D2)

	//B先解密，A后解密
	DD1, DD2 := make([]byte, 32), make([]byte, 32)
	bitutil.XORBytes(DD1, C2, SS2.Marshal())
	bitutil.XORBytes(DD2, DD1, SS1.Marshal())
	t.Log("DD2:", DD2)
}

func TestCpu(t *testing.T) {
	t.Log("hasBMI2:", hasBMI2)
}

