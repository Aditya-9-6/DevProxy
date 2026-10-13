package limiter

import (
	"sync"
	"testing"
	"time"
)

func TestGCRALimiter_Basic(t *testing.T) {
	// 10 requests per second, burst of 2
	// Emission interval is 100ms per request
	limiter := NewGCRALimiter(10.0, 2)

	key := "test-client"

	// First 2 requests should be allowed (burst capacity)
	for i := 0; i < 2; i++ {
		if !limiter.Allow(key) {
			t.Errorf("expected request %d to be allowed", i)
		}
	}

	// 3rd request should be blocked immediately
	if limiter.Allow(key) {
		t.Error("expected request to be rate limited due to burst exhaustion")
	}

	// Wait for emission interval to pass (100ms + buffer)
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if !limiter.Allow(key) {
		t.Error(
			"expected request to be allowed after waiting emission interval",
		)
	}
}

func TestGCRALimiter_Concurrent(t *testing.T) {
	limiter := NewGCRALimiter(100.0, 10)
	key := "concurrent-client"

	var wg sync.WaitGroup
	workers := 20
	requestsPerWorker := 10

	allowedCount := 0
	var mu sync.Mutex

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localAllowed := 0
			for j := 0; j < requestsPerWorker; j++ {
				if limiter.Allow(key) {
					localAllowed++
				}
			}
			mu.Lock()
			allowedCount += localAllowed
			mu.Unlock()
		}()
	}

	wg.Wait()

	if allowedCount == 0 {
		t.Error("expected some requests to be allowed")
	}
}
