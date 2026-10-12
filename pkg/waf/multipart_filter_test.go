package waf

import (
	"sync"
	"testing"
)

func TestBoundaryFilter_InspectChunk(t *testing.T) {
	filter := NewBoundaryFilter()
	tests := []struct {
		name  string
		chunk []byte
		want  bool
	}{
		{"Normal Boundary", []byte("--boundary123"), true},
		{"Empty", []byte(""), true},
		{"Data", []byte("some content"), true},
		{"Smuggling NullByte", []byte("--boundary\x00smuggle"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := filter.InspectChunk(tt.chunk); got != tt.want {
				t.Errorf("InspectChunk() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBoundaryFilter_ConcurrencyAndPool(t *testing.T) {
	filter := NewBoundaryFilter()
	var wg sync.WaitGroup
	workers := 50

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			buf := filter.GetBuffer()
			defer filter.PutBuffer(buf)

			if id%2 == 0 {
				_ = filter.InspectChunk([]byte("--concurrent-boundary"))
			} else {
				_ = filter.InspectChunk([]byte("safe-data-chunk"))
			}
			filter.Reset()
		}(i)
	}
	wg.Wait()
}
