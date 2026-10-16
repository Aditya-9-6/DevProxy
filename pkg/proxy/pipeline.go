package proxy

import (
	"context"
	"github.com/Aditya-9-6/DevProxy/pkg/internal/align"
	"github.com/Aditya-9-6/DevProxy/pkg/security"
	"net/http"
	"sync"
	"sync/atomic"
)

// PipelineQueue manages pipelined HTTP/1.1 requests without locks.
// It ensures that the capacity is within the uint64 range to prevent overflow.
type PipelineQueue struct {
	capacity uint64
	count    atomic.Uint64
	padding  align.CacheLinePad
	pool     sync.Pool
	queue    chan *http.Request
}

func NewPipelineQueue(capacity int) *PipelineQueue {
	// Convert capacity to uint64 to ensure it's within the range
	cap := uint64(capacity)

	// Check for integer overflow
	if cap > ^uint64(0) { // ^uint64(0) is the maximum value for uint64
		cap = ^uint64(0)
	}

	return &PipelineQueue{
		capacity: cap,
		queue:    make(chan *http.Request, int(cap)),
		pool: sync.Pool{
			New: func() interface{} {
				return new(http.Request)
			},
		},
	}
}

func (p *PipelineQueue) Enqueue(req *http.Request) bool {
	if p.count.Load() >= p.capacity {
		return false
	}

	p.count.Add(1)

	select {
	case p.queue <- req:
		return true
	default:
		p.count.Add(^uint64(0))
		return false
	}
}

func (p *PipelineQueue) Dequeue(ctx context.Context) (*http.Request, bool) {
	select {
	case <-ctx.Done():
		return nil, false
	case req := <-p.queue:
		p.count.Add(^uint64(0))
		return req, true
	}
}

// PipelineHandler manages HTTP/1.1 pipelining and smuggling protection.
type PipelineHandler struct {
	sanitizer *security.SmugglingSanitizer
}

func NewPipelineHandler() *PipelineHandler {
	return &PipelineHandler{
		sanitizer: security.NewSmugglingSanitizer(),
	}
}

// ValidateRequest checks for smuggling vulnerabilities before routing.
func (h *PipelineHandler) ValidateRequest(req *http.Request) bool {
	return h.sanitizer.Sanitize(req.Header)
}
