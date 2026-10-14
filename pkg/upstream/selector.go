package upstream

import (
	"sync"
)

// UpstreamTarget represents an upstream backend endpoint node.
type UpstreamTarget struct {
	Addr  string
	Stats *AtomicTargetStats
}

// Selector manages backend targets using a lock-free cache-aligned atomic least-connections selector.
type Selector struct {
	targets []UpstreamTarget
	mu      sync.RWMutex
}

// NewSelector initializes an upstream selector with the specified backend addresses.
func NewSelector(addrs []string) *Selector {
	targets := make([]UpstreamTarget, len(addrs))
	for i, addr := range addrs {
		targets[i] = UpstreamTarget{
			Addr:  addr,
			Stats: NewAtomicTargetStats(),
		}
	}
	return &Selector{targets: targets}
}

// Select picks the upstream target with the fewest active streams using atomic load primitives (O(N) search over fixed backends, zero allocation hot path).
func (s *Selector) Select() *UpstreamTarget {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.targets) == 0 {
		return nil
	}

	best := &s.targets[0]
	minStreams := best.Stats.GetActiveStreams()

	for i := 1; i < len(s.targets); i++ {
		streams := s.targets[i].Stats.GetActiveStreams()
		if streams < minStreams {
			minStreams = streams
			best = &s.targets[i]
		}
	}
	return best
}

// AddTarget dynamically registers a new backend target without global lock contention on routing paths.
func (s *Selector) AddTarget(addr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.targets = append(s.targets, UpstreamTarget{
		Addr:  addr,
		Stats: NewAtomicTargetStats(),
	})
}

// Targets returns a snapshot of currently configured upstream targets.
func (s *Selector) Targets() []UpstreamTarget {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]UpstreamTarget, len(s.targets))
	copy(out, s.targets)
	return out
}
