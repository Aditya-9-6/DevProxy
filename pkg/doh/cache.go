package doh

import (
	"sync"
	"time"
)

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

// Cache provides a thread-safe, zero-allocation-friendly response store.
type Cache struct {
	mu    sync.RWMutex
	items map[string]cacheEntry
	pool  sync.Pool
}

func NewCache() *Cache {
	return &Cache{
		items: make(map[string]cacheEntry),
		pool: sync.Pool{
			New: func() interface{} { return make([]byte, 512) },
		},
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
	c.items[key] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
}
