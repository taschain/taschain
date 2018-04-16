package common

import (
	"crypto/sha1"
	"fmt"
	"testing"

	"golang.org/x/crypto/sha3"
)

func TestPrivateKey(test *testing.T) {
	fmt.Printf("\nbegin TestPrivateKey...\n")
	sk := GenerateKey("")
	str := sk.GetHexString()
	fmt.Printf("sec key export, len=%v, data=%v.\n", len(str), str)
	new_sk := HexStringToSecKey(str)
	new_str := new_sk.GetHexString()
	fmt.Printf("import sec key and export again, len=%v, data=%v.\n", len(new_str), new_str)
	fmt.Printf("end TestPrivateKey.\n")
}

func TestPublickKey(test *testing.T) {
	fmt.Printf("\nbegin TestPublicKey...\n")
	sk := GenerateKey("")
	pk := sk.GetPubKey()
	//buf := pub_k.toBytes()
	//fmt.Printf("byte buf len of public key = %v.\n", len(buf))
	str := pk.GetHexString()
	fmt.Printf("pub key export, len=%v, data=%v.\n", len(str), str)
	new_pk := HexStringToPubKey(str)
	new_str := new_pk.GetHexString()
	fmt.Printf("import pub key and export again, len=%v, data=%v.\n", len(new_str), new_str)

	fmt.Printf("\nbegin test address...\n")
	a := pk.GetAddress()
	str = a.GetHexString()
	fmt.Printf("address export, len=%v, data=%v.\n", len(str), str)
	new_a := HexStringToAddress(str)
	new_str = new_a.GetHexString()
	fmt.Printf("import address and export again, len=%v, data=%v.\n", len(new_str), new_str)

	fmt.Printf("end TestPublicKey.\n")
}

func TestSign(test *testing.T) {
	fmt.Printf("\nbegin TestSign...\n")
	plain_txt := "My name is thiefox."
	buf := []byte(plain_txt)
	sha1_hash := sha1.Sum(buf)
	sha3_hash := sha3.Sum256(buf)
	fmt.Printf("hash test, sha1_len=%v, sha3_len=%v.\n", len(sha1_hash), len(sha3_hash))
	pri_k := GenerateKey("")
	pub_k := pri_k.GetPubKey()

	pub_buf := pub_k.ToBytes() //测试公钥到字节切片的转换
	pub_k = *BytesToPublicKey(pub_buf)

	var sha_buf []byte
	copy(sha_buf, sha1_hash[:])
	sha1_si := pri_k.Sign(sha_buf) //私钥签名
	{
		buf_r := sha1_si.r.Bytes()
		buf_s := sha1_si.s.Bytes()
		fmt.Printf("sha1 sign, r len = %v, s len = %v.\n", len(buf_r), len(buf_s))
	}
	success := pub_k.Verify(sha_buf, &sha1_si) //公钥验证
	fmt.Printf("sha1 sign verify result=%v.\n", success)

	copy(sha_buf, sha3_hash[:])
	sha3_si := pri_k.Sign(sha_buf)
	{
		buf_r := sha3_si.r.Bytes()
		buf_s := sha3_si.s.Bytes()
		fmt.Printf("sha3 sign, r len = %v, s len = %v.\n", len(buf_r), len(buf_s))
	}
	success = pub_k.Verify(sha_buf, &sha3_si)
	fmt.Printf("sha3 sign verify result=%v.\n", success)
	fmt.Printf("end TestSign.\n")
}

func TestSignBytes(test *testing.T) {
	plain_txt := "Sign bytes convert."
	buf := []byte(plain_txt)

	pri_k := GenerateKey("")

	sha1_hash := sha1.Sum(buf)
	var sha_buf []byte
	copy(sha_buf, sha1_hash[:])
	sign := pri_k.Sign(sha_buf) //私钥签名

	sign_bytes := sign.Bytes()
	sign_r := BytesToSign(sign_bytes)

	sign_r.Bytes()

}
