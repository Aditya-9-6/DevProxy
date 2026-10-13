package dns

import (
	"context"
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
	cancel context.CancelFunc
}

type shard struct {
	mu   sync.RWMutex
	data map[string]*cacheEntry
}

func NewDNSCache() *DNSCache {
	ctx, cancel := context.WithCancel(context.Background())
	c := &DNSCache{
		cancel: cancel,
	}
	for i := 0; i < shardCount; i++ {
		c.shards[i] = &shard{data: make(map[string]*cacheEntry)}
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
			now := time.Now().UnixNano()
			for _, s := range c.shards {
				s.mu.Lock()
				for k, entry := range s.data {
					if now > entry.expiry.Load()+int64(30*time.Minute) {
						delete(s.data, k)
					}
				}
				s.mu.Unlock()
			}
		}
	}
}

func (c *DNSCache) getShard(host string) *shard {
	hash := 0
	for i := 0; i < len(host); i++ {
		hash = 31*hash + int(host[i])
	}
	return c.shards[uint(hash)%shardCount]
}

func (c *DNSCache) Get(host string) ([]string, bool, bool) {
	s := c.getShard(host)
	s.mu.RLock()
	entry, ok := s.data[host]
	s.mu.RUnlock()

	if !ok {
		return nil, false, false
	}

	expired := time.Now().UnixNano() > entry.expiry.Load()
	if expired {
		return entry.ips, true, true
	}

	return entry.ips, false, true
}

func (c *DNSCache) Set(host string, ips []string, ttl time.Duration) {
	s := c.getShard(host)
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.data[host]
	if !exists {
		entry = &cacheEntry{}
		s.data[host] = entry
	}
	entry.ips = ips
	entry.expiry.Store(time.Now().Add(ttl).UnixNano())
	entry.stale.Store(false)
}
