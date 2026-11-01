package proxy

import (
	"io"
	"sync"
	"github.com/Aditya-9-6/DevProxy/pkg/proxy/http2flow"
)

// HTTP2ResponseWriter handles high-throughput, zero-allocation HTTP/2 frame writing.
type HTTP2ResponseWriter struct {
	writer io.Writer
	ledger *http2flow.Ledger
	mu     sync.Mutex
}

// NewHTTP2ResponseWriter creates a new HTTP2ResponseWriter.
func NewHTTP2ResponseWriter(w io.Writer, initialWindow int64) *HTTP2ResponseWriter {
	return &HTTP2ResponseWriter{
		writer: w,
		ledger: http2flow.NewLedger(initialWindow),
	}
}

// WriteFrame writes a payload frame respecting the flow control window.
func (r *HTTP2ResponseWriter) WriteFrame(streamID uint32, p []byte) (int, error) {
	n := int64(len(p))
	if !r.ledger.Consume(n) {
		return 0, io.ErrShortWrite
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	written, err := r.writer.Write(p)
	if err != nil {
		r.ledger.Replenish(n) // Rollback or partial handling
		return written, err
	}

	return written, nil
}

// ReplenishWindow increases the flow control window.
func (r *HTTP2ResponseWriter) ReplenishWindow(n int64) int64 {
	return r.ledger.Replenish(n)
}
