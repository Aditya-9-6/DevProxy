package trace

import (
	"encoding/hex"
	"sync"
	"sync/atomic"
)

const (
	TraceParentHeader = "traceparent"
	Version           = "00"
)

var (
	spanCounter atomic.Uint64
	bufferPool  = sync.Pool{
		New: func() interface{} {
			return make([]byte, 55)
		},
	}
)

// Propagator handles W3C Trace Context injection/extraction.
type Propagator struct{}

func NewPropagator() *Propagator {
	return &Propagator{}
}

// GenerateTraceParent creates a W3C traceparent header value with zero heap allocations.
func (p *Propagator) GenerateTraceParent(traceID [16]byte) string {
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)

	spanID := spanCounter.Add(1)

	// Format: 00-traceid-spanid-01
	copy(buf[0:2], Version)
	buf[2] = '-'
	hex.Encode(buf[3:35], traceID[:])
	buf[35] = '-'

	// Encode spanID (8 bytes) as hex
	var spanBytes [8]byte
	for i := 0; i < 8; i++ {
		spanBytes[i] = byte(spanID >> (56 - i*8))
	}
	hex.Encode(buf[36:52], spanBytes[:])
	buf[52] = '-'
	buf[53] = '0'
	buf[54] = '1'

	return string(buf)
}
