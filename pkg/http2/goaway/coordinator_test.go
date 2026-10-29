package goaway

import (
	"testing"
	"time"
)

func TestCoordinatorLifecycle(t *testing.T) {
	coord := NewCoordinator()
	if coord.State() != StateActive {
		tFatalf(t, "expected StateActive, got %v", coord.State())
	}

	f := &Frame{LastStreamID: 10, ErrorCode: 0}
	err := coord.HandleGoAway(f, time.Second)
	if err != nil {
		t.Fatalf("unexpected error handling goaway: %v", err)
	}

	if coord.State() != StateDraining {
		t.Fatalf("expected StateDraining, got %v", coord.State())
	}

	err = coord.CanCreateStream(5)
	if err != nil {
		t.Errorf("expected stream 5 to be allowed, got %v", err)
	}

	err = coord.CanCreateStream(12)
	if err == nil {
		t.Errorf("expected stream 12 to be rejected on draining connection")
	}

	// Test non-increasing last stream ID enforcement
	f2 := &Frame{LastStreamID: 15, ErrorCode: 0}
	err = coord.HandleGoAway(f2, time.Second)
	if err != ErrInvalidLastStreamID {
		t.Errorf("expected ErrInvalidLastStreamID, got %v", err)
	}
}

func tFatalf(t *testing.T, format string, args ...interface{}) {
	t.Fatalf(format, args...)
}
