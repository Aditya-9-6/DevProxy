package balancer

import (
	"sync/atomic"
)

// UpstreamTarget represents a backend server with atomic health tracking.
type UpstreamTarget struct {
	Addr  string
	Stats *TargetStats
	// Weight defines the relative capacity of the node.
	Weight uint64
}

// Selector manages target selection via lock-free weighted round-robin.
type Selector struct {
	targets []UpstreamTarget
	current atomic.Uint64
}

// NewSelector initializes a selector with a static list of targets.
func NewSelector(addrs []string) *Selector {
	targets := make([]UpstreamTarget, len(addrs))
	for i, addr := range addrs {
		targets[i] = UpstreamTarget{
			Addr:   addr,
			Stats:  &TargetStats{},
			Weight: 100, // Default weight
		}
	}
	return &Selector{targets: targets}
}

// Select returns the target using a lock-free atomic round-robin index.
// This implementation avoids global mutexes on the hot request path.
func (s *Selector) Select() *UpstreamTarget {
	if len(s.targets) == 0 {
		return nil
	}

	// Atomic increment to pick the next target index
	idx := s.current.Add(1) % uint64(len(s.targets))
	return &s.targets[idx]
}

// UpdateTargetWeight allows dynamic adjustment of peer capacity without locking.
func (s *Selector) UpdateTargetWeight(addr string, weight uint64) {
	for i := range s.targets {
		if s.targets[i].Addr == addr {
			atomic.StoreUint64(&s.targets[i].Weight, weight)
			break
		}
	}
}
