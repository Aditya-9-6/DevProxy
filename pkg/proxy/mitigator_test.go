package proxy

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMitigator_ActiveConnections(t *testing.T) {
	m := NewMitigator()
	if atomic.LoadUint64(&m.ActiveConnections[0]) != 0 {
		t.Fatalf("expected 0 active connections")
	}

	m.TrackConnection()
	m.TrackConnection()
	if atomic.LoadUint64(&m.ActiveConnections[0]) != 2 {
		t.Fatalf("expected 2 active connections, got %d", atomic.LoadUint64(&m.ActiveConnections[0]))
	}

	m.ReleaseConnection()
	if atomic.LoadUint64(&m.ActiveConnections[0]) != 1 {
		t.Fatalf("expected 1 active connection, got %d", atomic.LoadUint64(&m.ActiveConnections[0]))
	}

	m.ReleaseConnection()
	if atomic.LoadUint64(&m.ActiveConnections[0]) != 0 {
		t.Fatalf("expected 0 active connections, got %d", atomic.LoadUint64(&m.ActiveConnections[0]))
	}
}

func TestMitigator_Deadline(t *testing.T) {
	m := NewMitigator()
	if m.IsExpired() {
		t.Fatal("expected IsExpired to be false when deadline is 0")
	}

	// Future deadline
	m.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	if m.IsExpired() {
		t.Fatal("expected IsExpired to be false for future deadline")
	}

	// Past deadline
	m.SetReadDeadline(time.Now().Add(-500 * time.Millisecond))
	if !m.IsExpired() {
		t.Fatal("expected IsExpired to be true for past deadline")
	}
}

func TestMitigator_SlowLorisViolation(t *testing.T) {
	m := NewMitigator()
	m.RecordViolation()
	m.RecordViolation()
	if atomic.LoadUint64(&m.SlowLorisViolations[0]) != 2 {
		t.Fatalf("expected 2 violations, got %d", atomic.LoadUint64(&m.SlowLorisViolations[0]))
	}
}

func TestMitigator_ConcurrentConnections(t *testing.T) {
	m := NewMitigator()
	const concurrency = 50
	const iterations = 500

	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				m.TrackConnection()
				m.RecordViolation()
				m.ReleaseConnection()
			}
		}()
	}
	wg.Wait()

	if val := atomic.LoadUint64(&m.ActiveConnections[0]); val != 0 {
		t.Fatalf("expected 0 active connections, got %d", val)
	}
	expectedViolations := uint64(concurrency * iterations)
	if val := atomic.LoadUint64(&m.SlowLorisViolations[0]); val != expectedViolations {
		t.Fatalf("expected %d violations, got %d", expectedViolations, val)
	}
}
