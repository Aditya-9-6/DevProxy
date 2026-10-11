package quic

import (
	"context"
	"testing"
)

func TestMultiplexer_Dispatch(t *testing.T) {
	m := NewMultiplexer()
	ctx := context.Background()

	data := []byte("test-quic-datagram")
	err := m.Dispatch(ctx, "conn-1", data)
	if err != nil {
		t.Errorf("Dispatch failed: %v", err)
	}

	if m.active.Load() != 0 {
		t.Error("Expected active count to return to 0")
	}
}
