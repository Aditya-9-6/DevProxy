package h2

import (
	"context"
	"sync"
	"time"
)

// StreamReaper monitors and cleans up zombie streams.
type StreamReaper struct {
	streams sync.Map
	timeout time.Duration
}

func NewStreamReaper(timeout time.Duration) *StreamReaper {
	return &StreamReaper{timeout: timeout}
}

func (r *StreamReaper) Register(s *StreamState) {
	r.streams.Store(s.ID, s)
}

func (r *StreamReaper) Run(ctx context.Context) {
	ticker := time.NewTicker(r.timeout / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.streams.Range(func(key, value interface{}) bool {
				s := value.(*StreamState)
				if s.GetIdleDuration() > r.timeout {
					r.streams.Delete(key)
				}
				return true
			})
		}
	}
}
