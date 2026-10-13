package waf

import (
	"sync"
)

// PooledBuffer wraps a byte slice to ensure safe return to the pool.
type PooledBuffer struct {
	Data []byte
}

// Close returns the buffer to the pool.
func (p *PooledBuffer) Close() error {
	if p.Data != nil {
		bufferPool.Put(p.Data[:cap(p.Data)])
		p.Data = nil
	}
	return nil
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

// GetBuffer returns a raw byte slice from the pool.
func GetBuffer() []byte {
	return bufferPool.Get().([]byte)
}

// PutBuffer returns a raw byte slice to the pool.
func PutBuffer(b []byte) {
	bufferPool.Put(b)
}

// GetPooledBuffer returns a PooledBuffer wrapper.
func GetPooledBuffer() *PooledBuffer {
	return &PooledBuffer{Data: bufferPool.Get().([]byte)}
}
