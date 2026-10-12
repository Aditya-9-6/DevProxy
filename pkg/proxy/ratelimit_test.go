package proxy

import (
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	b := NewTokenBucket(10, 100)
	for i := 0; i < 10; i++ {
		if !b.Allow() {
			t.Errorf("expected allow at index %d", i)
		}
	}
	if b.Allow() {
		t.Error("expected deny after capacity exhausted")
	}
	time.Sleep(10 * time.Millisecond)
	if !b.Allow() {
		t.Error("expected allow after refill")
	}
}
