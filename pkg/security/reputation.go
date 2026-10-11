package security

import "net"

// ReputationEngine manages IP blocklists using a Bloom filter.
type ReputationEngine struct {
	filter *BloomFilter
}

func NewReputationEngine(capacity uint64) *ReputationEngine {
	return &ReputationEngine{
		filter: NewBloomFilter(capacity, 5),
	}
}

func (e *ReputationEngine) BlockIP(ip string) {
	e.filter.Add([]byte(ip))
}

func (e *ReputationEngine) IsBlocked(ip string) bool {
	host, _, err := net.SplitHostPort(ip)
	if err == nil {
		ip = host
	}
	return e.filter.Contains([]byte(ip))
}
