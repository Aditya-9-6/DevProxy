package h2

import (
	"testing"
)

func TestValidateFrame(t *testing.T) {
	tracker := NewStreamTracker()
	streamID := uint32(1)

	// Test normal header flow
	err := ValidateFrame(tracker, streamID, 0x1, 100, false)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Test continuation flood
	for i := 0; i < 105; i++ {
		err = ValidateFrame(tracker, streamID, 0x9, 10, false)
	}
	if err != ErrTooManyContinuations {
		t.Errorf("expected ErrTooManyContinuations, got %v", err)
	}

	// Test size limit
	tracker.Reset(streamID)
	err = ValidateFrame(tracker, streamID, 0x1, MaxHeaderBlockSize+1, true)
	if err != ErrHeaderBlockTooLarge {
		t.Errorf("expected ErrHeaderBlockTooLarge, got %v", err)
	}
}
