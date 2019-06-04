package network

import (
	"bytes"
	"container/list"
	"sync"
)

type BufferPoolItem struct {
	buffers *list.List
	size    int
	max     int
	inuse   int
}

func newBufferPoolItem(size int, max int) *BufferPoolItem {

	item := &BufferPoolItem{
		buffers: list.New(), size: size, max: max,
	}

	return item
}

func (poolItem *BufferPoolItem) getBuffer() *bytes.Buffer {

	if poolItem.buffers.Len() > 0 {
		e := poolItem.buffers.Front()
		buf := e.Value.(*bytes.Buffer)
		poolItem.buffers.Remove(poolItem.buffers.Front())
		buf.Reset()
		return buf
	}
	buf := bytes.NewBuffer(make([]byte, poolItem.size))
	buf.Reset()
	poolItem.inuse++
	return buf
}

func (poolItem *BufferPoolItem) freeBuffer(buf *bytes.Buffer) {

	if buf.Cap() == poolItem.size && poolItem.buffers.Len() < poolItem.max {
		poolItem.buffers.PushBack(buf)
	}
	poolItem.inuse--
}

type BufferPool struct {
	items [5]*BufferPoolItem // The key is the network ID
	mutex sync.RWMutex
}

func newBufferPool() *BufferPool {

	pool := &BufferPool{}

	pool.init()

	return pool
}

func (pool *BufferPool) init() {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	pool.items[0] = newBufferPoolItem(1024, 1024)
	pool.items[1] = newBufferPoolItem(1024*4, 512)
	pool.items[2] = newBufferPoolItem(1024*32, 256)
	pool.items[3] = newBufferPoolItem(1024*512, 64)
	pool.items[4] = newBufferPoolItem(1024*1024*1.5, 32)
}

func (pool *BufferPool) GetPoolItem(size int) *BufferPoolItem {

	for i := 0; i < len(pool.items); i++ {
		if pool.items[i].size >= size {
			return pool.items[i]
		}
	}
	return nil
}

func (pool *BufferPool) Print() {

	for i := 0; i < len(pool.items); i++ {
		Logger.Debugf("[ BufferPool Print ] size :%v  count :%v inuse: %v", pool.items[i].size, pool.items[i].buffers.Len(), pool.items[i].inuse)
	}
}

func (pool *BufferPool) getBuffer(size int) *bytes.Buffer {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	poolItem := pool.GetPoolItem(size)
	if poolItem != nil {
		buf := poolItem.getBuffer()
		return buf
	}

	return new(bytes.Buffer)
}

func (pool *BufferPool) freeBuffer(buf *bytes.Buffer) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	poolItem := pool.GetPoolItem(buf.Cap())
	if poolItem != nil {
		poolItem.freeBuffer(buf)
	}

}
