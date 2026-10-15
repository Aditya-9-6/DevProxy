package h2

import (
	"sync/atomic"
)

const MaxHeaderBlockSize = 1024 * 1024 // 1MB limit per header block

// StreamState tracks the state of an HTTP/2 stream for security validation.
type StreamState struct {
	HeaderBlockSize   atomic.Uint64
	ContinuationCount atomic.Uint64
}

// StreamTracker manages state for up to 65536 concurrent streams using a fixed-size array.
type StreamTracker struct {
	streams [65536]StreamState
}

func NewStreamTracker() *StreamTracker {
	return &StreamTracker{}
}

func (t *StreamTracker) Reset(streamID uint32) {
	idx := streamID % 65536
	t.streams[idx].HeaderBlockSize.Store(0)
	t.streams[idx].ContinuationCount.Store(0)
}

func (t *StreamTracker) Get(streamID uint32) *StreamState {
	return &t.streams[streamID%65536]
}
