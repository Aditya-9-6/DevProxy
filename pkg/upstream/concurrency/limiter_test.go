package concurrency

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	l := NewLimiter(1, 10)
	if !l.Acquire() {
		t.Fatal("Should acquire")
	}
	if l.Acquire() {
		t.Fatal("Should not acquire")
	}
	l.Release(10 * time.Millisecond)
	if !l.Acquire() {
		t.Fatal("Should acquire after release")
	}
}
