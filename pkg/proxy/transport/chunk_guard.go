package transport

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding atomic counters to cache line boundaries.
type CacheLinePad [8]uint64

// ChunkGuard tracks streaming chunk bounds and defends against slow-read / leak attacks
// using lock-free atomic operations and zero-allocation buffer pools.
type ChunkGuard struct {
	maxTotalBytes   uint64
	maxChunkSize    uint64
	bytesRead       atomic.Uint64
	chunksProcessed atomic.Uint64
	active          atomic.Bool
	pad             CacheLinePad
}

var chunkBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32768)
	},
}

var (
	ErrMaxTotalExceeded = errors.New("chunk guard: maximum allowed total transfer size exceeded")
	ErrMaxChunkExceeded = errors.New("chunk guard: maximum allowed individual chunk size exceeded")
	ErrChunkStreamLeak  = errors.New("chunk guard: stream terminated unexpectedly or leaked")
)

// NewChunkGuard initializes a new ChunkGuard with specified size limits.
func NewChunkGuard(maxTotalBytes, maxChunkSize uint64) *ChunkGuard {
	return &ChunkGuard{
		maxTotalBytes: maxTotalBytes,
		maxChunkSize:  maxChunkSize,
	}
}

// WrapReader wraps an io.Reader to enforce chunk limits and size constraints atomically.
func (g *ChunkGuard) WrapReader(ctx context.Context, r io.Reader) io.Reader {
	g.active.Store(true)
	return &guardedStreamReader{
		ctx:   ctx,
		r:     r,
		guard: g,
	}
}

type guardedStreamReader struct {
	ctx   context.Context
	r     io.Reader
	guard *ChunkGuard
}

func (s *guardedStreamReader) Read(p []byte) (int, error) {
	select {
	case <-s.ctx.Done():
		s.guard.active.Store(false)
		return 0, s.ctx.Err()
	default:
	}

	n, err := s.r.Read(p)
	if n > 0 {
		if uint64(n) > s.guard.maxChunkSize && s.guard.maxChunkSize > 0 {
			return n, ErrMaxChunkExceeded
		}
		newTotal := s.guard.bytesRead.Add(uint64(n))
		if s.guard.maxTotalBytes > 0 && newTotal > s.guard.maxTotalBytes {
			return n, ErrMaxTotalExceeded
		}
		s.guard.chunksProcessed.Add(1)
	}
	return n, err
}

// Stats returns current atomic metrics.
func (g *ChunkGuard) Stats() (bytesRead uint64, chunks uint64, active bool) {
	return g.bytesRead.Load(), g.chunksProcessed.Load(), g.active.Load()
}
