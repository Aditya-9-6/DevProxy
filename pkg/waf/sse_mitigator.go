package waf

import (
	"sync/atomic"
)

// SSEMitigator tracks per-client buffer saturation using lock-free atomic ring structures.
type SSEMitigator struct {
	// ring stores saturation levels, indexed by client ID hash.
	ring [1024]atomic.Uint64
	// threshold defines the saturation level at which to trigger backpressure.
	threshold uint64
}

// NewSSEMitigator creates a new mitigator instance.
func NewSSEMitigator(threshold uint64) *SSEMitigator {
	return &SSEMitigator{threshold: threshold}
}

// RecordSaturation increments the saturation counter for a given client hash.
func (m *SSEMitigator) RecordSaturation(clientIDHash uint32) {
	idx := clientIDHash % 1024
	m.ring[idx].Add(1)
}

// ShouldDrop evaluates if a client should be dropped based on current saturation.
func (m *SSEMitigator) ShouldDrop(clientIDHash uint32) bool {
	idx := clientIDHash % 1024
	return m.ring[idx].Load() > m.threshold
}

// ResetSaturation clears the saturation counter for a client.
func (m *SSEMitigator) ResetSaturation(clientIDHash uint32) {
	idx := clientIDHash % 1024
	m.ring[idx].Store(0)
}
