package replay

import (
	"context"
	"net/http"
	"sync/atomic"
)

// Recorder captures traffic events and sanitizes them before storage.
type Recorder struct {
	ringBuf   *RingBuffer
	sanitizer *Sanitizer
	seq       atomic.Uint64
}

// NewRecorder creates a new traffic recorder using the provided ring buffer.
func NewRecorder(rb *RingBuffer) *Recorder {
	return &Recorder{
		ringBuf:   rb,
		sanitizer: NewSanitizer(),
	}
}

// Record captures an HTTP transaction, sanitizes it, and pushes it to the ring buffer.
func (r *Recorder) Record(ctx context.Context, req *http.Request, respBody []byte) {
	select {
	case <-ctx.Done():
		return
	default:
		// Sanitize headers if request is present
		if req != nil && req.Header != nil {
			r.sanitizer.SanitizeHeaders(req.Header)
		}

		// Sanitize body
		sanitizedBody := r.sanitizer.SanitizeBody(respBody)

		seqVal := r.seq.Add(1)
		event := &TrafficEvent{
			Seq:     seqVal,
			Payload: sanitizedBody,
		}

		// Push to ring buffer; if full, the event is dropped to preserve proxy performance
		r.ringBuf.Push(event)
	}
}
