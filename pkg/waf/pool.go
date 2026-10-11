package waf

import "sync"

var BufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}
