package http3

import (
	"testing"
)

func TestQPACKDecoder_BlockedCount(t *testing.T) {
	decoder := NewQPACKDecoder()
	decoder.blockedCount.Add(1)
	if decoder.GetBlockedCount() != 1 {
		t.Errorf("expected blocked count 1, got %d", decoder.GetBlockedCount())
	}
}

func TestStreamBufferPool(t *testing.T) {
	buf := GetBuffer()
	if len(buf) != 16384 {
		t.Errorf("expected buffer size 16384, got %d", len(buf))
	}
	PutBuffer(buf)
}
