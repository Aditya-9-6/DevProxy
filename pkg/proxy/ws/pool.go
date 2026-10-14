package ws

import "sync"

var frameBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 65536)
	},
}

func GetBuffer() []byte {
	return frameBufferPool.Get().([]byte)
}

func PutBuffer(b []byte) {
	if cap(b) >= 65536 {
		frameBufferPool.Put(b)
	}
}
