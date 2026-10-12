package proxy

import (
	"net"
	"sync"
)

// SubnetClassifier uses a Radix Tree (Trie) for O(K) lookup performance.
type SubnetClassifier struct {
	mu   sync.RWMutex
	root *node
}

type node struct {
	children [2]*node
	tier     int
	isLeaf   bool
}

func NewSubnetClassifier() *SubnetClassifier {
	return &SubnetClassifier{root: &node{}}
}

func (s *SubnetClassifier) AddTier(cidr string, tier int) error {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	bits, _ := ipNet.Mask.Size()
	curr := s.root
	for i := 0; i < bits; i++ {
		bit := (ipNet.IP[i/8] >> (7 - (i % 8))) & 1
		if curr.children[bit] == nil {
			curr.children[bit] = &node{}
		}
		curr = curr.children[bit]
	}
	curr.tier = tier
	curr.isLeaf = true
	return nil
}

func (s *SubnetClassifier) Classify(ip net.IP) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	curr := s.root
	bestTier := 0
	ipV4 := ip.To4()
	if ipV4 != nil {
		ip = ipV4
	}

	for i := 0; i < len(ip)*8; i++ {
		if curr.isLeaf {
			bestTier = curr.tier
		}
		bit := (ip[i/8] >> (7 - (i % 8))) & 1
		if curr.children[bit] == nil {
			break
		}
		curr = curr.children[bit]
	}
	if curr.isLeaf {
		bestTier = curr.tier
	}
	return bestTier
}
