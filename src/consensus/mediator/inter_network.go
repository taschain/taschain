package mediator

import (
	"common"
	"consensus/groupsig"
)

///////////////////////////////////////////////////////////////////////////////
//消息签名函数
func SignMessage(msg common.Hash, sk groupsig.Seckey) (sign []byte) {
	s := groupsig.Sign(sk, msg.Bytes())
	sign = s.Serialize()
	return
}

//消息验签函数
func VerifyMessage(msg common.Hash, sign []byte, pk groupsig.Pubkey) bool {
	var s groupsig.Signature
	if s.Deserialize(sign) != nil {
		return false
	}
	b := groupsig.VerifySig(pk, msg.Bytes(), s)
	return b
}
