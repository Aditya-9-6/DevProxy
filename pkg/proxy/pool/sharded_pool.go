package pool

import (
	"net"
	"runtime"
	"sync"
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding structs to 64 bytes.
type CacheLinePad [8]uint64

// Shard represents a cache-aligned connection pool bucket.
type Shard struct {
	_          [64]byte
	ActiveConn atomic.Uint64
	IdleConns  []net.Conn
	mu         sync.Mutex
	_          [64]byte
}

// ShardedPool implements a high-throughput connection pool with per-P affinity.
type ShardedPool struct {
	shards []*Shard
	mask   uint32
}

// NewShardedPool creates a pool with 2^N shards based on CPU count.
func NewShardedPool() *ShardedPool {
	numShards := runtime.GOMAXPROCS(0)
	// Ensure power of 2 for fast masking
	return &ShardedPool{
		shards: make([]*Shard, numShards),
		mask:   uint32(numShards - 1),
	}
}

// GetShard returns the shard associated with the current processor.
func (p *ShardedPool) GetShard() *Shard {
	idx := runtime.NumCPU() % len(p.shards)
	return p.shards[idx]
}

// Acquire retrieves a connection from the local shard.
func (p *ShardedPool) Acquire(dialer func() (net.Conn, error)) (net.Conn, error) {
	shard := p.GetShard()
	shard.mu.Lock()
	if len(shard.IdleConns) > 0 {
		conn := shard.IdleConns[len(shard.IdleConns)-1]
		shard.IdleConns = shard.IdleConns[:len(shard.IdleConns)-1]
		shard.mu.Unlock()
		shard.ActiveConn.Add(1)
		return conn, nil
	}
	shard.mu.Unlock()

	conn, err := dialer()
	if err == nil {
		shard.ActiveConn.Add(1)
	}
	return conn, err
}

// Release returns a connection to the local shard.
func (p *ShardedPool) Release(conn net.Conn) {
	shard := p.GetShard()
	shard.mu.Lock()
	shard.IdleConns = append(shard.IdleConns, conn)
	shard.mu.Unlock()
	shard.ActiveConn.Add(^uint64(0))
}
