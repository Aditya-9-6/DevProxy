package waf

import (
	"sync"
	"testing"
)

func TestNormalizeQueryString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{{"b=2&a=1", "a=1&b=2"}, {"z=1&z=0", "z=0&z=1"}, {"", ""}, {"a=1", "a=1"}}
	for _, tt := range tests {
		got := NormalizeQueryString(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeQueryString(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestEntropyScannerConcurrent(t *testing.T) {
	s := &EntropyScanner{}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Add([]byte("testdata"))
			s.Calculate()
		}()
	}
	wg.Wait()
}

func BenchmarkNormalizeQueryString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NormalizeQueryString("c=3&b=2&a=1")
	}
}

func BenchmarkEntropyScanner(b *testing.B) {
	s := &EntropyScanner{}
	data := []byte("benchmark data")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Add(data)
		s.Calculate()
	}
}
