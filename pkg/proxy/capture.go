package proxy

import (
	"bytes"
	"io"
	"sync"
)

// DefaultMaxBodyCaptureBytes limits how many bytes of a body are buffered for security scanning (64 KB).
// The rest of the stream passes through to the client at full line-rate without consuming RAM.
const DefaultMaxBodyCaptureBytes = 64 * 1024

// BoundedCaptureWriter records up to limit bytes while accepting unbounded writes.
type BoundedCaptureWriter struct {
	buf   bytes.Buffer
	limit int
	mu    sync.Mutex
}

// NewBoundedCaptureWriter creates a writer that caps stored bytes at limit.
func NewBoundedCaptureWriter(limit int) *BoundedCaptureWriter {
	if limit <= 0 {
		limit = DefaultMaxBodyCaptureBytes
	}
	return &BoundedCaptureWriter{limit: limit}
}

func (w *BoundedCaptureWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	remaining := w.limit - w.buf.Len()
	if remaining > 0 {
		if len(p) <= remaining {
			w.buf.Write(p)
		} else {
			w.buf.Write(p[:remaining])
		}
	}
	return len(p), nil
}

// Bytes returns the captured bytes.
func (w *BoundedCaptureWriter) Bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Bytes()
}

// ReadAndCapture wraps an io.ReadCloser with a bounded capture buffer.
// As the caller reads from the returned reader, bytes are streamed directly while the first
// 'limit' bytes are copied into the capture buffer.
func ReadAndCapture(rc io.ReadCloser, limit int) (io.ReadCloser, *BoundedCaptureWriter) {
	if rc == nil {
		return nil, NewBoundedCaptureWriter(limit)
	}
	capWriter := NewBoundedCaptureWriter(limit)
	tee := io.TeeReader(rc, capWriter)
	return &wrappedReadCloser{Reader: tee, Closer: rc}, capWriter
}

type wrappedReadCloser struct {
	io.Reader
	io.Closer
}
