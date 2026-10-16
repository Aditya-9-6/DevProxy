package tls

import (
	"testing"
	"time"
)

func TestKeyRotator(t *testing.T) {
	rotator, err := NewKeyRotator(10 * time.Millisecond)
	if err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer rotator.Stop()

	first := rotator.GetActive()
	time.Sleep(25 * time.Millisecond)
	second := rotator.GetActive()

	if first == second {
		t.Error("key did not rotate")
	}
}
