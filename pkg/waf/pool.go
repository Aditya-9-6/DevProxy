package waf

import "sync"

var BufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

// GetBuffer returns a buffer from the pool.
func GetBuffer() []byte {
	return BufferPool.Get().([]byte)
}

// PutBuffer returns a buffer to the pool.
func PutBuffer(b []byte) {
	BufferPool.Put(b)
}
