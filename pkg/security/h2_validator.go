package security

import (
	"fmt"
	"sync"
)

// HTTP2FrameType represents standard HTTP/2 frame types.
type HTTP2FrameType uint8

const (
	FrameData         HTTP2FrameType = 0x0
	FrameHeaders      HTTP2FrameType = 0x1
	FramePriority     HTTP2FrameType = 0x2
	FrameRSTStream    HTTP2FrameType = 0x3
	FrameSettings     HTTP2FrameType = 0x4
	FramePushPromise  HTTP2FrameType = 0x5
	FramePing         HTTP2FrameType = 0x6
	FrameGoAway       HTTP2FrameType = 0x7
	FrameWindowUpdate HTTP2FrameType = 0x8
	FrameContinuation HTTP2FrameType = 0x9
)

// StreamState tracks HTTP/2 stream lifecycle states.
type StreamState uint8

const (
	StateIdle StreamState = iota
	StateReservedLocal
	StateReservedRemote
	StateOpen
	StateHalfClosedLocal
	StateHalfClosedRemote
	StateClosed
)

// H2StreamAnomaly represents a protocol violation detected in HTTP/2 frame sequencing.
type H2StreamAnomaly struct {
	StreamID  uint32
	FrameType HTTP2FrameType
	Message   string
}

// HTTP2StreamValidator validates HTTP/2 frame sequencing using lock-free bit vectors per stream.
type HTTP2StreamValidator struct {
	mu     sync.Mutex
	states map[uint32]StreamState
	bitVec *CacheAlignedBitVector
}

// NewHTTP2StreamValidator initializes an HTTP2StreamValidator.
func NewHTTP2StreamValidator() *HTTP2StreamValidator {
	return &HTTP2StreamValidator{
		states: make(map[uint32]StreamState),
		bitVec: NewCacheAlignedBitVector(65536),
	}
}

// ValidateFrame inspects an incoming HTTP/2 frame against RFC 7540 sequencing rules.
func (v *HTTP2StreamValidator) ValidateFrame(streamID uint32, frameType HTTP2FrameType, flags uint8) (*H2StreamAnomaly, error) {
	v.mu.Lock()
	state, exists := v.states[streamID]
	if !exists {
		state = StateIdle
	}

	// Sequence rule: HEADERS or PUSH_PROMISE must establish stream if ID != 0
	if streamID != 0 && state == StateIdle {
		if frameType != FrameHeaders && frameType != FramePriority {
			v.mu.Unlock()
			return &H2StreamAnomaly{
				StreamID:  streamID,
				FrameType: frameType,
				Message:   fmt.Sprintf("protocol anomaly: stream %d opened with invalid frame type %d (expected HEADERS)", streamID, frameType),
			}, nil
		}
		state = StateOpen
		v.states[streamID] = state
		v.bitVec.Set(int(streamID % 65536))
	}

	// Check half-closed or closed state violations
	if state == StateClosed {
		v.mu.Unlock()
		return &H2StreamAnomaly{
			StreamID:  streamID,
			FrameType: frameType,
			Message:   fmt.Sprintf("protocol anomaly: frame received on closed stream %d", streamID),
		}, nil
	}

	// Update state based on frame flags (END_STREAM = 0x1)
	if (flags & 0x1) != 0 {
		if state == StateOpen {
			v.states[streamID] = StateHalfClosedRemote
		} else if state == StateHalfClosedLocal {
			v.states[streamID] = StateClosed
		}
	}

	v.mu.Unlock()
	return nil, nil
}
