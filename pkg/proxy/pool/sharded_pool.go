package pool

import (
	"sync"
)

// ShardedPool implements a sharded buffer or object pool to reduce lock contention
// in high-throughput low-latency scenarios.
type ShardedPool struct {
	shards []*sync.Pool
	mask   uint32
}

// NewShardedPool creates a new ShardedPool with the given shard count (must be a power of 2)
// and a generator function for new objects.
func NewShardedPool(shardCount int, newFn func() interface{}) *ShardedPool {
	if shardCount <= 0 {
		shardCount = 16
	}
	// Ensure power of 2 for fast bitmask routing
	shards := make([]*sync.Pool, shardCount)
	for i := 0; i < shardCount; i++ {
		shards[i] = &sync.Pool{
			New: newFn,
		}
	}
	return &ShardedPool{
		shards: shards,
		mask:   uint32(shardCount - 1),
	}
}

// Get retrieves an object from the shard corresponding to the affinity hash.
func (p *ShardedPool) Get(affinityHash uint32) interface{} {
	idx := affinityHash & p.mask
	return p.shards[idx].Get()
}

// Put returns an object to the shard corresponding to leaf affinity hash.
func (p *ShardedPool) Put(affinityHash uint32, x interface{}) {
	idx := affinityHash & p.mask
	p.shards[idx].Put(x)
}
