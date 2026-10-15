package security

import (
	"fmt"
	"sync"
	"testing"
)

func TestCookieFilter(t *testing.T) {
	tests := []struct {
		name      string
		sig       string
		shouldAdd bool
		expect    bool
	}{
		{"Basic insertion", "sig1", true, true},
		{"Unknown signature", "unknown", false, false},
		{"Empty string", "", true, true},
		{"Long signature", "very-long-signature-string-that-exceeds-standard-length", true, true},
	}

	filter := NewCookieFilter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldAdd {
				filter.Add(tt.sig)
			}
			if filter.Contains(tt.sig) != tt.expect {
				t.Errorf("Contains(%q) = %v, want %v", tt.sig, !tt.expect, tt.expect)
			}
		})
	}
}

func TestCookieFilterConcurrent(t *testing.T) {
	filter := NewCookieFilter()
	const workers = 10
	const opsPerWorker = 1000
	var wg sync.WaitGroup

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerWorker; j++ {
				sig := fmt.Sprintf("sig-%d-%d", id, j)
				filter.Add(sig)
				if !filter.Contains(sig) {
					t.Errorf("Concurrent failure: could not find %s", sig)
				}
			}
		}(i)
	}
	wg.Wait()
}
