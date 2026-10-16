package trace

import (
	"bytes"
	"sync"
	"testing"
)

func TestGenerateTraceParent(t *testing.T) {
	tests := []struct {
		name    string
		traceID [16]byte
	}{
		{"standard", [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}},
		{"zeros", [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPropagator()
			buf := new(bytes.Buffer)
			err := p.WriteTraceParent(buf, tt.traceID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			val := buf.String()
			guidLen := len(val)
			if guidLen != 55 {
				t.Errorf("expected length 55, got %d", guidLen)
			}
		})
	}
}

func TestGenerateTraceParent_Concurrency(t *testing.T) {
	p := NewPropagator()
	var wg sync.WaitGroup
	iterations := 1000
	results := make(chan string, iterations)

	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			buf := new(bytes.Buffer)
			_ = p.WriteTraceParent(buf, [16]byte{})
			results <- buf.String()
		}()
	}
	wg.Wait()
	close(results)

	seen := make(map[string]bool)
	for res := range results {
		if seen[res] {
			t.Errorf("duplicate span ID detected: %s", res)
		}
		seen[res] = true
	}
}
