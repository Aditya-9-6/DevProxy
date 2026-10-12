package security

import (
	"net"
	"sync"
	"testing"
)

func TestCountMinSketch(t *testing.T) {
	s := NewCountMinSketch()
	defer s.Close()
	key := []byte("192.168.1.1")
	s.Increment(key)
	s.Increment(key)
	if s.Estimate(key) < 2 {
		t.Errorf("Expected count >= 2, got %d", s.Estimate(key))
	}
}

func TestCountMinSketchConcurrent(t *testing.T) {
	s := NewCountMinSketch()
	defer s.Close()
	key := []byte("192.168.1.1")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Increment(key)
		}()
	}
	wg.Wait()
	if s.Estimate(key) < 100 {
		t.Errorf("Expected count >= 100, got %d", s.Estimate(key))
	}
}

func TestSlowlorisFilter(t *testing.T) {
	filter := NewSlowlorisFilter(5)
	defer filter.Close()
	addr := &net.TCPAddr{IP: net.ParseIP("1.2.3.4"), Port: 80}
	conn := &mockConn{remoteAddr: addr}

	for i := 0; i < 10; i++ {
		blocked := filter.ShouldBlock(conn)
		if i >= 5 && !blocked {
			t.Errorf("Expected block at iteration %d", i)
		}
	}
}

type mockConn struct {
	net.Conn
	remoteAddr net.Addr
}

func (m *mockConn) RemoteAddr() net.Addr { return m.remoteAddr }
