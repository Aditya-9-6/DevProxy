package pool

import (
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

const MaxIdlePerShard = 128

type Shard struct {
	_          [64]byte
	ActiveConn atomic.Uint64
	IdleConns  []net.Conn
	mu         sync.Mutex
	_          [64]byte
}

type ShardedPool struct {
	shards []*Shard
	mask   uint32
}

func NewShardedPool() *ShardedPool {
	numShards := 1
	procs := runtime.GOMAXPROCS(0)
	for numShards < procs {
		numShards <<= 1
	}
	shards := make([]*Shard, numShards)
	for i := range shards {
		shards[i] = &Shard{IdleConns: make([]net.Conn, 0, MaxIdlePerShard)}
	}
	return &ShardedPool{shards: shards, mask: uint32(numShards - 1)}
}

func (p *ShardedPool) GetShard() *Shard {
	var x int
	ptr := unsafe.Pointer(&x)
	idx := (uint32(uintptr(ptr)) >> 6) & p.mask
	return p.shards[idx]
}

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

func (p *ShardedPool) Release(conn net.Conn) {
	shard := p.GetShard()
	shard.mu.Lock()
	if len(shard.IdleConns) < MaxIdlePerShard {
		shard.IdleConns = append(shard.IdleConns, conn)
	} else {
		conn.Close()
	}
	shard.mu.Unlock()
	shard.ActiveConn.Add(^uint64(0))
}
