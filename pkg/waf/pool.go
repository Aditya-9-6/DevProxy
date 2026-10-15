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

func GetBuffer() []byte {
	return bufferPool.Get().([]byte)
}

func PutBuffer(b []byte) {
	bufferPool.Put(b)
}
