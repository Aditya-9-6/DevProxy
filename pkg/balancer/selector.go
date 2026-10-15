package balancer

import (
	"github.com/Aditya-9-6/DevProxy/pkg/proxy/congestion"
	"sync"
)

// UpstreamTarget represents a backend server.
type UpstreamTarget struct {
	Addr       string
	Stats      *TargetStats
	ECNTracker *congestion.ECNTracker
}

// Selector manages target selection via weighted least-connections and ECN feedback.
type Selector struct {
	targets []UpstreamTarget
	mu      sync.RWMutex
}

func NewSelector(addrs []string) *Selector {
	targets := make([]UpstreamTarget, len(addrs))
	for i, addr := range addrs {
		targets[i] = UpstreamTarget{
			Addr:       addr,
			Stats:      &TargetStats{},
			ECNTracker: congestion.NewECNTracker(),
		}
	}
	return &Selector{targets: targets}
}

// Select returns the target with the lowest load score, adjusted by ECN congestion.
func (s *Selector) Select() *UpstreamTarget {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.targets) == 0 {
		return nil
	}

	best := &s.targets[0]
	// Score = Connections + Latency + (ECN_Weight * CongestionCount)
	minScore := best.Stats.GetScore() + (best.ECNTracker.GetCongestionScore() * 500)

	for i := 1; i < len(s.targets); i++ {
		score := s.targets[i].Stats.GetScore() + (s.targets[i].ECNTracker.GetCongestionScore() * 500)
		if score < minScore {
			minScore = score
			best = &s.targets[i]
		}
	}
	return best
}
