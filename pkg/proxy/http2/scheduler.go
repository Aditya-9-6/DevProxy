package http2

import (
	"sync"
)

// StreamPriority represents the priority state of an HTTP/2 stream.
type StreamPriority struct {
	Weight    uint8
	Dependent uint32
	Exclusive bool
}

// Scheduler manages stream scheduling.
type Scheduler struct {
	pool sync.Pool
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		pool: sync.Pool{
			New: func() interface{} {
				return &StreamPriority{}
			},
		},
	}
}

func (s *Scheduler) GetPriority() *StreamPriority {
	return s.pool.Get().(*StreamPriority)
}

func (s *Scheduler) PutPriority(p *StreamPriority) {
	s.pool.Put(p)
}
