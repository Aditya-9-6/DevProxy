package doh

import (
	"sync"
	"time"
)

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

// Cache provides a thread-safe, size-limited store.
type Cache struct {
	mu    sync.RWMutex
	items map[string]cacheEntry
	limit int
}

func NewCache() *Cache {
	return &Cache{
		items: make(map[string]cacheEntry),
		limit: 1000,
	}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.items[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

func (c *Cache) Set(key string, data []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.items) >= c.limit {
		for k := range c.items {
			delete(c.items, k)
			break
		}
	}
	c.items[key] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
}
