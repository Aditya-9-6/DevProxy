package balancer

import "sync"

// UpstreamTarget represents a backend server.
type UpstreamTarget struct {
	Addr  string
	Stats *TargetStats
}

// Selector manages target selection via weighted least-connections.
type Selector struct {
	targets []UpstreamTarget
	mu      sync.RWMutex
}

func NewSelector(addrs []string) *Selector {
	targets := make([]UpstreamTarget, len(addrs))
	for i, addr := range addrs {
		targets[i] = UpstreamTarget{Addr: addr, Stats: &TargetStats{}}
	}
	return &Selector{targets: targets}
}

// Select returns the target with the lowest load score.
func (s *Selector) Select() *UpstreamTarget {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.targets) == 0 {
		return nil
	}

	best := &s.targets[0]
	minScore := best.Stats.GetScore()

	for i := 1; i < len(s.targets); i++ {
		score := s.targets[i].Stats.GetScore()
		if score < minScore {
			minScore = score
			best = &s.targets[i]
		}
	}
	return best
}
