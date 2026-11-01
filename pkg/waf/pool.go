package waf

import (
	"sync"
)

var bufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 0, 4096)
		return &buf
	},
}

// GetBuffer retrieves a byte slice buffer from the pool.
func GetBuffer() *[]byte {
	return bufferPool.Get().(*[]byte)
}

// PutBuffer returns a byte slice buffer to the pool after resetting length.
func PutBuffer(buf *[]byte) {
	*buf = (*buf)[:0]
bufferPool.Put(buf)
}
