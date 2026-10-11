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
	c := &DNSCache{}
	go c.cleanupLoop()
	return c
}

func (c *DNSCache) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		c.data.Range(func(key, value interface{}) bool {
			if entry, ok := value.(*cacheEntry); ok && time.Now().After(entry.expiry) {
				c.data.Delete(key)
			}
			return true
		})
	}
}

func (c *DNSCache) Get(host string) ([]string, bool) {
	val, ok := c.data.Load(host)
	if !ok {
		return nil, false
	}
	entry := val.(*cacheEntry)
	if time.Now().After(entry.expiry) {
		c.data.Delete(host)
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
