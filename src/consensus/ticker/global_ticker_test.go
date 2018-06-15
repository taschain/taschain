package ticker

import (
	"testing"
	"time"
	"log"
)

func handler(str string) RoutineFunc {
	return func() bool {
		log.Printf(str)
		return true
	}
}

func TestGlobalTicker_RegisterRoutine(t *testing.T) {

	ticker := NewGlobalTicker("test")

	time.Sleep(time.Second * 5)

	ticker.RegisterRoutine("name1", handler("name1 exec1"), uint32(2))

	time.Sleep(time.Second * 5)
	ticker.RegisterRoutine("name2", handler("name2 exec1"), uint32(3))
	time.Sleep(time.Second * 5)

	ticker.RegisterRoutine("name3", handler("name3 exec1"), uint32(4))

	ticker.StopTickerRoutine("name1")


	time.Sleep(time.Second * 5)
	ticker.StopTickerRoutine("name3")
	time.Sleep(time.Second * 55)
}
