package balancer

import (
	"github.com/Aditya-9-6/DevProxy/pkg/proxy/congestion"
	"sync"
	"sync/atomic"
)

// UpstreamTarget represents a backend server with atomic health tracking.
type UpstreamTarget struct {
	Addr       string
	Stats      *TargetStats
	ECNTracker *congestion.ECNTracker
	Weight     uint64
	_          CacheLinePad
}

// Selector manages target selection via weighted least-connections and ECN feedback.
type Selector struct {
	targets []UpstreamTarget
	mu      sync.RWMutex
}

// NewSelector initializes a selector with a static list of targets.
func NewSelector(addrs []string) *Selector {
	targets := make([]UpstreamTarget, len(addrs))
	for i, addr := range addrs {
		targets[i] = UpstreamTarget{
			Addr:       addr,
			Stats:      &TargetStats{},
			ECNTracker: congestion.NewECNTracker(),
			Weight:     100,
		}
	}
	return &Selector{targets: targets}
}

// Select returns the target with the lowest load score, adjusted by ECN and Weight.
func (s *Selector) Select() *UpstreamTarget {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.targets) == 0 {
		return nil
	}

	var best *UpstreamTarget
	minScore := int64(^uint64(0) >> 1)

	for i := range s.targets {
		t := &s.targets[i]
		weight := atomic.LoadUint64(&t.Weight)
		if weight == 0 {
			weight = 1
		}
		// Score = (Connections + Latency + (ECN * 500)) / Weight
		score := int64(t.Stats.GetScore()+(t.ECNTracker.GetCongestionScore()*500)) / int64(weight)

		if score < minScore {
			minScore = score
			best = t
		}
	}
	return best
}

// UpdateTargetWeight allows dynamic adjustment of peer capacity.
func (s *Selector) UpdateTargetWeight(addr string, weight uint64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.targets {
		if s.targets[i].Addr == addr {
			atomic.StoreUint64(&s.targets[i].Weight, weight)
			break
		}
	}
}
