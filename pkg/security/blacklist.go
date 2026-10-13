package security

import (
	"net"
	"sync"
)

// BlacklistManager manages the upstream peer blacklist.
type BlacklistManager struct {
	filter *BloomFilter
	mu     sync.RWMutex
}

func NewBlacklistManager() *BlacklistManager {
	return &BlacklistManager{
		filter: NewBloomFilter(65536, 4),
	}
}

func (m *BlacklistManager) IsBlacklisted(ip net.IP) bool {
	return m.filter.Contains(ip.To16())
}

func (m *BlacklistManager) AddToBlacklist(ip net.IP) {
	m.filter.Add(ip.To16())
}
