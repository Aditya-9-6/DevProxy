package waf

import (
	"testing"
	"time"
)

func TestTokenBucket_Allow(t *testing.T) {
	// 100 tokens per second, burst 5
	rate := uint64(100)
	burst := uint64(5)
	tb := NewTokenBucket(rate, burst)

	// Consume burst
	for i := uint64(0); i < burst; i++ {
		if !tb.Allow() {
			t.Errorf("expected token at index %d", i)
		}
	}

	// Should be empty
	if tb.Allow() {
		t.Error("expected bucket to be empty")
	}

	// Wait for refill
	time.Sleep(20 * time.Millisecond)
	if !tb.Allow() {
		t.Error("expected token after refill")
	}
}
