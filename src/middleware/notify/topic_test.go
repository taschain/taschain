package notify

import (
	"testing"
	"fmt"
)

//hello world2
//hello world
func TestTopic_Subscribe(t *testing.T) {
	topic := &Topic{
		Id: "test",
	}

	h1 := &handler1{}
	topic.Subscribe(h1)
	h2 := &handler2{}
	topic.Subscribe(h2)
	topic.Handle(&DummyMessage{})
}

//hello world2
func TestTopic_UnSubscribe0(t *testing.T) {
	topic := &Topic{
		Id: "test",
	}

	h1 := &handler1{}
	topic.Subscribe(h1)
	h2 := &handler2{}
	topic.Subscribe(h2)

	topic.UnSubscribe(h1)
	topic.Handle(&DummyMessage{})
}

//hello world3
//hello world
func TestTopic_UnSubscribe1(t *testing.T) {
	topic := &Topic{
		Id: "test",
	}

	h1 := &handler1{}
	topic.Subscribe(h1)
	h2 := &handler2{}
	topic.Subscribe(h2)
	h3 := &handler3{}
	topic.Subscribe(h3)

	topic.UnSubscribe(h2)
	topic.Handle(&DummyMessage{})
}

// hello world
// hello world2
func TestTopic_UnSubscribe2(t *testing.T) {
	topic := &Topic{
		Id: "test",
	}

	h1 := &handler1{}
	topic.Subscribe(h1)
	h2 := &handler2{}
	topic.Subscribe(h2)
	h3 := &handler3{}
	topic.Subscribe(h3)

	topic.UnSubscribe(h3)
	topic.Handle(&DummyMessage{})
}

type handler1 struct {
}

func (h *handler1) Handle(message Message) {
	fmt.Println("hello world")
}

type handler2 struct {
}

func (h *handler2) Handle(message Message) {
	fmt.Println("hello world2")
}

type handler3 struct {
}

func (h *handler3) Handle(message Message) {
	fmt.Println("hello world3")
}
