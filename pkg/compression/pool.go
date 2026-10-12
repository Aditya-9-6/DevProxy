package compression

import (
	"sync"
)

var byteBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

// AcquireBuffer acquires a 32KB byte slice from the pool.
func AcquireBuffer() []byte {
	return byteBufferPool.Get().([]byte)
}

// ReleaseBuffer returns a byte slice to the pool.
func ReleaseBuffer(buf []byte) {
	if cap(buf) >= 32*1024 {
		byteBufferPool.Put(buf[:32*1024])
	}
}
