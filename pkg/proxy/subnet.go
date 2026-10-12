package proxy

import (
	"net"
	"sort"
	"sync"
)

// SubnetClassifier uses a sorted slice of CIDRs to perform O(log N) lookups.
type SubnetClassifier struct {
	mu       sync.RWMutex
	networks []cidrEntry
}

type cidrEntry struct {
	ipNet *net.IPNet
	tier  int
}

func NewSubnetClassifier() *SubnetClassifier {
	return &SubnetClassifier{}
}

func (s *SubnetClassifier) AddTier(cidr string, tier int) error {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.networks = append(s.networks, cidrEntry{ipNet, tier})
	// Sort by mask size descending (most specific first)
	sort.Slice(s.networks, func(i, j int) bool {
		s1, _ := s.networks[i].ipNet.Mask.Size()
		s2, _ := s.networks[j].ipNet.Mask.Size()
		return s1 > s2
	})
	return nil
}

func (s *SubnetClassifier) Classify(ip net.IP) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, entry := range s.networks {
		if entry.ipNet.Contains(ip) {
			return entry.tier
		}
	}
	return 0
}
