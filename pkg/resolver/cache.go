package resolver

import (
	"sync/atomic"
	"time"
)

const ShardCount = 32

type CacheEntry struct {
	IPs      []string
	Expiry   int64
	NextIdx  atomic.Uint64
	IsActive atomic.Bool
}

type CacheShard struct {
	data atomic.Pointer[map[string]*CacheEntry]
}

type DNSCache struct {
	shards [ShardCount]*CacheShard
}

func NewDNSCache() *DNSCache {
	c := &DNSCache{}
	for i := 0; i < ShardCount; i++ {
		c.shards[i] = &CacheShard{}
		c.shards[i].data.Store(&map[string]*CacheEntry{})
	}
	return c
}

func (c *DNSCache) getShard(host string) *CacheShard {
	hash := uint32(0)
	for i := 0; i < len(host); i++ {
		hash = 31*hash + uint32(host[i])
	}
	return c.shards[hash%ShardCount]
}

func (c *DNSCache) Get(host string) ([]string, bool) {
	shard := c.getShard(host)
	m := shard.data.Load()
	entry, ok := (*m)[host]
	if !ok || entry == nil {
		return nil, false
	}
	if time.Now().UnixNano() > atomic.LoadInt64(&entry.Expiry) {
		return nil, false
	}
	return entry.IPs, true
}

func (c *DNSCache) Set(host string, ips []string, ttl time.Duration) {
	shard := c.getShard(host)
	entry := &CacheEntry{
		IPs:    ips,
		Expiry: time.Now().Add(ttl).UnixNano(),
	}
	entry.IsActive.Store(true)

	for {
		oldMap := shard.data.Load()
		newMap := make(map[string]*CacheEntry, len(*oldMap)+1)
		for k, v := range *oldMap {
			newMap[k] = v
		}
		newMap[host] = entry
		if shard.data.CompareAndSwap(oldMap, &newMap) {
			break
		}
	}
}
