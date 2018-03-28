package groupsig

// types

//POP即私钥对公钥的签名
type Pop Signature

//Proof-of-Possesion：拥有证明

//POP生成，用私钥对公钥的序列化值签名。
func GeneratePop(sec Seckey, pub Pubkey) Pop {
	return Pop(Sign(sec, pub.Serialize()))
}

//POP验证，用公钥对POP进行验证，确认POP是由该公钥对应的私钥生成的。
func VerifyPop(pub Pubkey, pop Pop) bool {
	return VerifySig(pub, pub.Serialize(), Signature(pop))
}
