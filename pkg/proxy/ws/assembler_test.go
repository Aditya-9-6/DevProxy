package ws

import (
	"bytes"
	"sync"
	"testing"
)

func TestAssembler(t *testing.T) {
	tests := []struct {
		name     string
		frames   []*Frame
		expected []byte
	}{
		{"simple", []*Frame{{Fin: false, Payload: []byte("hello ")}, {Fin: true, Payload: []byte("world")}}, []byte("hello world")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAssembler()
			var res []byte
			for _, f := range tt.frames {
				val, _ := a.Add(f)
				if val != nil {
					res = val
				}
			}
			if !bytes.Equal(res, tt.expected) {
				t.Errorf("expected %s, got %s", tt.expected, string(res))
			}
			if res != nil {
				PutBuffer(res)
			}
		})
	}
}

func TestAssemblerConcurrent(t *testing.T) {
	a := NewAssembler()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Simulate fragmented stream
			_, _ = a.Add(&Frame{Fin: false, Payload: GetBuffer()[:10]})
		}()
	}
	wg.Wait()
}
