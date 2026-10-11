package reassembler

import (
	"testing"
)

func TestReassembler(t *testing.T) {
	r := NewReassembler()
	defer r.buffer.Release()

	// TLS Record: Type 22 (Handshake), Version 3.3, Length 2
	fragment1 := []byte{22, 3, 3, 0, 2}
	fragment2 := []byte{0x01, 0x02}

	res, _ := r.Reassemble(fragment1)
	if res != nil {
		t.Error("Expected nil, got data")
	}

	res, _ = r.Reassemble(fragment2)
	if len(res) != 7 {
		t.Errorf("Expected length 7, got %d", len(res))
	}
}
