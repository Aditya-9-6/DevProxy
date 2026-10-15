package waf

import (
	"sync/atomic"
)

// WebRTCInspector provides lock-free analysis of WebRTC traffic.
// Uses [1000]atomic.Uint64 buckets for frequency tracking.
type WebRTCInspector struct {
	// MessageTypeCounts tracks frequency of STUN/TURN/DataChannel types.
	MessageTypeCounts [1000]atomic.Uint64
	// ByteLengthBuckets tracks distribution of packet sizes.
	ByteLengthBuckets [1000]atomic.Uint64
}

func NewWebRTCInspector() *WebRTCInspector {
	return &WebRTCInspector{}
}

// Inspect analyzes a packet descriptor. O(1) complexity.
func (w *WebRTCInspector) Inspect(msgType uint16, length uint64) {
	if msgType < 1000 {
		w.MessageTypeCounts[msgType].Add(1)
	}
	bucket := length / 128
	if bucket < 1000 {
		w.ByteLengthBuckets[bucket].Add(1)
	}
}

// GetStats returns current metrics without heap allocation.
func (w *WebRTCInspector) GetStats(msgType uint16) uint64 {
	if msgType >= 1000 {
		return 0
	}
	return w.MessageTypeCounts[msgType].Load()
}
