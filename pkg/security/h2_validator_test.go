package security

import (
	"sync"
	"testing"
	"time"
)

func TestHTTP2StreamValidator_ValidSequence(t *testing.T) {
	t.Parallel()
	val := NewHTTP2StreamValidator()
	defer val.Close()

	// Valid start: HEADERS on stream 1
	anomaly, err := val.ValidateFrame(1, FrameHeaders, 0x4) // END_HEADERS
	if err != nil || anomaly != nil {
		t.Fatalf("unexpected anomaly on valid stream open: %v", anomaly)
	}

	// Data frame on open stream 1
	anomaly, err = val.ValidateFrame(1, FrameData, 0x1) // END_STREAM
	if err != nil || anomaly != nil {
		t.Fatalf("unexpected anomaly on valid data frame: %v", anomaly)
	}
}

func TestHTTP2StreamValidator_InvalidSequence(t *testing.T) {
	t.Parallel()
	val := NewHTTP2StreamValidator()
	defer val.Close()

	// Invalid start: DATA frame on unopened stream 3
	anomaly, err := val.ValidateFrame(3, FrameData, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if anomaly == nil {
		t.Fatalf("expected protocol anomaly when opening stream with DATA frame")
	}
}

func TestHTTP2StreamValidator_ConcurrencyAndStress(t *testing.T) {
	t.Parallel()
	val := NewHTTP2StreamValidator()
	defer val.Close()

	var wg sync.WaitGroup
	workers := 50
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				streamID := uint32(workerID*iterations + j + 1)
				_, _ = val.ValidateFrame(streamID, FrameHeaders, 0x4)
				_, _ = val.ValidateFrame(streamID, FrameData, 0x1)
			}
		}(i)
	}
	wg.Wait()
}

func TestHTTP2StreamValidator_EvictionAndTTL(t *testing.T) {
	t.Parallel()
	val := NewHTTP2StreamValidatorWithTTL(50 * time.Millisecond)
	defer val.Close()

	streamID := uint32(99)
	_, _ = val.ValidateFrame(streamID, FrameHeaders, 0x1)

	// Wait for eviction TTL
	time.Sleep(100 * time.Millisecond)

	// Stream should be cleaned up and treated as idle (so sending Data will now error)
	anomaly, _ := val.ValidateFrame(streamID, FrameData, 0)
	if anomaly == nil {
		t.Fatalf("expected anomaly because expired stream should be evicted and treated as uninitialized")
	}
}
