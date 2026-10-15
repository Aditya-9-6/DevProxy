package balancer

import (
	"github.com/Aditya-9-6/DevProxy/pkg/proxy/congestion"
	"sync/atomic"
)

// UpstreamTarget represents a backend server with atomic health tracking.
type UpstreamTarget struct {
	Addr       string
	Stats      *TargetStats
	ECNTracker *congestion.ECNTracker
	// Weight defines the relative capacity of the node.
	Weight uint64
}

// Selector manages target selection via weighted least-connections and ECN feedback.
type Selector struct {
	targets []UpstreamTarget
	current atomic.Uint64
}

// NewSelector initializes a selector with a static list of targets.
func NewSelector(addrs []string) *Selector {
	targets := make([]UpstreamTarget, len(addrs))
	for i, addr := range addrs {
		targets[i] = UpstreamTarget{
			Addr:       addr,
			Stats:      &TargetStats{},
			ECNTracker: congestion.NewECNTracker(),
			Weight:     100, // Default weight
		}
	}
	return &Selector{targets: targets}
}

// Select returns the target with the lowest load score, adjusted by ECN congestion.
func (s *Selector) Select() *UpstreamTarget {
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

// UpdateTargetWeight allows dynamic adjustment of peer capacity without locking.
func (s *Selector) UpdateTargetWeight(addr string, weight uint64) {
	for i := range s.targets {
		if s.targets[i].Addr == addr {
			atomic.StoreUint64(&s.targets[i].Weight, weight)
			break
		}
	}
}
