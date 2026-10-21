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
	if est < 100 {
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

func TestCountMinSketch_NonPositiveCountsIgnored(t *testing.T) {
	cms := NewCountMinSketch(100, 4)
	key := []byte("signed-count")

	cms.Add(key, -10)
	cms.Add(key, 0)
	if est := cms.Estimate(key); est != 0 {
		t.Fatalf("expected non-positive counts to be ignored, got %d", est)
	}

	cms.Add(key, 7)
	if est := cms.Estimate(key); est != 7 {
		t.Errorf("expected estimate 7, got %d", est)
	}
}

func TestCountMinSketch_MaxPositiveCount(t *testing.T) {
	cms := NewCountMinSketch(100, 4)
	key := []byte("max-count")
	const maxPositiveInt64 = 1<<63 - 1

	cms.Add(key, maxPositiveInt64)
	if est := cms.Estimate(key); est != maxPositiveInt64 {
		t.Errorf("expected estimate %d, got %d", maxPositiveInt64, est)
	}
}

func TestCountMinSketch_ZeroWidthDepth(t *testing.T) {
	cms := NewCountMinSketch(0, 0)
	cms.Add([]byte("key"), 5)
	if est := cms.Estimate([]byte("key")); est != 5 {
		t.Errorf("expected 5, got %d", est)
	}
}

func TestCountMinSketch_ZeroValue(t *testing.T) {
	var cms CountMinSketch
	cms.Add([]byte("key"), 5)
	if est := cms.Estimate([]byte("key")); est != 0 {
		t.Errorf("expected zero-value sketch to remain empty, got %d", est)
	}
}

func TestCountMinSketch_NilReceiver(t *testing.T) {
	var cms *CountMinSketch
	cms.Add([]byte("key"), 5)
	cms.Reset()
	if est := cms.Estimate([]byte("key")); est != 0 {
		t.Errorf("expected nil receiver estimate 0, got %d", est)
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

	deadline := time.Now().Add(time.Second)
	for sw.Estimate(key) != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if est := sw.Estimate(key); est != 0 {
		t.Errorf("expected 0 after window reset, got %d", est)
	}
}

func TestSlidingWindowCountMinSketch_Concurrent(t *testing.T) {
	window := 5 * time.Second
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

	if est := sw.Estimate([]byte("concurrent")); est == 0 {
		t.Error("expected non-zero estimate")
	}
}

func TestSlidingWindowCountMinSketch_StopWithoutStart(t *testing.T) {
	sw := NewSlidingWindowCountMinSketch(100, 4, time.Second)
	sw.Stop()
}

func TestSlidingWindowCountMinSketch_ContextCancellation(t *testing.T) {
	sw := NewSlidingWindowCountMinSketch(100, 4, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	sw.Start(ctx)
	cancel()
	sw.Stop()
	sw.Stop()
}

func TestSlidingWindowCountMinSketch_StartIsIdempotent(t *testing.T) {
	sw := NewSlidingWindowCountMinSketch(100, 4, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	sw.Start(ctx)
	sw.Start(ctx)
	defer func() {
		cancel()
		sw.Stop()
	}()

	key := []byte("idempotent-start")
	sw.Add(key, 3)
	if est := sw.Estimate(key); est != 3 {
		t.Errorf("expected estimate 3, got %d", est)
	}
}

func TestSlidingWindowCountMinSketch_InvalidWindow(t *testing.T) {
	for _, window := range []time.Duration{0, -time.Second} {
		sw := NewSlidingWindowCountMinSketch(100, 4, window)
		ctx, cancel := context.WithCancel(context.Background())
		sw.Start(ctx)
		cancel()
		sw.Stop()
	}
}

func TestSlidingWindowCountMinSketch_ConcurrentLifecycle(t *testing.T) {
	sw := NewSlidingWindowCountMinSketch(100, 4, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sw.Start(ctx)
			sw.Stop()
		}()
	}
	wg.Wait()
	sw.Stop()
}

func TestSlidingWindowCountMinSketch_RestartAfterStop(t *testing.T) {
	sw := NewSlidingWindowCountMinSketch(100, 4, time.Hour)
	sw.Start(context.Background())
	sw.Stop()
	sw.Start(context.Background())
	sw.Stop()
}

func TestSlidingWindowCountMinSketch_ZeroValueAndNilContext(t *testing.T) {
	var sw SlidingWindowCountMinSketch
	sw.Start(nil)
	defer sw.Stop()

	key := []byte("zero-value")
	sw.Add(key, 2)
	if est := sw.Estimate(key); est != 2 {
		t.Errorf("expected estimate 2, got %d", est)
	}
}

func TestSlidingWindowCountMinSketch_NilReceiver(t *testing.T) {
	var sw *SlidingWindowCountMinSketch
	sw.Start(context.Background())
	sw.Stop()
	sw.Add([]byte("key"), 1)
	if est := sw.Estimate([]byte("key")); est != 0 {
		t.Errorf("expected nil receiver estimate 0, got %d", est)
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
		t.Error("expected different hashes for different seeds")
	}
}

func TestMurmurHash64_AllTailLengths(t *testing.T) {
	for length := 0; length < 16; length++ {
		key := make([]byte, length)
		for i := range key {
			key[i] = byte(i*31 + length)
		}

		first := murmurHash64(key, 0)
		second := murmurHash64(key, 0)
		if first != second {
			t.Errorf("length %d: expected deterministic hash, got %d vs %d", length, first, second)
		}
	}
}
