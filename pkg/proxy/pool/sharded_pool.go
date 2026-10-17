package pool

import (
	"net"
	"runtime"
	"sync"
	"sync/atomic"
)

const MaxIdlePerShard = 128

type pooledConn struct {
	net.Conn
	shardIdx uint32
}

type Shard struct {
	_          [64]byte
	ActiveConn atomic.Uint64
	IdleConns  []net.Conn
	mu         sync.Mutex
	_          [64]byte
}

type ShardedPool struct {
	shards  []*Shard
	mask    uint32
	counter atomic.Uint32
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

func (p *ShardedPool) getShardIndex() uint32 {
	return p.counter.Add(1) & p.mask
}

func (p *ShardedPool) getShard(idx uint32) *Shard {
	return p.shards[idx]
}

func (p *ShardedPool) Acquire(dialer func() (net.Conn, error)) (net.Conn, error) {
	idx := p.getShardIndex()
	shard := p.getShard(idx)

	conn, err := dialer()
	if err == nil && conn != nil {
		shard.mu.Lock()
		defer shard.mu.Unlock()

		shard.ActiveConn.Add(1)
		return &pooledConn{Conn: conn, shardIdx: idx}, nil
	}

	return conn, err
}

func (p *ShardedPool) Release(conn net.Conn) {
	if conn == nil {
		return
	}

	var idx uint32
	var rawConn net.Conn

	if pc, ok := conn.(*pooledConn); ok {
		idx = pc.shardIdx
		rawConn = pc.Conn
	} else {
		rawConn = conn
		idx = p.getShardIndex()
	}

	shard := p.getShard(idx)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if len(shard.IdleConns) < MaxIdlePerShard {
		shard.IdleConns = append(shard.IdleConns, rawConn)
	} else {
		rawConn.Close()
	}
	shard.ActiveConn.Add(^uint64(0))
}
