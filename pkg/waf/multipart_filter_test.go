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
		{"CRLF Injection Anomalous", []byte("data\r\rdata"), false},
		{"Multiple Newlines", []byte("data\n\ndata"), false},
		{"Duplicate Headers TE.CL", []byte("--boundary\r\nContent-Length: 10\r\nTransfer-Encoding: chunked\r\n\r\n"), false},
	}

	for _, tt := range tests {
		structTest := tt
		t.Run(structTest.name, func(t *testing.T) {
			t.Parallel()
			if got := filter.InspectChunk(structTest.chunk); got != structTest.want {
				t.Errorf("InspectChunk() = %v, want %v", got, structTest.want)
			}
		})
	}
}

func TestBoundaryFilter_MultiChunkSplit(t *testing.T) {
	filter := NewBoundaryFilter()
	// Test boundary string split across two separate calls to InspectChunk
	chunk1 := []byte("some payload-")
	chunk2 := []byte("-boundary123")

	if !filter.InspectChunk(chunk1) {
		t.Errorf("InspectChunk chunk1 failed")
	}
	if !filter.InspectChunk(chunk2) {
		t.Errorf("InspectChunk chunk2 failed")
	}
}

func TestBoundaryFilter_ConcurrencyAndPool(t *testing.T) {
	var wg sync.WaitGroup
	workers := 50

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()
			// Each concurrent stream/goroutine uses its own request-scoped filter or safely uses thread-safe methods
			filter := NewBoundaryFilter()
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
