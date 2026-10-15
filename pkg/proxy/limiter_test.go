package proxy

import (
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	// Capacity 5, 10 tokens per second
	b := NewTokenBucket(5, 10)

	// Consume 5 tokens
	for i := 0; i < 5; i++ {
		if !b.Allow() {
			t.Errorf("expected token %d to be allowed", i)
		}
	}

	// Should be empty
	if b.Allow() {
		t.Error("expected bucket to be empty")
	}

	// Wait for refill
	time.Sleep(200 * time.Millisecond)

	// Should have refilled at least 1
	if !b.Allow() {
		t.Error("expected token after refill")
	}
}
