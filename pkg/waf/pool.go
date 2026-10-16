package waf

import (
	"sync"
)

<<<<<<< HEAD
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
=======
// BufferPool is a global pool for 32KB buffers.
var BufferPool = sync.Pool{
>>>>>>> origin/main
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

<<<<<<< HEAD
// GetBuffer returns a raw byte slice from the pool.
=======
var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

// GetBuffer retrieves a 4KB buffer from the internal pool.
>>>>>>> origin/main
func GetBuffer() []byte {
	return bufferPool.Get().([]byte)
}

<<<<<<< HEAD
// PutBuffer returns a raw byte slice to the pool.
=======
// PutBuffer returns a 4KB buffer to the internal pool.
>>>>>>> origin/main
func PutBuffer(b []byte) {
	bufferPool.Put(b)
}

// GetPooledBuffer returns a PooledBuffer wrapper.
func GetPooledBuffer() *PooledBuffer {
	return &PooledBuffer{Data: bufferPool.Get().([]byte)}
}
