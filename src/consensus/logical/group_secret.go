package logical

import (
	"consensus/groupsig"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/6/15 下午2:23
**  Description: 
*/

type GroupSecret struct {
	SecretSign   []byte      //秘密签名
	EffectHeight uint64      //生效高度
	DataHash     common.Hash //签名的数据hash
}

func NewGroupSecret(sign groupsig.Signature, height uint64, hash common.Hash) *GroupSecret {
	return &GroupSecret{
		SecretSign:   sign.Serialize(),
		EffectHeight: height,
		DataHash:     hash,
	}
}