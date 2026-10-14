package waf

import (
	"sync"
	"testing"
)

func TestWAFEngineAndMatcher(t *testing.T) {
	engine := NewEngine(4096)

	tests := []struct {
		name          string
		payload       string
		shouldInspect bool
	}{
		{
			name:          "Clean Payload",
			payload:       "Hello World, welcome to DevProxy",
			shouldInspect: false,
		},
		{
			name:          "SQL Injection Payload",
			payload:       "admin' OR 1=1 --",
			shouldInspect: true,
		},
		{
			name:          "XSS Injection Payload",
			payload:       "<script>alert(1)</script>",
			shouldInspect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched := engine.InspectPayload([]byte(tt.payload))
			if matched != tt.shouldInspect {
				t.Errorf("InspectPayload() = %v, want %v", matched, tt.shouldInspect)
			}
		})
	}

	scans, matches := engine.Stats()
	if scans != 3 {
		t.Errorf("Expected 3 scans, got %d", scans)
	}
	if matches != 2 {
		t.Errorf("Expected 2 matches, got %d", matches)
	}
}

func TestMatcherConcurrency(t *testing.T) {
	matcher := NewMatcher([]string{"test"})
	var wg sync.WaitGroup
	const goroutines = 50
	const iterations = 100

	// Concurrent reads
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				matcher.Scan([]byte("test"))
			}
		}()
	}

	// Concurrent writes
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				matcher.AddPattern("pattern-" + string(rune(val)))
			}
		}(i)
	}

	wg.Wait()
}
