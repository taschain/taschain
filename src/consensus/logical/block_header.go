package logical

import (
	"common"
	"consensus/groupsig"
)

//数据签名结构
type SignData struct {
	DataHash   common.Hash        //哈希值
	DataSign   groupsig.Signature //签名
	SignMember groupsig.ID        //用户ID或组ID，看消息类型
}

func (sd SignData) GetID() groupsig.ID {
	return sd.SignMember
}

//用pk验证签名，验证通过返回true，否则false。
func (sd SignData) VerifySign(pk groupsig.Pubkey) bool {
	b := pk.IsValid()
	if b {
		b = groupsig.VerifySig(pk, sd.DataHash.Bytes(), sd.DataSign)
	}
	return b
}

func (sd SignData) HasSign() bool {
	return sd.DataSign.IsValid() && sd.SignMember.IsValid()
}