package time

import "testing"
import (
	"fmt"
	"github.com/beevik/ntp"
	"github.com/taschain/taschain/common"
	"time"
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

	for i := 1; i < 8; i++ {
		t.Log(ntp.Time(fmt.Sprintf("ntp%v.aliyun.com", i)))
	}
}

func TestSync(t *testing.T) {
	InitTimeSync()
	time.Sleep(time.Hour)
}

func TestUTCAndLocal(t *testing.T) {
	InitTimeSync()
	now := time.Now()

	utc := time.Now().UTC()
	t.Log(now, utc)

	t.Log(utc.After(now), utc.Local().After(now), utc.Before(now))
}

func TestTimeMarshal(t *testing.T) {
	now := time.Now().UTC()
	bs, _ := now.MarshalBinary()
	t.Log(bs, len(bs))
}

func TestTimeUnixSec(t *testing.T) {
	now := time.Now()
	t.Log(now.Unix(), now.UTC().Unix())

	t.Log(time.Unix(now.Unix(), 0))
	t.Log(time.Unix(int64(common.MaxUint32), 0))
}

func TestTimeStampString(t *testing.T) {
	fmt.Printf("ts:%v", TimeToTimeStamp(time.Now()))
}
