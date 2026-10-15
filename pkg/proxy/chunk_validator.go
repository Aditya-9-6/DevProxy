package proxy

import (
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrChunkDepthExceeded = errors.New("chunk nesting depth exceeded")
	ErrTooManyExtensions  = errors.New("too many chunk extensions")

	// bufferPool provides reusable byte slices for chunk parsing.
	bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 1024)
		},
	}
)

// ChunkValidator tracks state for chunked transfer encoding validation.
// Uses cache-aligned atomic counters to prevent false sharing.
type ChunkValidator struct {
	_            [56]byte
	depth        atomic.Uint64
	extensionCnt atomic.Uint64
	_            [56]byte
	maxDepth     uint64
	maxExt       uint64
}

func NewChunkValidator(maxDepth, maxExt uint64) *ChunkValidator {
	return &ChunkValidator{
		maxDepth: maxDepth,
		maxExt:   maxExt,
	}
}

// ValidateChunk processes a chunk header. Returns error if limits are exceeded.
func (v *ChunkValidator) ValidateChunk(depthDelta int, extCount uint64) error {
	if depthDelta > 0 {
		if v.depth.Add(uint64(depthDelta)) > v.maxDepth {
			return ErrChunkDepthExceeded
		}
	} else if depthDelta < 0 {
		newDepth := v.depth.Load() - uint64(-depthDelta)
		v.depth.Store(newDepth)
	}

	if v.extensionCnt.Add(extCount) > v.maxExt {
		return ErrTooManyExtensions
	}
	return nil
}

func (v *ChunkValidator) Reset() {
	v.depth.Store(0)
	v.extensionCnt.Store(0)
}
