package security

import (
	"sync"
	"testing"
)

func TestStreamReassembler(t *testing.T) {
	tests := []struct {
		name     string
		segments []struct {
			seq  uint64
			data string
		}
		expected string
	}{
		{"HappyPath", []struct {
			seq  uint64
			data string
		}{{0, "hello"}, {5, "world"}}, "helloworld"},
		{"Overlap", []struct {
			seq  uint64
			data string
		}{{0, "hello"}, {3, "lo world"}}, "hello world"},
		{"Duplicate", []struct {
			seq  uint64
			data string
		}{{0, "hello"}, {0, "hello"}}, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewStreamReassembler(0)
			var result []byte
			for _, s := range tt.segments {
				ready := r.Push(s.seq, []byte(s.data))
				for _, b := range ready {
					result = append(result, b...)
					r.Release(b)
				}
			}
			if string(result) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(result))
			}
		})
	}
}

func TestStreamReassemblerConcurrent(t *testing.T) {
	r := NewStreamReassembler(0)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i uint64) {
			defer wg.Done()
			r.Push(i*5, []byte("data"))
		}(uint64(i))
	}
	wg.Wait()
}
