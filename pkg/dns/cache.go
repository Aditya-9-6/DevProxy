package dns

import (
	"sync"
	"sync/atomic"
	"time"
)

type cacheEntry struct {
	ips    []string
	expiry atomic.Int64
	stale  atomic.Bool
}

const shardCount = 32

type DNSCache struct {
	shards [shardCount]*shard
}

type shard struct {
	mu   sync.RWMutex
	data map[string]*cacheEntry
}

func NewDNSCache() *DNSCache {
	c := &DNSCache{}
	for i := 0; i < shardCount; i++ {
		c.shards[i] = &shard{data: make(map[string]*cacheEntry)}
	}
	return c
}

func (c *DNSCache) Close() {
	// No-op for sharded cache, satisfies test cleanup requirements
}

func (c *DNSCache) getShard(host string) *shard {
	hash := 0
	for i := 0; i < len(host); i++ {
		hash = 31*hash + int(host[i])
	}
	return c.shards[uint(hash)%shardCount]
}

func (c *DNSCache) Get(host string) ([]string, bool) {
	s := c.getShard(host)
	s.mu.RLock()
	entry, ok := s.data[host]
	s.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if time.Now().UnixNano() > entry.expiry.Load() {
		entry.stale.Store(true)
		return nil, false
	}

	return entry.ips, true
}

func (c *DNSCache) Set(host string, ips []string, ttl time.Duration) {
	s := c.getShard(host)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[host] = &cacheEntry{
		ips:    ips,
		expiry: *new(atomic.Int64),
	}
	s.data[host].expiry.Store(time.Now().Add(ttl).UnixNano())
}
