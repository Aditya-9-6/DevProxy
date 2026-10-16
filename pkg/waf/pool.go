package waf

import "sync"

<<<<<<< HEAD
// BufferPool provides 32KB buffers for large operations.
=======
// BufferPool is a global pool for 32KB buffers.
>>>>>>> origin/main
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

<<<<<<< HEAD
// GetBuffer returns a 32KB buffer from the pool.
=======
// GetBuffer retrieves a 4KB buffer from the internal pool.
>>>>>>> origin/main
func GetBuffer() []byte {
	return BufferPool.Get().([]byte)
}

// PutBuffer returns a 32KB buffer to the pool.
func PutBuffer(b []byte) {
	BufferPool.Put(b)
}

// GetSmallBuffer returns a 4KB buffer from the secondary pool.
func GetSmallBuffer() []byte {
	return bufferPool.Get().([]byte)
}

<<<<<<< HEAD
// PutSmallBuffer returns a 4KB buffer to the secondary pool.
func PutSmallBuffer(b []byte) {
=======
// PutBuffer returns a 4KB buffer to the internal pool.
func PutBuffer(b []byte) {
>>>>>>> origin/main
	bufferPool.Put(b)
}
