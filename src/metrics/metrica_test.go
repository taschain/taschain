package metrics

import (
	"testing"
	"time"
)

func init() {
	EnableMetrics(false)
}

func TestEnableMetrics(t *testing.T) {
	for i:=0; i <= 6; i ++ {
		Sleep()
		time.Sleep(time.Second/2)
	}
	ConsoleReport()
}

func TestNewDurationCollector(t *testing.T) {
	dc := NewDurationCollector("test2")
	for i:=0; i <= 30; i ++ {
		dc.Reset()
		time.Sleep(time.Second/2)
		dc.Stop()
	}
	ConsoleReport()
}

func Sleep() {
	defer CollectRunDuration("test")()
	time.Sleep(time.Second)
}