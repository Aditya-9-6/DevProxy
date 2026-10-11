package dns

import (
	"context"
	"sync"
	"time"
)

type cacheEntry struct {
	ips    []string
	expiry time.Time
}

type DNSCache struct {
	data   sync.Map
	cancel context.CancelFunc
}

func NewDNSCache() *DNSCache {
	ctx, cancel := context.WithCancel(context.Background())
	c := &DNSCache{
		cancel: cancel,
	}
	go c.cleanupLoop(ctx)
	return c
}

func (c *DNSCache) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *DNSCache) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.data.Range(func(key, value interface{}) bool {
				if entry, ok := value.(*cacheEntry); ok && time.Now().After(entry.expiry) {
					c.data.Delete(key)
				}
				return true
			})
		}
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
