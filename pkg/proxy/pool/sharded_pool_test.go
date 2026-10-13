package pool

import (
	"net"
	"sync"
	"testing"
)

func TestShardedPool_AcquireRelease(t *testing.T) {
	p := NewShardedPool()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			c, err := listener.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()

	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", listener.Addr().String())
	}

	conn, err := p.Acquire(dialer)
	if err != nil || conn == nil {
		t.Fatal("failed to acquire")
	}
	p.Release(conn)
}

func TestShardedPool_Race(t *testing.T) {
	p := NewShardedPool()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			c, err := listener.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()

	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", listener.Addr().String())
	}

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := p.Acquire(dialer)
			if err == nil && conn != nil {
				p.Release(conn)
			}
		}()
	}
	wg.Wait()
}

func TestShardedPool_Distribution(t *testing.T) {
	p := NewShardedPool()
	shardsHit := make(map[int]bool)
	for i := 0; i < 1000; i++ {
		idx := p.getShardIndex()
		shardsHit[int(idx)] = true
	}
	if len(shardsHit) <= 1 && len(p.shards) > 1 {
		t.Errorf("expected load to be distributed across multiple shards, got %d shard(s) hit", len(shardsHit))
	}
}
