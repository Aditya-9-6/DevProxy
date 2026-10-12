package proxy

import (
	"sync"
	"testing"
)

func TestTokenBucket(t *testing.T) {
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

func TestTokenBucketConcurrent(t *testing.T) {
	b := NewTokenBucket(1000, 10000)
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				b.Allow()
			}
		}()
	}
	wg.Wait()
}
