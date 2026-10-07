package adapters

import (
	"sync"
)

type BufferPoolManager struct {
	pool sync.Pool
	size int
}

func NewBufferPoolManager(bufferSize int) *BufferPoolManager {
	return &BufferPoolManager{
		size: bufferSize,
		pool: sync.Pool{
			New: func() any {
				buf := make([]byte, bufferSize)
				return &buf
			},
		},
	}
}

func (m *BufferPoolManager) Get() *[]byte {
	return m.pool.Get().(*[]byte)
}

func (m *BufferPoolManager) Put(buf *[]byte) {
	if buf != nil && len(*buf) == m.size {
		m.pool.Put(buf)
	}
}

type TieredBufferPool struct {
	mu    sync.RWMutex
	pools map[int]*BufferPoolManager
}

func NewTieredBufferPool() *TieredBufferPool {
	return &TieredBufferPool{
		pools: make(map[int]*BufferPoolManager),
	}
}

func (t *TieredBufferPool) Get(size int) *[]byte {
	t.mu.RLock()
	p, ok := t.pools[size]
	t.mu.RUnlock()

	if !ok {
		t.mu.Lock()
		p, ok = t.pools[size]
		if !ok {
			p = NewBufferPoolManager(size)
			t.pools[size] = p
		}
		t.mu.Unlock()
	}

	return p.Get()
}

func (t *TieredBufferPool) Put(size int, buf *[]byte) {
	t.mu.RLock()
	p, ok := t.pools[size]
	t.mu.RUnlock()

	if ok && buf != nil {
		p.Put(buf)
	}
}
