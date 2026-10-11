package dns

import (
	"sync"
	"time"
)

type cacheEntry struct {
	ips    []string
	expiry time.Time
}

type DNSCache struct {
	data sync.Map
}

func NewDNSCache() *DNSCache {
	return &DNSCache{}
}

func (c *DNSCache) Get(host string) ([]string, bool) {
	val, ok := c.data.Load(host)
	if !ok {
		return nil, false
	}
	entry := val.(*cacheEntry)
	if time.Now().After(entry.expiry) {
		return nil, false
	}
	return entry.ips, true
}

func (c *DNSCache) Set(host string, ips []string, ttl time.Duration) {
	c.data.Store(host, &cacheEntry{
		ips:    ips,
		expiry: time.Now().Add(ttl),
	})
}
