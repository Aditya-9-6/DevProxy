package ws

import (
	"sync"
	"testing"
)

func TestAssembler(t *testing.T) {
	tests := []struct {
		name     string
		frames   []*Frame
		expected string
	}{
		{"simple", []*Frame{{Fin: false, Payload: []byte("hello ")}, {Fin: true, Payload: []byte("world")}}, "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAssembler()
			var res []byte
			for _, f := range tt.frames {
				res = a.Add(f)
			}
			if string(res) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(res))
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
			_ = a.Add(&Frame{Fin: false, Payload: []byte("a")})
		}()
	}
	wg.Wait()
}
