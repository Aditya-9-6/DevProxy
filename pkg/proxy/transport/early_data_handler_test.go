package transport

import (
	"testing"
)

func TestEarlyDataHandler_Validate(t *testing.T) {
	handler := NewEarlyDataHandler()

	// Initial digest should be accepted (not seen before)
	digest := uint64(12345)
	if !handler.Validate(digest) {
		t.Errorf("Expected first digest %d to be accepted", digest)
	}

	// Same digest should be rejected as replay
	if handler.Validate(digest) {
		t.Errorf("Expected repeated digest %d to be rejected as replay", digest)
	}

	// Different digest should be accepted
	newDigest := uint64(67890)
	if !handler.Validate(newDigest) {
		t.Errorf("Expected new digest %d to be accepted", newDigest)
	}
}
