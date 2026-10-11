package proxy

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)

	if !rl.Allow("127.0.0.1") {
		t.Error("Expected first request to be allowed")
	}
	if !rl.Allow("127.0.0.1") {
		t.Error("Expected second request to be allowed")
	}
	if rl.Allow("127.0.0.1") {
		t.Error("Expected third request to be blocked")
	}

	time.Sleep(150 * time.Millisecond)

	if !rl.Allow("127.0.0.1") {
		t.Error("Expected request after window to be allowed")
	}
}
