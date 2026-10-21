package metrics

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestCountMinSketch_AddAndEstimate(t *testing.T) {
	cms := NewCountMinSketch(100, 4)
	key := []byte("test-key")
	cms.Add(key, 10)
	if est := cms.Estimate(key); est != 10 {
		t.Errorf("expected estimate 10, got %d", est)
	}
}

func TestCountMinSketch_MultipleKeys(t *testing.T) {
	cms := NewCountMinSketch(100, 4)
	keys := []string{"a", "b", "c", "d"}
	for _, k := range keys {
		cms.Add([]byte(k), 5)
	}
	for _, k := range keys {
		if est := cms.Estimate([]byte(k)); est < 5 {
			t.Errorf("key %s: expected estimate >= 5, got %d", k, est)
		}
	}
	// Check a key not added
	if est := cms.Estimate([]byte("nonexistent")); est != 0 {
		t.Errorf("expected 0 for nonexistent key, got %d", est)
	}
}

func TestCountMinSketch_ConcurrentAdds(t *testing.T) {
	cms := NewCountMinSketch(100, 4)
	key := []byte("concurrent-key")
	var wg sync.WaitGroup
	goroutines := 100
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cms.Add(key, 1)
		}()
	}
	wg.Wait()
	est := cms.Estimate(key)
	if est < uint64(goroutines) {
		t.Errorf("expected estimate >= %d, got %d", goroutines, est)
	}
}

func TestCountMinSketch_Reset(t *testing.T) {
	cms := NewCountMinSketch(100, 4)
	cms.Add([]byte("key"), 10)
	if est := cms.Estimate([]byte("key")); est != 10 {
		t.Fatalf("before reset: expected 10, got %d", est)
	}
	cms.Reset()
	if est := cms.Estimate([]byte("key")); est != 0 {
		t.Errorf("after reset: expected 0, got %d", est)
	}
}

func TestCountMinSketch_LargeKey(t *testing.T) {
	cms := NewCountMinSketch(100, 4)
	key := make([]byte, 1024)
	for i := range key {
		key[i] = byte(i % 256)
	}
	cms.Add(key, 42)
	if est := cms.Estimate(key); est != 42 {
		t.Errorf("expected 42, got %d", est)
	}
}

func TestCountMinSketch_TableDriven(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		count int64
		want  uint64
	}{
		{"single", "a", 1, 1},
		{"multiple", "b", 5, 5},
		{"zero", "c", 0, 0},
		{"large", "d", 100, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cms := NewCountMinSketch(100, 4)
			cms.Add([]byte(tt.key), tt.count)
			got := cms.Estimate([]byte(tt.key))
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSlidingWindowCountMinSketch_Basic(t *testing.T) {
	window := 50 * time.Millisecond
	sw := NewSlidingWindowCountMinSketch(100, 4, window)
	ctx, cancel := context.WithCancel(context.Background())
	sw.Start(ctx)
	defer func() {
		cancel()
		sw.Stop()
	}()

	key := []byte("sw-key")
	sw.Add(key, 10)
	if est := sw.Estimate(key); est != 10 {
		t.Errorf("expected 10, got %d", est)
	}

	// Wait for window to pass (plus some buffer)
	time.Sleep(window * 2)

	// After window, the sketch should have been reset
	if est := sw.Estimate(key); est != 0 {
		t.Errorf("expected 0 after window reset, got %d", est)
	}
}

func TestSlidingWindowCountMinSketch_Concurrent(t *testing.T) {
	window := 100 * time.Millisecond
	sw := NewSlidingWindowCountMinSketch(100, 4, window)
	ctx, cancel := context.WithCancel(context.Background())
	sw.Start(ctx)
	defer func() {
		cancel()
		sw.Stop()
	}()

	var wg sync.WaitGroup
	goroutines := 50
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				sw.Add([]byte("concurrent"), 1)
				time.Sleep(time.Millisecond)
			}
		}()
	}
	wg.Wait()
	// Estimate should be at least something
	est := sw.Estimate([]byte("concurrent"))
	if est == 0 {
		t.Errorf("expected non-zero estimate")
	}
}

func TestSlidingWindowCountMinSketch_StopWithoutStart(t *testing.T) {
	sw := NewSlidingWindowCountMinSketch(100, 4, time.Second)
	// Stop should not panic even if Start was not called
	sw.Stop()
}

func TestSlidingWindowCountMinSketch_ContextCancellation(t *testing.T) {
	sw := NewSlidingWindowCountMinSketch(100, 4, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	sw.Start(ctx)
	cancel()
	// Wait a bit for goroutine to exit
	time.Sleep(50 * time.Millisecond)
	// Should not leak; we can call Stop again safely.
	sw.Stop()
}

func TestCountMinSketch_ZeroWidthDepth(t *testing.T) {
	// Should default to 1
	cms := NewCountMinSketch(0, 0)
	cms.Add([]byte("key"), 5)
	if est := cms.Estimate([]byte("key")); est != 5 {
		t.Errorf("expected 5, got %d", est)
	}
}

func TestMurmurHash64_Deterministic(t *testing.T) {
	key := []byte("deterministic")
	h1 := murmurHash64(key, 0)
	h2 := murmurHash64(key, 0)
	if h1 != h2 {
		t.Errorf("expected same hash, got %d vs %d", h1, h2)
	}
	h3 := murmurHash64(key, 1)
	if h1 == h3 {
		t.Errorf("expected different hashes for different seeds")
	}
}
