package time

import "testing"
import (
	"github.com/beevik/ntp"
	"time"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2019/4/10 上午11:16
**  Description: 
*/

func TestNTPQuery(t *testing.T) {
	tt, _ := ntp.Time("time.asia.apple.com")
	t.Log(tt, time.Now())

	rsp, _ := ntp.Query("time.pool.aliyun.com")
	t.Log(rsp.Time, rsp.ClockOffset.String(), time.Now(), time.Now().Add(rsp.ClockOffset))

	rsp, err := ntp.Query("cn.pool.ntp.org")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(rsp.Time, rsp.ClockOffset.String(), time.Now(), time.Now().Add(rsp.ClockOffset))

	for i:=1; i < 8;i++ {
		t.Log(ntp.Time(fmt.Sprintf("ntp%v.aliyun.com", i)))
	}
}

func TestSync(t *testing.T) {
	InitTimeSync()
	time.Sleep(time.Hour)
}