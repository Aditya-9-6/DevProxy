package security

import (
	"net"
)

// BlacklistManager manages the upstream peer blacklist.
type BlacklistManager struct {
	filter *BloomFilter
}

// NewBlacklistManager creates a new manager with a thread-safe Bloom filter.
func NewBlacklistManager() *BlacklistManager {
	return &BlacklistManager{
		filter: NewBloomFilter(65536, 4),
	}
}

// IsBlacklisted returns true if the IP is present in the Bloom filter.
func (m *BlacklistManager) IsBlacklisted(ip net.IP) bool {
	if ip == nil {
		return false
	}
	ip16 := ip.To16()
	if ip16 == nil {
		return false
	}
	return m.filter.Contains(ip16)
}

// AddToBlacklist adds an IP address to the Bloom filter.
func (m *BlacklistManager) AddToBlacklist(ip net.IP) {
	if ip == nil {
		return
	}
	ip16 := ip.To16()
	if ip16 == nil {
		return
	}
	m.filter.Add(ip16)
}
