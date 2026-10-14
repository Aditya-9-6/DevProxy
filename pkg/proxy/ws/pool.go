package ws

import "sync"

var (
	smallPool = sync.Pool{New: func() interface{} { return make([]byte, 4096) }}
	largePool = sync.Pool{New: func() interface{} { return make([]byte, 65536) }}
)

func GetBufferBySize(size int) []byte {
	if size <= 4096 {
		return smallPool.Get().([]byte)
	}
	return largePool.Get().([]byte)
}

func PutBuffer(b []byte) {
	if cap(b) >= 65536 {
		largePool.Put(b)
	} else if cap(b) >= 4096 {
		smallPool.Put(b)
	}
}
