package waf

import (
	"sync"
	"testing"
	"time"
)

func TestTokenBucket_Allow(t *testing.T) {
	rate := uint64(100)
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

	time.Sleep(20 * time.Millisecond)
	if !tb.Allow() {
		t.Error("expected token after refill")
	}
}

func TestTokenBucket_Concurrent(t *testing.T) {
	rate := uint64(1000)
	burst := uint64(100)
	tb := NewTokenBucket(rate, burst)
	workers := 10
	iterations := 50
	var wg sync.WaitGroup

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				tb.Allow()
			}
		}()
	}
	wg.Wait()
}
