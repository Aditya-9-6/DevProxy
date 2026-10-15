package telemetry

import "sync"

// BufferPool manages pre-allocated 2048-byte blocks for flight recording.
var BufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 2048)
	},
}
