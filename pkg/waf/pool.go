package waf

import "sync"

// BufferPool is a global pool for 32KB buffers.
var BufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}
<<<<<<< HEAD
=======

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}
>>>>>>> origin/main

// GetBuffer retrieves a 4KB buffer from the internal pool.
func GetBuffer() []byte {
	return bufferPool.Get().([]byte)
}

// PutBuffer returns a 4KB buffer to the internal pool.
func PutBuffer(b []byte) {
	bufferPool.Put(b)
}
