package waf

import "sync"

var BufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

// Renamed internal buffer pool or consolidated to avoid redeclaration with json_sanitizer.go
var wafAltBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

func GetBuffer() []byte {
	return wafAltBufferPool.Get().([]byte)
}

func PutBuffer(b []byte) {
	wafAltBufferPool.Put(b)
}
