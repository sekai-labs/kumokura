package adapters

import (
	"sync"
)

const (
	Tier5MB  = 5 * 1024 * 1024
	Tier8MB  = 8 * 1024 * 1024
	Tier16MB = 16 * 1024 * 1024
	Tier32MB = 32 * 1024 * 1024
	Tier64MB = 64 * 1024 * 1024
)

var StandardTiers = [...]int{
	Tier5MB,
	Tier8MB,
	Tier16MB,
	Tier32MB,
	Tier64MB,
}

func QuantizeBufferSize(requested int) int {
	for _, tier := range StandardTiers {
		if requested <= tier {
			return tier
		}
	}
	return Tier64MB
}

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
	if buf != nil && cap(*buf) >= m.size {
		*buf = (*buf)[:m.size]
		m.pool.Put(buf)
	}
}

type TieredBufferPool struct {
	pools [5]*BufferPoolManager
}

func NewTieredBufferPool() *TieredBufferPool {
	t := &TieredBufferPool{}
	for i, tier := range StandardTiers {
		t.pools[i] = NewBufferPoolManager(tier)
	}
	return t
}

func (t *TieredBufferPool) getTierIndex(size int) int {
	quantized := QuantizeBufferSize(size)
	for i, tier := range StandardTiers {
		if tier == quantized {
			return i
		}
	}
	return len(StandardTiers) - 1
}

func (t *TieredBufferPool) Get(size int) *[]byte {
	idx := t.getTierIndex(size)
	return t.pools[idx].Get()
}

func (t *TieredBufferPool) Put(size int, buf *[]byte) {
	if buf == nil {
		return
	}
	idx := t.getTierIndex(size)
	t.pools[idx].Put(buf)
}
