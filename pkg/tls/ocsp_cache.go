package tls

import (
	"sync"
)

// OCSPCache provides a thread-safe, lock-free lookup for OCSP staples.
type OCSPCache struct {
	mu    sync.RWMutex
	items map[string]*StapleEntry
}

func NewOCSPCache() *OCSPCache {
	return &OCSPCache{items: make(map[string]*StapleEntry)}
}

func (c *OCSPCache) Get(host string) *OCSPStaple {
	c.mu.RLock()
	entry, ok := c.items[host]
	c.mu.RUnlock()
	if !ok {
		return nil
	}
	return entry.Staple.Load()
}

func (c *OCSPCache) Set(host string, staple *OCSPStaple) {
	c.mu.Lock()
	entry, ok := c.items[host]
	if !ok {
		entry = &StapleEntry{}
		c.items[host] = entry
	}
	c.mu.Unlock()
	entry.Staple.Store(staple)
}
