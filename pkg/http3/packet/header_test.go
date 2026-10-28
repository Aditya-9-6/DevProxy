package packet

import (
	"bytes"
	"testing"
)

func TestParseVarInt(t *testing.T) {
	buf := []byte{0x65, 0x85}
	r := bytes.NewReader(buf)
	val, n, err := ParseVarInt(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 bytes read, got %d", n)
	}
	if val != 9605 {
		t.Errorf(
			"expected 9605, got %d", val,
		)
	}
}
