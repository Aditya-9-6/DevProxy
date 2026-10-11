package ws

import (
	"bytes"
	"testing"
)

func TestMaskXOR(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		key   []byte
		want  []byte
	}{{"basic", []byte("hello"), []byte{1, 2, 3, 4}, []byte{'h' ^ 1, 'e' ^ 2, 'l' ^ 3, 'l' ^ 4, 'o' ^ 1}}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := append([]byte(nil), tt.input...)
			MaskXOR(data, tt.key)
			if !bytes.Equal(data, tt.want) {
				t.Errorf("MaskXOR() = %v, want %v", data, tt.want)
			}
			// Verify reversibility
			MaskXOR(data, tt.key)
			if !bytes.Equal(data, tt.input) {
				t.Errorf("Reversibility failed: got %v, want %v", data, tt.input)
			}
		})
	}
}
