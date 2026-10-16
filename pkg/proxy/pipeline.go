package proxy

import (
<<<<<<< HEAD
	"context"
	"net/http"
	"sync"
	"sync/atomic"
)

// CacheLinePad prevents false sharing.
type CacheLinePad [8]uint64

// PipelineQueue manages pipelined HTTP/1.1 requests without locks.
type PipelineQueue struct {
	capacity uint64
	count    atomic.Uint64
	padding  CacheLinePad
	pool     sync.Pool
	queue    chan *http.Request
}

func NewPipelineQueue(capacity int) *PipelineQueue {
	return &PipelineQueue{
		capacity: uint64(capacity),
		queue:    make(chan *http.Request, capacity),
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
=======
	"github.com/Aditya-9-6/DevProxy/pkg/security"
	"net/http"
)

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
>>>>>>> origin/main
}
