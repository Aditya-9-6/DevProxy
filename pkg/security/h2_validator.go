package security

import (
	"fmt"
	"sync"
	"time"
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

// streamRecord stores stream state and last updated timestamp for eviction.
type streamRecord struct {
	state     StreamState
	updatedAt time.Time
}

const shardCount = 16

// streamShard represents a sharded lock container to reduce global lock contention.
type streamShard struct {
	sync.RWMutex
	states map[uint32]streamRecord
}

// HTTP2StreamValidator validates HTTP/2 frame sequencing using sharded locks and lock-free bit vectors.
type HTTP2StreamValidator struct {
	shards    [shardCount]streamShard
	bitVec    *CacheAlignedBitVector
	ttl       time.Duration
	closeChan chan struct{}
	wg        sync.WaitGroup
}

// NewHTTP2StreamValidator initializes an HTTP2StreamValidator with sliding window cleanup.
func NewHTTP2StreamValidator() *HTTP2StreamValidator {
	return NewHTTP2StreamValidatorWithTTL(time.Minute * 10)
}

// NewHTTP2StreamValidatorWithTTL initializes an HTTP2StreamValidator with a custom TTL.
func NewHTTP2StreamValidatorWithTTL(ttl time.Duration) *HTTP2StreamValidator {
	v := &HTTP2StreamValidator{
		bitVec:    NewCacheAlignedBitVector(1 << 31), // Full HTTP/2 31-bit stream ID space support
		ttl:       ttl,
		closeChan: make(chan struct{}),
	}

	for i := 0; i < shardCount; i++ {
		v.shards[i].states = make(map[uint32]streamRecord)
	}

	v.wg.Add(1)
	go v.cleanupLoop()

	return v
}

// getShard selects the appropriate shard for a given stream ID.
func (v *HTTP2StreamValidator) getShard(streamID uint32) *streamShard {
	return &v.shards[streamID%shardCount]
}

// cleanupLoop periodically evicts closed or stale streams to prevent memory leaks.
func (v *HTTP2StreamValidator) cleanupLoop() {
	defer v.wg.Done()
	ticker := time.NewTicker(v.ttl / 2)
	defer ticker.Stop()

	for {
		select {
		case <-v.closeChan:
			return
		case <-ticker.C:
			now := time.Now()
			for i := 0; i < shardCount; i++ {
				shard := &v.shards[i]
				shard.Lock()
				for id, rec := range shard.states {
					if rec.state == StateClosed || now.Sub(rec.updatedAt) > v.ttl {
						delete(shard.states, id)
						v.bitVec.Clear(int(id))
					}
				}
				shard.Unlock()
			}
		}
	}
}

// Close terminates the validator background cleanup goroutine.
func (v *HTTP2StreamValidator) Close() {
	close(v.closeChan)
	v.wg.Wait()
}

// ValidateFrame inspects an incoming HTTP/2 frame against RFC 7540 sequencing rules.
func (v *HTTP2StreamValidator) ValidateFrame(streamID uint32, frameType HTTP2FrameType, flags uint8) (*H2StreamAnomaly, error) {
	if streamID == 0 {
		// Connection-level frames are handled separately
		return nil, nil
	}

	shard := v.getShard(streamID)
	shard.Lock()
	defer shard.Unlock()

	rec, exists := shard.states[streamID]
	state := StateIdle
	if exists {
		state = rec.state
	}

	now := time.Now()

	// Sequence rule: HEADERS or PUSH_PROMISE must establish stream if ID != 0
	if state == StateIdle {
		if frameType != FrameHeaders && frameType != FramePriority && frameType != FramePushPromise {
			return &H2StreamAnomaly{
				StreamID:  streamID,
				FrameType: frameType,
				Message:   fmt.Sprintf("protocol anomaly: stream %d opened with invalid frame type %d (expected HEADERS)", streamID, frameType),
			}, nil
		}
		state = StateOpen
		shard.states[streamID] = streamRecord{state: state, updatedAt: now}
		v.bitVec.Set(int(streamID))
	}

	// Check half-closed or closed state violations
	if state == StateClosed {
		return &H2StreamAnomaly{
			StreamID:  streamID,
			FrameType: frameType,
			Message:   fmt.Sprintf("protocol anomaly: frame received on closed stream %d", streamID),
		}, nil
	}

	// Update state based on frame flags (END_STREAM = 0x1)
	if (flags & 0x1) != 0 {
		if state == StateOpen {
			state = StateHalfClosedRemote
		} else if state == StateHalfClosedLocal {
			state = StateClosed
		}
	}

	if frameType == FrameRSTStream {
		state = StateClosed
	}

	shard.states[streamID] = streamRecord{state: state, updatedAt: now}
	if state == StateClosed {
		v.bitVec.Clear(int(streamID))
	}

	return nil, nil
}
