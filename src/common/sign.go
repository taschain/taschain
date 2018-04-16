package common

import (
	"math/big"
)

type Sign struct {
	r big.Int
	s big.Int
}

//签名构造函数
func (s *Sign) Set(_r, _s *big.Int) {
	s.r = *_r
	s.s = *_s
}

//检查签名是否有效
func (s Sign) Valid() bool {
	return s.r.BitLen() != 0 && s.s.BitLen() != 0
}

//获取R值
func (s Sign) GetR() big.Int {
	return s.r
}

//获取S值
func (s Sign) GetS() big.Int {
	return s.s
}

func (s Sign) Bytes() []byte {
	rb := s.r.Bytes()
	sb := s.s.Bytes()
	r := make([]byte, len(rb)+len(sb))
	copy(r, rb)
	copy(r[len(rb):], sb)
	return r
}

func BytesToSign(b []byte) *Sign {
	var r, s big.Int
	br := b[:32]
	r = *r.SetBytes(br)

	sr := b[32:]
	s = *s.SetBytes(sr)
	return &Sign{r, s}
}
