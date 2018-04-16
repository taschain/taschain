package biz

import (
	"consensus/groupsig"
	"consensus/logical"
)

//-----------------------------------------------------回调函数定义-----------------------------------------------------
//根据id获取PUB，用于校验签名
type getPubkeyByIdFn func(id groupsig.ID) (groupsig.Pubkey, error)

//根据组id获取组签名，用于校验签名
type getGroupPubkeyByIdFn func(id groupsig.ID) (groupsig.Pubkey, error)
//-----------------------------------------------------------------------------------------------------------------------

type signValidator struct {

	getPubkey      getPubkeyByIdFn
	getGroupPubkey getGroupPubkeyByIdFn
}

var sv *signValidator


func GetSignValidatorInstance()*signValidator{
	if sv == nil{
		//TODO 写入回调函数实现
		sv = newSignValidator(nil,nil)
	}
	return sv
}

func newSignValidator(getPubkey getPubkeyByIdFn, getGroupPubkey getGroupPubkeyByIdFn,) *signValidator{

	return &signValidator{
		getPubkey:           getPubkey,
		getGroupPubkey:      getGroupPubkey,
	}
}

func (v signValidator) isNodeSignCorrect(sd logical.SignData) bool {
	id := sd.SignMember
	pubkey, e := v.getPubkey(id)
	if e != nil {
		result := sd.VerifySign(pubkey)
		return result
	}
	return false
}

func (v signValidator) isGroupSignCorrect(sd logical.SignData) bool {
	id := sd.SignMember
	pubkey, e := v.getGroupPubkey(id)
	if e != nil {
		result := sd.VerifySign(pubkey)
		return result
	}
	return false
}