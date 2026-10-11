package proxy

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	time "time"
)

func TestConnectionShard(t *testing.T) {
	shard := NewConnectionShard(2)
	defer shard.Close()

	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		shard.Dispatch(func() {
			wg.Done()
		})
	}
	wg.Wait()
}

func TestConnectionShardPanicRecovery(t *testing.T) {
	shard := NewConnectionShard(1)
	defer shard.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	// First task panics
	shard.Dispatch(func() {
		defer wg.Done()
		panic("simulated worker panic")
	})

	// Second task should execute despite the first panicking
	shard.Dispatch(func() {
		defer wg.Done()
	})

	wg.Wait()
}

func TestConnectionShardGracefulShutdown(t *testing.T) {
	shard := NewConnectionShard(2)

	var executed atomic.Int32
	var wg sync.WaitGroup
	wg.Add(5)

	for i := 0; i < 5; i++ {
		shard.Dispatch(func() {
			defer wg.Done()
			executed.Add(1)
		})
	}

	shard.Close()
	wg.Wait()

	if executed.Load() != 5 {
		t.Errorf("Expected 5 tasks to drain on close, got %d", executed.Load())
	}
}

// mockConn implements net.Conn for testing Scheduler context cancellation
type mockConn struct {
	net.Conn
	closed bool
	mu     sync.Mutex
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockConn) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func TestConnectionSchedulerContextCancellation(t *testing.T) {
	shard := NewConnectionShard(1)
	defer shard.Close()

	scheduler := NewConnectionScheduler(shard)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel

	conn := &mockConn{}
	var handlerCalled atomic.Bool

	scheduler.ScheduleConnection(ctx, conn, func(c net.Conn) {
		handlerCalled.Store(true)
	})

	time.Sleep(50 * time.Millisecond)

	if handlerCalled.Load() {
		t.Error("Handler should not have been called due to canceled context")
	}
	if !conn.IsClosed() {
		t.Error("Connection should have been closed due to canceled context")
	}
}
