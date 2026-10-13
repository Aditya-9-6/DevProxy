package pool

import (
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	time "time"
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

func (p *ShardedPool) GetShard() *Shard {
	idx := p.counter.Add(1) & p.mask
	return p.shards[idx]
}

func isConnClosed(conn net.Conn) bool {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetReadDeadline(time.Now().Add(time.Millisecond))
		var one [1]byte
		n, err := tcpConn.Read(one[:])
		_ = tcpConn.SetReadDeadline(time.Time{})
		if n == 0 && err != nil {
			return true
		}
	}
	return false
}

func (p *ShardedPool) Acquire(dialer func() (net.Conn, error)) (net.Conn, error) {
	shard := p.GetShard()
	shard.mu.Lock()
	for len(shard.IdleConns) > 0 {
		conn := shard.IdleConns[len(shard.IdleConns)-1]
		shard.IdleConns = shard.IdleConns[:len(shard.IdleConns)-1]
		shard.mu.Unlock()
		if isConnClosed(conn) {
			conn.Close()
			continue
		}
		shard.ActiveConn.Add(1)
		return conn, nil
	}
	shard.mu.Unlock()

	conn, err := dialer()
	if err == nil && conn != nil {
		shard.ActiveConn.Add(1)
	}
	return conn, err
}

func (p *ShardedPool) Release(conn net.Conn) {
	if conn == nil {
		return
	}
	if isConnClosed(conn) {
		conn.Close()
		return
	}
	shard := p.GetShard()
	shard.mu.Lock()
	if len(shard.IdleConns) < MaxIdlePerShard {
		shard.IdleConns = append(shard.IdleConns, conn)
		shard.mu.Unlock()
	} else {
		shard.mu.Unlock()
		conn.Close()
	}
	shard.ActiveConn.Add(^uint64(0))
}
