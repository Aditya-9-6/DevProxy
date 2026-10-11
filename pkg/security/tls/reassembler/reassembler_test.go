package reassembler

import (
	"testing"
)

func TestReassembler(t *testing.T) {
	tests := []struct {
		name     string
		chunks   [][]byte
		expected int
		err      bool
	}{
		{"HappyPath", [][]byte{{22, 3, 3, 0, 2}, {0x01, 0x02}}, 7, false},
		{"Fragmented", [][]byte{{22, 3, 3}, {0, 2, 0x01, 0x02}}, 7, false},
		{"Overflow", [][]byte{make([]byte, 70000)}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReassembler()
			defer r.buffer.Release()
			var res []byte
			var err error
			for _, chunk := range tt.chunks {
				res, err = r.Reassemble(chunk)
				if err != nil && !tt.err {
					t.Fatalf("unexpected error: %v", err)
				}
			}
			if !tt.err && len(res) != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, len(res))
			}
		})
	}
}
