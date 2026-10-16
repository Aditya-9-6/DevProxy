package waf

import (
	"sync"
)

// BufferPool is a global pool for 32KB buffers.
var BufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

// PooledBuffer wraps a byte slice to ensure safe return to the pool.
type PooledBuffer struct {
	Data []byte
}

// GetBuffer retrieves a buffer from the internal pool.
func GetBuffer() []byte {
	return BufferPool.Get().([]byte)
}

// PutBuffer returns a buffer to the internal pool.
func PutBuffer(b []byte) {
	BufferPool.Put(b)
}

// GetPooledBuffer returns a PooledBuffer wrapper.
func GetPooledBuffer() *PooledBuffer {
	return &PooledBuffer{Data: BufferPool.Get().([]byte)}
}

// Close returns the buffer to the pool.
func (p *PooledBuffer) Close() error {
	if p.Data != nil {
		BufferPool.Put(p.Data[:cap(p.Data)])
		p.Data = nil
	}
	return nil
}
