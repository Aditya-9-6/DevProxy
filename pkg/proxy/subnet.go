package proxy

import (
	"net"
	"sync"
)

type SubnetClassifier struct {
	mu       sync.RWMutex
	networks []*net.IPNet
	tiers    map[string]int
}

func NewSubnetClassifier() *SubnetClassifier {
	return &SubnetClassifier{
		tiers: make(map[string]int),
	}
}

func (s *SubnetClassifier) AddTier(cidr string, tier int) error {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.networks = append(s.networks, ipNet)
	s.tiers[cidr] = tier
	return nil
}

func (s *SubnetClassifier) Classify(ip net.IP) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, net := range s.networks {
		if net.Contains(ip) {
			return s.tiers[net.String()]
		}
	}
	return 0
}
