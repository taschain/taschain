package ticker

import (
	"testing"
	"time"
	"log"
)

func TestGlobalTicker_RegisterRoutine(t *testing.T) {

	ticker := NewGlobalTicker("test", 2)

	time.Sleep(time.Second * 5)

	ticker.RegisterRoutine("name1", func() {
		log.Println("name1 exect1")
	}, true)

	time.Sleep(time.Second * 5)

	ticker.RegisterRoutine("name2", func() {
		log.Println("name2 aafffddd")
	}, false)

	time.Sleep(time.Second * 5)


	ticker.RegisterRoutine("name3", func() {
		log.Println("name3 22333")
	}, false)


	time.Sleep(time.Second * 5)

	ticker.RemoveRoutine("name1")

	ticker.RemoveRoutine("name3")

	time.Sleep(time.Second * 5)
	time.Sleep(time.Second * 5)
}