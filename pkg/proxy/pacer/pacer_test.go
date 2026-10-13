package pacer

import (
	"testing"
	"time"
)

func TestTokenBucket_Consume(t *testing.T) {
	b := NewTokenBucket(1000, 100)

	if !b.Consume(50) {
		t.Error("expected to consume 50 tokens")
	}

	if b.Consume(100) {
		t.Error("expected to fail consuming 100 tokens")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	b := NewTokenBucket(1000, 100)
	_ = b.Consume(100)

	time.Sleep(10 * time.Millisecond)

	if !b.Consume(5) {
		t.Error("expected tokens to refill after sleep")
	}
}
