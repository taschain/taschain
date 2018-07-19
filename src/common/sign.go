package common

import (
	"math/big"
	//"vm/ecdsa-go"
	"vm/common"
)

type Sign struct {
	r big.Int
	s big.Int
}

//数据签名结构 for message casting
type SignData struct {
	DataHash common.Hash //哈希值
	DataSign Sign        //签名
	Id       string      //用户ID
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
