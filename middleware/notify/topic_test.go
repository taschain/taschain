package notify

import (
	"fmt"
	"testing"
)

//hello world2
//hello world
func TestTopic_Subscribe(t *testing.T) {
	topic := &Topic{
		Id: "test",
	}

	topic.Subscribe(handler1)
	topic.Subscribe(handler2)
	topic.Handle(&DummyMessage{})
}

//hello world2
func TestTopic_UnSubscribe0(t *testing.T) {
	topic := &Topic{
		Id: "test",
	}

	topic.Subscribe(handler1)
	topic.Subscribe(handler2)

	topic.UnSubscribe(handler1)
	topic.Handle(&DummyMessage{})
}

//hello world3
//hello world
func TestTopic_UnSubscribe1(t *testing.T) {
	topic := &Topic{
		Id: "test",
	}

	topic.Subscribe(handler1)
	topic.Subscribe(handler2)
	topic.Subscribe(handler3)

	topic.UnSubscribe(handler2)
	topic.Handle(&DummyMessage{})
}

// hello world
// hello world2
func TestTopic_UnSubscribe2(t *testing.T) {
	topic := &Topic{
		Id: "test",
	}

	topic.Subscribe(handler1)
	topic.Subscribe(handler2)
	topic.Subscribe(handler3)

	topic.UnSubscribe(handler3)
	topic.Handle(&DummyMessage{})
}

func handler1(message Message) {
	fmt.Println("hello world")
}

func handler2(message Message) {
	fmt.Println("hello world2")
}

func handler3(message Message) {
	fmt.Println("hello world3")
}
