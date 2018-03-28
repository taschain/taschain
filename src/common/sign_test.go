package common

import (
	"crypto/sha1"
	"fmt"
	"testing"

	"golang.org/x/crypto/sha3"
)

func TestPrivateKey(test *testing.T) {
	fmt.Printf("begin TestPrivateKey...\n")
	pk := GenerateKey()
	buf := pk.ToBytes()
	fmt.Printf("byte buf len of private key = %v.\n", len(buf))
}

func TestPublickKey(test *testing.T) {
	fmt.Printf("begin TestPublicKey...\n")
	pri_k := GenerateKey()
	pub_k := pri_k.GetPubKey()
	buf := pub_k.ToBytes()
	fmt.Printf("byte buf len of public key = %v.\n", len(buf))
}

func TestSign(test *testing.T) {
	plain_txt := "My name is thiefox."
	buf := []byte(plain_txt)
	sha1_hash := sha1.Sum(buf)
	sha3_hash := sha3.Sum256(buf)
	fmt.Printf("hash test, sha1_len=%v, sha3_len=%v.\n", len(sha1_hash), len(sha3_hash))
	pri_k := GenerateKey()
	pub_k := pri_k.GetPubKey()

	pub_buf := pub_k.ToBytes() //测试公钥到字节切片的转换
	pub_k = BytesToPublicKey(pub_buf)

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
}
