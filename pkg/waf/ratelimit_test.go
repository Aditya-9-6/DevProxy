package waf

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestTokenBucket_Allow(t *testing.T) {
	t.Run("BasicBurst", func(t *testing.T) {
		rate := uint64(1000)
		burst := uint64(5)
		tb := NewTokenBucket(rate, burst)

		for i := uint64(0); i < burst; i++ {
			if !tb.Allow() {
				t.Errorf("expected token at index %d", i)
			}
		}

		if tb.Allow() {
			t.Error("expected bucket to be empty")
		}
	})

	t.Run("EdgeCases", func(t *testing.T) {
		tb := NewTokenBucket(1, 0)
		if tb.Allow() {
			t.Error("expected zero burst to deny")
		}
	})
}

func TestTokenBucket_Concurrent(t *testing.T) {
	rate := uint64(10000)
	burst := uint64(100)
	tb := NewTokenBucket(rate, burst)
	workers := 10
	iterations := 100
	var allowedCount uint64
	var wg sync.WaitGroup

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if tb.Allow() {
					atomic.AddUint64(&allowedCount, 1)
				}
			}
		}()
	}
	wg.Wait()

	if atomic.LoadUint64(&allowedCount) == 0 {
		t.Error("expected some tokens to be allowed")
	}
}

func BenchmarkTokenBucket_Allow(b *testing.B) {
	tb := NewTokenBucket(1000000, 1000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tb.Allow()
		}
	})
}
