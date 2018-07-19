package common

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"golang.org/x/crypto/sha3"
	"testing"
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

	var sha_buf []byte
	sha_buf = make([]byte, 32)
	copy(sha_buf, sha3_hash[:32])

	sha3_si, err := pri_k.Sign(sha_buf) //私钥签名
	if err != nil {
		fmt.Errorf("sign error")
		return
	}

	pub_buf := pub_k.ToBytes() //测试公钥到字节切片的转换
	pub_k = *BytesToPublicKey(pub_buf)
	sig := sha3_si[:len(sha3_si)-1] // remove recovery id

	success := pub_k.Verify(sha_buf, sig) //公钥验证
	fmt.Printf("sha1 sign verify result=%v.\n", success)
}

func TestEncryptDecrypt(t *testing.T) {
	fmt.Printf("\nbegin TestEncryptDecrypt...\n")
	sk1 := GenerateKey("")
	pk1 := sk1.GetPubKey()

	sk2 := GenerateKey("")

	message := []byte("Hello, world.")
	ct, err := Encrypt(rand.Reader, &pk1, message)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	pt, err := sk1.Decrypt(rand.Reader, ct)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	fmt.Println(message)
	fmt.Println(ct)
	fmt.Println(pt)

	if !bytes.Equal(pt, message) {
		fmt.Println("ecies: plaintext doesn't match message")
		t.FailNow()
	}

	_, err = sk2.Decrypt(rand.Reader, ct)
	if err == nil {
		fmt.Println("ecies: encryption should not have succeeded")
		t.FailNow()
	}
	fmt.Printf("end TestEncryptDecrypt.\n")
}

func TestHash(test *testing.T) {
	h1 := Hash{1, 2, 3, 4}
	h2 := Hash{1, 2, 3, 4}
	fmt.Printf("%v", h1 == h2)
}
