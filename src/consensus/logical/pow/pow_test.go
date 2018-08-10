package pow

import (
	"testing"
	"consensus/base"
	"time"
	"math/big"
	"log"
	"fmt"
	"os"
	"taslog"
	rand2 "math/rand"
)

/*
**  Creator: pxf
**  Date: 2018/8/2 下午2:51
**  Description:
*/

func TestHash(t *testing.T) {
	rand := base.NewRand().Deri(1)
	t.Log(rand.GetHexString())
	bi := new(big.Int).SetBytes(rand.Bytes())

	i := 0
	go func() {
		for {
			bi.Add(bi, new(big.Int).SetUint64(uint64(i)))
			base.Data2CommonHash(base.Data2CommonHash(bi.Bytes()).Bytes())
			i++
		}
	}()
	go func() {
		cnt := 0
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			cnt++
			t.Log(i, i/cnt)
		}
	}()


	time.Sleep(time.Second*30)
}

func TestSetString(t *testing.T) {
	s := "00000111"
	bi, _ := new(big.Int).SetString(s, 2)
	log.Println(bi.Text(2))
	log.Println(bi.Text(16))
}

func hash2NZero(zeroNum uint) (uint64, float64) {
	rand := base.NewRand().Deri(1)
	bi := new(big.Int).SetBytes(rand.Bytes())

	s := ""
	zero := uint(0)
	for zero < zeroNum  {
		s += "0"
		zero++
	}
	for zero < 256 {
		s += "1"
		zero++
	}
	//log.Println(s, len(s))

	d, _ := new(big.Int).SetString(s, 2)
	//t := d.Text(16)
	//log.Println(t, len(t))
	begin := time.Now()
	nonce := uint64(0)
	for i := uint64(0); i >= 0; i++ {
		rand := rand2.Int63()
		bi.Add(bi, new(big.Int).SetUint64(uint64(rand)))
		h := base.Data2CommonHash(base.Data2CommonHash(bi.Bytes()).Bytes())
		//log.Println(len(h.Big().Text(16)), h.String())
		if h.Big().Cmp(d) <= 0 {
			nonce = i
			break
		}
		i++
	}
	//log.Printf("zero %v, nonce %v, cost %v\n", zeroNum, nonce, time.Since(begin).String())
	return nonce, time.Since(begin).Seconds()
}

func TestCountZero(t *testing.T) {
	averN := uint64(0)
	averC := float64(0)
	num := 200
	z := 7
	for i := 1; i <= num; i++ {
		n, c := hash2NZero(uint(z))
		averN += n
		averC += c
		log.Println("diff", z, "round", i, fmt.Sprintf("(%v,%v)", n, c),"aver nonce", averN/uint64(i), "aver cost", averC/float64(i))
	}
}

func TestBenchCountZero(t *testing.T) {
	z := 1
	for z < 8 {
		averN := uint64(0)
		averC := float64(0)
		num := 1
		for i := 0; i < num; i++ {
			n, c := hash2NZero(uint(z))
			averN += n
			averC += c
		}
		log.Println("diff", z, "aver nonce", averN/uint64(num), "aver cost", averC/float64(num))
		z++
	}
}

func hashEndByTime(seconds int, z int, logger taslog.Logger) {
	averN := uint64(0)
	averC := float64(0)
	begin := time.Now()
	for i := 1; i > 0; i++ {
		n, c := hash2NZero(uint(z))
		averN += n
		averC += c
		s := fmt.Sprintf("zero %v round %v (%v,%v) aver (%v,%v)", z, i, n, c, averN/uint64(i), averC/float64(i))
		log.Println(s)
		logger.Infof("%v,%v,%v,%v,%v,%v", z, i, n, c, averN/uint64(i), averC/float64(i))
		if int(time.Since(begin).Seconds()) > seconds {
			break
		}
	}
}
func tracefile(str_content string) {
	fd, _ := os.OpenFile("result.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	buf := []byte(str_content)
	fd.Write(buf)
	fd.Close()
}

func TestByTime(t *testing.T) {
	logger := taslog.GetLoggerByName("hash_test")
	tMap := make(map[int]int)
	for i := 23; i < 31; i++ {
		tMap[i] = 10
	}
	tMap[23] = 1000
	tMap[24] = 1800
	tMap[25] = 2000
	tMap[26] = 3600
	tMap[27] = 4000
	tMap[28] = 7200
	tMap[29] = 7200
	tMap[30] = 7200
	tMap[31] = 8000
	tMap[32] = 8000

	for i := 23; i < 33; i++ {
		hashEndByTime(10, i, logger)
	}

	taslog.Close()
}