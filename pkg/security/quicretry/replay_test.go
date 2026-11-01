package quicretry

import (
	"testing"
	"time"
)

func TestReplayCache(t *testing.T) {
	rc := NewReplayCache(100 * time.Millisecond)

	id := "unique-token-1"
	if !rc.CheckAndAdd(id) {
		t.Errorf("expected first check to succeed")
	}

	if rc.CheckAndAdd(id) {
		t.Errorf("expected second check to fail (replay)")
	}

	time.Sleep(120 * time.Millisecond)

	if !rc.CheckAndAdd(id) {
		t.Errorf("expected check to succeed after TTL expiration")
	}
}
