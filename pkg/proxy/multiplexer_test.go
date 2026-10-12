package proxy

import (
	"net"
	"sync"
	"testing"
	"time"
)

func TestMultiplexer(t *testing.T) {
	m := NewMultiplexer()
	defer m.Close()
	addr := "127.0.0.1:8080"
	conn, _ := net.Pipe()
	m.Put(addr, conn)
	got, ok := m.Get(addr)
	if !ok || got != conn {
		t.Errorf("failed to retrieve connection")
	}
	m.Remove(addr)
	_, ok = m.Get(addr)
	if ok {
		t.Errorf("connection should have been removed")
	}
}

func TestMultiplexerReaperCancellation(t *testing.T) {
	m := NewMultiplexer()
	m.Close()
}

func TestMultiplexerTTL(t *testing.T) {
	m := NewMultiplexer()
	m.TTL = 10 * time.Millisecond
	defer m.Close()
	addr := "127.0.0.1:8081"
	conn, _ := net.Pipe()
	m.Put(addr, conn)
	time.Sleep(20 * time.Millisecond)
	_, ok := m.Get(addr)
	if ok {
		t.Errorf("connection should have expired")
	}
}

func TestBackpressureShedding(t *testing.T) {
	bp := NewBackpressureController(1)
	if !bp.Acquire() {
		t.Fatal("failed to acquire first slot")
	}
	if bp.Acquire() {
		t.Error("should have shed load")
	}
	bp.Release()
	if !bp.Acquire() {
		t.Error("should have acquired after release")
	}
}

func TestBackpressureConcurrent(t *testing.T) {
	bp := NewBackpressureController(10)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if bp.Acquire() {
				bp.Release()
			}
		}()
	}
	wg.Wait()
	if bp.activeRequests.Load() != 0 {
		t.Errorf("leaked requests: %d", bp.activeRequests.Load())
	}
}

func BenchmarkBackpressure(b *testing.B) {
	bp := NewBackpressureController(1000)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if bp.Acquire() {
				bp.Release()
			}
		}
	})
}
