package proxy

import (
	"sync"
	"testing"
	"time"
)

func TestTokenBucket_Basic(t *testing.T) {
	t.Parallel()
	b := NewTokenBucket(10, 1000)
	for i := 0; i < 10; i++ {
		if !b.Allow() {
			t.Errorf("expected allow at index %d", i)
		}
	}
	if b.Allow() {
		t.Error("expected deny after capacity exhausted")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	t.Parallel()
	b := NewTokenBucket(5, 10)
	for i := 0; i < 5; i++ {
		if !b.Allow() {
			t.Errorf("expected token %d to be allowed", i)
		}
	}
	if b.Allow() {
		t.Error("expected bucket to be empty")
	}
	time.Sleep(250 * time.Millisecond)
	if !b.Allow() {
		t.Error("expected token after refill")
	}
}

func TestTokenBucketConcurrent(t *testing.T) {
	t.Parallel()
	b := NewTokenBucket(10000, 50000)
	var wg sync.WaitGroup
	workers := 20
	iterations := 200
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = b.Allow()
			}
		}()
	}
	wg.Wait()
}
