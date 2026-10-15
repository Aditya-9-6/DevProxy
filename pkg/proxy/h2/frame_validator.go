package h2

import (
	"errors"
)

var ErrHeaderBlockTooLarge = errors.New("header block size exceeded")
var ErrTooManyContinuations = errors.New("too many continuation frames")

// ValidateFrame processes an HTTP/2 frame and enforces security invariants.
func ValidateFrame(tracker *StreamTracker, streamID uint32, frameType uint8, payloadLen uint32, isEndHeaders bool) error {
	state := tracker.Get(streamID)

	if frameType == 0x9 { // CONTINUATION
		state.ContinuationCount.Add(1)
		if state.ContinuationCount.Load() > 100 { // Limit to 100 continuation frames per block
			return ErrTooManyContinuations
		}
	}

	// Track total header block size
	if frameType == 0x1 || frameType == 0x9 { // HEADERS or CONTINUATION
		newSize := state.HeaderBlockSize.Add(uint64(payloadLen))
		if newSize > MaxHeaderBlockSize {
			return ErrHeaderBlockTooLarge
		}
	}

	if isEndHeaders {
		tracker.Reset(streamID)
	}

	return nil
}
