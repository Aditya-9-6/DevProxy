package http3

import "sync"

// streamBufferPool provides zero-allocation byte buffers for HTTP/3 stream processing.
var streamBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 16384)
	},
}

// GetBuffer retrieves a buffer from the pool.
func GetBuffer() []byte {
	return streamBufferPool.Get().([]byte)
}

// PutBuffer returns a buffer to the pool.
func PutBuffer(b []byte) {
	if cap(b) == 16384 {
		streamBufferPool.Put(b)
	}
}
