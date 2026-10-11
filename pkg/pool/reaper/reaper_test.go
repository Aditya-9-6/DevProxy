package reaper

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

type mockConn struct {
	net.Conn
	closed bool
	mu     sync.Mutex
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	return nil
}

func (m *mockConn) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func TestReaper(t *testing.T) {
	t.Parallel()

	t.Run("ReapsExpiredConnection", func(t *testing.T) {
		r := NewReaper(2 * time.Millisecond)
		conn := &mockConn{}
		hazard := &HazardPointer{}
		r.Register("test", conn, hazard)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go r.Start(ctx, 1*time.Millisecond)

		for i := 0; i < 200; i++ {
			if conn.IsClosed() {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Error("expected connection to be closed")
	})

	t.Run("DoesNotReapActiveHazard", func(t *testing.T) {
		r := NewReaper(2 * time.Millisecond)
		conn := &mockConn{}
		hazard := &HazardPointer{}
		hazard.Set(1)
		r.Register("test", conn, hazard)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go r.Start(ctx, 1*time.Millisecond)
		time.Sleep(30 * time.Millisecond)

		if conn.IsClosed() {
			t.Error("expected connection to remain open due to hazard pointer")
		}
	})

	t.Run("ConcurrentStress", func(t *testing.T) {
		r := NewReaper(1 * time.Millisecond)
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				h := &HazardPointer{}
				h.Set(uintptr(idx))
				r.Register("id", &mockConn{}, h)
				h.Clear()
			}(i)
		}
		wg.Wait()
	})
}
