package waf

import "sync"

var BufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
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

// GetSmallBuffer returns a smaller buffer from the secondary pool.
func GetSmallBuffer() []byte {
	return bufferPool.Get().([]byte)
}

// PutSmallBuffer returns a small buffer to the secondary pool.
func PutSmallBuffer(b []byte) {
	bufferPool.Put(b)
}