package websocket

import (
	"testing"
)

func TestPool(t *testing.T) {
	t.Parallel()
	buf := GetBuffer()
	if len(buf) < 65536 {
		t.Errorf("Expected buffer length at least 65536, got %d", len(buf))
	}

	PutBuffer(buf)

	smallBuf := make([]byte, 100)
	PutBuffer(smallBuf) // Should be discarded safely due to cap check
}
