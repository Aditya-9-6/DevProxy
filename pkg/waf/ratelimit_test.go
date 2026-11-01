package waf

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestTokenBucket_BasicOperation(t *testing.T) {
	b := NewTokenBucket(10, 10)
	defer b.Release()

	ctx := context.Background()

	allowed, err := b.Allow(ctx, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Errorf("expected request for 5 tokens to be allowed")
	}

	allowed, err = b.Allow(ctx, 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Errorf("expected request for 6 tokens to be denied (only 5 left)")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	// Capacity 5, Refill 100 tokens per second (1 token every 10ms)
	b := NewTokenBucket(5, 100)
	defer b.Release()

	ctx := context.Background()

	// Consume all tokens
	allowed, _ := b.Allow(ctx, 5)
	if !allowed {
		t.Fatalf("initial consumption failed")
	}

	// Immediate follow-up should fail
	allowed, _ = b.Allow(ctx, 1)
	if allowed {
		t.Errorf("expected denial before refill")
	}

	// Wait for replenishment
	time.Sleep(30 * time.Millisecond)

	allowed, _ = b.Allow(ctx, 2)
	if !allowed {
		t.Errorf("expected allowance after token refill, got denied")
	}
}

func TestTokenBucket_ContextCancellation(t *testing.T) {
	b := NewTokenBucket(10, 1)
	defer b.Release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := b.Allow(ctx, 1)
	if err == nil {
		t.Errorf("expected error due to cancelled context, got nil")
	}
}

func TestTokenBucket_ConcurrentSafety(t *testing.T) {
	b := NewTokenBucket(1000, 10000)
	defer b.Release()

	var wg sync.WaitGroup
	workers := 50
	requestsPerWorker := 20

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < requestsPerWorker; j++ {
				_, _ = b.Allow(ctx, 1)
			}
		}()
	}

	wg.Wait()
}
