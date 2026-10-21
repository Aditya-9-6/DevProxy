package waf

import (
	"sync"
	"sync/atomic"
	"testing"
)

type rateLimiterTest struct {
	name                 string
	capacity             uint32
	refillRate           uint32
	refillNs             int64
	goroutineIDs         []int
	expected             []bool
	concurrentGoroutines int
	concurrentIterations int
}

func TestRateLimiter(t *testing.T) {
	tests := []rateLimiterTest{
		{
			name:         "normal operation",
			capacity:     3,
			refillRate:   0,
			refillNs:     1_000_000_000,
			goroutineIDs: []int{1, 1, 1},
			expected:     []bool{true, true, true},
		},
		{
			name:         "rate limiting",
			capacity:     1,
			refillRate:   0,
			refillNs:     1_000_000_000,
			goroutineIDs: []int{1, 1},
			expected:     []bool{true, false},
		},
		{
			name:         "capacity exceeded",
			capacity:     2,
			refillRate:   0,
			refillNs:     1_000_000_000,
			goroutineIDs: []int{1, 1, 1},
			expected:     []bool{true, true, false},
		},
		{
			name:                 "concurrent access",
			capacity:             50,
			refillRate:           10,
			refillNs:             1_000_000_000,
			concurrentGoroutines: 10,
			concurrentIterations: 3,
		},
		{
			name:       "sliding rate scale",
			capacity:   50,
			refillRate: 1,
			refillNs:   1_000_000_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRateLimiter(tt.capacity, tt.refillRate, tt.refillNs)

			// Run sequential Allow calls.
			for i, id := range tt.goroutineIDs {
				result := r.Allow(id)
				if result != tt.expected[i] {
					t.Errorf("Allow call %d: got %v, want %v", i, result, tt.expected[i])
				}
			}

			// If concurrent test, run goroutines.
			if tt.concurrentGoroutines > 0 {
				var wg sync.WaitGroup
				successCount := atomic.Uint64{}
				errorCount := atomic.Uint64{}

				iterations := tt.concurrentIterations
				if iterations == 0 {
					iterations = 1
				}

				for g := 0; g < tt.concurrentGoroutines; g++ {
					wg.Add(1)
					go func(workerID int) {
						defer wg.Done()
						for i := 0; i < iterations; i++ {
							id := workerID*1000 + i
							if r.Allow(id) {
								successCount.Add(1)
							} else {
								errorCount.Add(1)
							}
						}
					}(g)
				}

				wg.Wait()

				t.Logf("Concurrent test: %d successes, %d errors (rate limited)",
					successCount.Load(), errorCount.Load())

				if successCount.Load() == 0 {
					t.Error("All requests were rate limited - unexpected")
				}
			}
		})
	}
}
