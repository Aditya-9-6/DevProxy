package proxy

import (
	"github.com/Aditya-9-6/DevProxy/pkg/proxy/http2priority"
)

// HTTP2Scheduler wraps http2priority.Scheduler for proxy use.
type HTTP2Scheduler struct {
	s *http2priority.Scheduler
}

// NewHTTP2Scheduler initializes a new proxy HTTP2Scheduler.
func NewHTTP2Scheduler() *HTTP2Scheduler {
	return &HTTP2Scheduler{
		s: http2priority.NewScheduler(),
	}
}
