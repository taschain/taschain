package notify

import (
	"testing"
	"fmt"
	"time"
	"sync"
)

func TestBus_Publish(t *testing.T) {
	ch := make(chan int, 1)
	go produce(ch)
	go consumer(ch)
	go consumer2(ch)
	time.Sleep(1 * time.Second)
	bus:= NewBus()
	bus.Publish("test", &DummyMessage{})

}

func TestNewBus(t *testing.T) {
	condition := false // 条件不满足
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	// 让例程去创造条件
	go func() {
		mu.Lock()
		condition = true // 更改条件
		cond.Signal()    // 发送通知：条件已经满足
		mu.Unlock()
	}()
	mu.Lock()
	// 检查条件是否满足，避免虚假通知，同时避免 Signal 提前于 Wait 执行。
	for !condition {
		// 等待条件满足的通知，如果收到虚假通知，则循环继续等待。
		cond.Wait() // 等待时 mu 处于解锁状态，唤醒时重新锁定。
	}
	fmt.Println("条件满足，开始后续动作...")
	mu.Unlock()
	fmt.Println("条件满足2，开始后续动作...")
}

func produce(ch chan<- int) {
	for i := 0; i < 20; i++ {
		ch <- i
		fmt.Println("Send:", i)
	}
}

func consumer(ch <-chan int) {
	for i := 0; i < 10; i++ {

		fmt.Println("Receive:", <-ch)
	}
}

func consumer2(ch <-chan int) {
	for i := 0; i < 10; i++ {

		fmt.Println("Receive2:", <-ch)
	}
}
