package tls

import (
	"sync"
)

// BufferPool provides zero-allocation byte buffers for certificate parsing.
var BufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}
