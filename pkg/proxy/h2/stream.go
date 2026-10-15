package h2

import (
	"sync/atomic"
	"time"
)

// CacheLinePad prevents false sharing.
type CacheLinePad [8]uint64

// StreamState tracks the lifecycle of an HTTP/2 stream.
type StreamState struct {
	ID           uint32
	LastActivity CacheLinePad
	State        atomic.Uint32 // 0: Idle, 1: Open, 2: HalfClosed, 3: Closed
}

func NewStreamState(id uint32) *StreamState {
	s := &StreamState{ID: id}
	s.LastActivity[0] = uint64(time.Now().UnixNano())
	return s
}

func (s *StreamState) UpdateActivity() {
	s.LastActivity[0] = uint64(time.Now().UnixNano())
}

func (s *StreamState) GetIdleDuration() time.Duration {
	last := atomic.LoadUint64(&s.LastActivity[0])
	return time.Since(time.Unix(0, int64(last)))
}
