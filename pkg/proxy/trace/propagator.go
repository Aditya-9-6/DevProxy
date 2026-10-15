package trace

import (
	"encoding/hex"
	"io"
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

// WriteTraceParent writes a W3C traceparent header value to the provided writer with zero heap allocations.
func (p *Propagator) WriteTraceParent(w io.Writer, traceID [16]byte) error {
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)

	spanID := spanCounter.Add(1)

	copy(buf[0:2], Version)
	buf[2] = '-'
	hex.Encode(buf[3:35], traceID[:])
	buf[35] = '-'

	var spanBytes [8]byte
	for i := 0; i < 8; i++ {
		shift := uint(56 - i*8)
		spanBytes[i] = uint8((spanID >> shift) & 0xFF)
	}
	hex.Encode(buf[36:52], spanBytes[:])
	buf[52] = '-'
	buf[53] = '0'
	buf[54] = '1'

	_, err := w.Write(buf)
	return err
}
