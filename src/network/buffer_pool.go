package network

import (
	"sync"
	"container/list"
	"bytes"
)


type BufferPoolItem struct {
	buffers     *list.List
	size 		int
	max			int
}

func newBufferPoolItem(size int ,max int) *BufferPoolItem {

	item := &BufferPoolItem{
		buffers:list.New(),size:size,max:max,
	}

	return item
}

func ( poolItem *BufferPoolItem) GetBuffer() *bytes.Buffer {

	if poolItem.buffers.Len() > 0 {
		e := poolItem.buffers.Front()
		buf := e.Value.(*bytes.Buffer)
		poolItem.buffers.Remove(poolItem.buffers.Front())
		buf.Reset()
		return buf
	}
	buf := bytes.NewBuffer(make([]byte,poolItem.size))
	buf.Reset()

	return buf
}

func ( poolItem *BufferPoolItem) freeBuffer(buf *bytes.Buffer ) {
	//for e := poolItem.buffers.Front(); e != nil; e = e.Next() {
	//	item := e.Value.(*bytes.Buffer)
	//	if item == buf {
	//		Logger.Debugf("[ BufferPool ] double free : %p", buf)
	//		return
	//	}
	//}
	if buf.Cap() == poolItem.size && poolItem.buffers.Len() < poolItem.max {
			poolItem.buffers.PushBack(buf)
	}

}

//BufferPool
type BufferPool struct {
	items              [7]*BufferPoolItem //key为网络ID
	mutex              sync.RWMutex
}

func newBufferPool() *BufferPool {

	pool := &BufferPool{
	}

	pool.Init()

	return pool
}

func ( pool *BufferPool) Init()  {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	pool.items[0] = newBufferPoolItem(1024,1024*8)
	pool.items[1] = newBufferPoolItem(1024*4,1024*2)
	pool.items[2] = newBufferPoolItem(1024*8,1024)
	pool.items[3] = newBufferPoolItem(1024*32,256)
	pool.items[4] = newBufferPoolItem(1024*64,128)
	pool.items[5] = newBufferPoolItem(1024*128,64)
	pool.items[6] = newBufferPoolItem(1024*512,32)
}


func ( pool *BufferPool) GetPoolItem(size int) *BufferPoolItem {


	for i:=0;i<len(pool.items);i++ {
		if pool.items[i].size >=  size {
			return pool.items[i]
		}
	}
	return nil
}

func ( pool *BufferPool) Print()  {
	
	for i:=0;i<len(pool.items);i++ {
		Logger.Debugf("[ BufferPool Print ] size :%v  count :%v", pool.items[i].size,pool.items[i].buffers.Len())
	}
}


func ( pool *BufferPool) GetBuffer(size int) *bytes.Buffer {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	//Logger.Debugf("[BufferPool] GetBuffer size : %v ", size )
	if netCore.natTraversalEnable {
		poolItem := pool.GetPoolItem(size)
		if poolItem != nil {
			buf :=  poolItem.GetBuffer()
		//	Logger.Debugf("[BufferPool] GetBuffer buf.Cap:%v address: %p ", buf.Cap(),buf)

			return buf
		}
	}


	return new(bytes.Buffer)
}

func ( pool *BufferPool) FreeBuffer( buf *bytes.Buffer) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	if netCore.natTraversalEnable {
		poolItem := pool.GetPoolItem(buf.Cap())
		if poolItem != nil {
			poolItem.freeBuffer(buf)
		//	Logger.Debugf("[BufferPool] FreeBuffer size : %v buffers :%v address: %p ", buf.Cap(), poolItem.buffers.Len(), buf)
		}
	}
}
