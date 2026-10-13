package pool

import (
	"net"
	"testing"
)

func TestShardedPool_AcquireRelease(t *testing.T) {
	p := NewShardedPool()
	dialer := func() (net.Conn, error) { return &net.TCPConn{}, nil }

	conn, err := p.Acquire(dialer)
	if err != nil || conn == nil {
		t.Fatal("failed to acquire")
	}
	p.Release(conn)
}

func TestShardedPool_Race(t *testing.T) {
	p := NewShardedPool()
	dialer := func() (net.Conn, error) { return &net.TCPConn{}, nil }

	for i := 0; i < 100; i++ {
		go func() {
			conn, _ := p.Acquire(dialer)
			p.Release(conn)
		}()
	}
}
