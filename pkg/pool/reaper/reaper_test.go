package reaper

import (
	"context"
	"net"
	"testing"
	"time"
)

type mockConn struct {
	net.Conn
	closed bool
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func TestReaper(t *testing.T) {
	t.Parallel()

	t.Run("ReapsExpiredConnection", func(t *testing.T) {
		r := NewReaper(10 * time.Millisecond)
		conn := &mockConn{}
		hazard := &HazardPointer{}
		r.Register("test", conn, hazard)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		go r.Start(ctx, 5*time.Millisecond)
		time.Sleep(30 * time.Millisecond)

		if !conn.closed {
			t.Error("expected connection to be closed")
		}
	})

	t.Run("DoesNotReapActiveHazard", func(t *testing.T) {
		r := NewReaper(10 * time.Millisecond)
		conn := &mockConn{}
		hazard := &HazardPointer{}
		hazard.Set(1) // Mark active
		r.Register("test", conn, hazard)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		go r.Start(ctx, 5*time.Millisecond)
		time.Sleep(30 * time.Millisecond)

		if conn.closed {
			t.Error("expected connection to remain open due to hazard pointer")
		}
	})
}
