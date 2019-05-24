package concurrent

import (
	"sync"

	"container/list"
)

type Queue struct {
	data *list.List
	lock sync.Mutex
	max  int
}

func NewQueue(max int) *Queue {
	return &Queue{
		data: list.New(),
		lock: sync.Mutex{},
		max:  max,
	}

}

func (q *Queue) Push(data interface{}) bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.data.Len() == q.max {
		return false
	}

	q.data.PushBack(data)
	return true
}

func (q *Queue) Pop() interface{} {
	q.lock.Lock()
	defer q.lock.Unlock()

	data := q.data.Front()
	q.data.Remove(data)

	return data
}
