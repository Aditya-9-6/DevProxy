package waf

import "sync"

// BufferPool provides a global pool of 32KB buffers for large operations.
var BufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

// bufferPool provides 4KB buffers for smaller operations.
var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

// GetBuffer retrieves a 32KB buffer from the pool.
func GetBuffer() []byte {
	return BufferPool.Get().([]byte)
}

// PutBuffer returns the 32KB buffer to the pool.
func PutBuffer(b []byte) {
	BufferPool.Put(b)
}

// GetSmallBuffer fetches a 4KB buffer from the secondary pool.
func GetSmallBuffer() []byte {
	return bufferPool.Get().([]byte)
}

// PutSmallBuffer returns a used 4KB buffer to the secondary pool.
func PutSmallBuffer(b []byte) {
	bufferPool.Put(b)
}
