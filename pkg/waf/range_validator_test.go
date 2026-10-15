package waf

import (
	"sync"
	"testing"
)

func TestRangeValidator(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   bool
	}{
		{"Valid", "bytes=0-100, 200-300", true},
		{"Overlap", "bytes=0-100, 50-150", false},
		{"TooMany", "bytes=0-1, 2-3, 4-5, 6-7, 8-9, 10-11, 12-13, 14-15, 16-17, 18-19, 20-21, 22-23, 24-25, 26-27, 28-29, 30-31, 32-33", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewRangeValidator()
			if got := v.ValidateRangeHeader(tt.header); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRangeValidator_Persistence(t *testing.T) {
	v := NewRangeValidator()
	if !v.ValidateRangeHeader("bytes=0-10") {
		t.Fatal("First request failed")
	}
	v.Reset()
	if !v.ValidateRangeHeader("bytes=0-10") {
		t.Fatal("Second request failed after reset")
	}
}

func TestRangeValidator_Concurrency(t *testing.T) {
	v := NewRangeValidator()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Concurrent validation of non-overlapping ranges
			_ = v.ValidateRangeHeader("bytes=0-10")
		}()
	}
	wg.Wait()
}
