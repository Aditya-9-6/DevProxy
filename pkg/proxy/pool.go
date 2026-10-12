package proxy

import "sync"

// StreamBufferSize defines the buffer size used for streaming data chunks (32 KB).
const StreamBufferSize = 32 * 1024

// BufferPool provides zero-allocation byte buffers for streaming.
var BufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, StreamBufferSize)
		return &buf
	},
}

// GetBuffer retrieves a recycled 32KB buffer from the memory pool.
func GetBuffer() *[]byte {
	return BufferPool.Get().(*[]byte)
}

// PutBuffer returns a buffer to the memory pool.
func PutBuffer(b *[]byte) {
	if b != nil {
		BufferPool.Put(b)
	}
}
