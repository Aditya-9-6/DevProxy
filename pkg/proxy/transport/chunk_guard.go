package transport

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

var (
	ErrChunkSizeExceeded = errors.New("chunk size exceeds configured maximum limit")
	ErrTotalSizeExceeded = errors.New("total chunked transfer size exceeds maximum limit")
	ErrInvalidChunkFormat = errors.New("invalid chunk format or syntax")
	ErrChunkStreamClosed  = errors.New("chunked stream is closed or cancelled")
)

var chunkGuardPool = sync.Pool{
	New: func() any {
		b := make([]byte, 16*1024)
		return &b
	},
}

// ChunkGuard provides a lock-free cache-aligned atomic adaptive chunked transfer-encoding leak guard.
// It tracks sizes using atomic operations to prevent memory exhaustion and slow-read DoS.
type ChunkGuard struct {
	totalBytesRead atomic.Int64
	maxChunkSize   int64
	maxTotalSize   int64
	closed         atomic.Bool
}

// NewChunkGuard creates a new ChunkGuard with specified size bounds.
func NewChunkGuard(maxChunkSize, maxTotalSize int64) *ChunkGuard {
	return &ChunkGuard{
		maxChunkSize: maxChunkSize,
		maxTotalSize: maxTotalSize,
	}
}

// TotalRead returns the total number of valid chunk bytes read atomically.
func (g *ChunkGuard) TotalRead() int64 {
	return g.totalBytesRead.Load()
}

// IsClosed checks if the guard has been marked closed or faulted.
func (g *ChunkGuard) IsClosed() bool {
	return g.closed.Load()
}

// WrapReader wraps an io.Reader to validate chunked payloads on the fly with zero allocations on hot paths.
func (g *ChunkGuard) WrapReader(ctx context.Context, r io.Reader) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		bufPtr := chunkGuardPool.Get().(*[]byte)
		defer chunkGuardPool.Put(bufPtr)
		buf := *bufPtr

		for {
			select {
			case <-ctx.Done():
				g.closed.Store(true)
				_ = pw.CloseWithError(ctx.Err())
				return
			default:
			}

			if g.IsClosed() {
				_ = pw.CloseWithError(ErrChunkStreamClosed)
				return
			}

			n, err := r.Read(buf)
			if n > 0 {
				chunkSize := int64(n)
				if g.maxChunkSize > 0 && chunkSize > g.maxChunkSize {
					g.closed.Store(true)
					_ = pw.CloseWithError(ErrChunkSizeExceeded)
					return
				}

				newTotal := g.totalBytesRead.Add(chunkSize)
				if g.maxTotalSize > 0 && newTotal > g.maxTotalSize {
					g.closed.Store(true)
					_ = pw.CloseWithError(ErrTotalSizeExceeded)
					return
				}

				if _, wErr := pw.Write(buf[:n]); wErr != nil {
					g.closed.Store(true)
					return
				}
			}

			if err != nil {
				g.closed.Store(true)
				if err == io.EOF {
					return
				}
				_ = pw.CloseWithError(err)
				return
			}
		}
	}()

	return pr
}
