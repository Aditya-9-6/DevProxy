package waf

import (
	"sync/atomic"
)

// ViolationVector tracks header anomalies using cache-line padded atomic bitfields.
// Each bit represents a specific violation type to ensure lock-free recording.
type ViolationVector struct {
	_    [56]byte // Cache line padding
	bits atomic.Uint64
	_    [56]byte
}

const (
	ViolationDuplicatePseudoHeader = 1 << iota
	ViolationForbiddenChar
	ViolationLengthOverflow
)

// Record marks a violation in the bitfield.
func (v *ViolationVector) Record(violation uint64) {
	for {
		old := v.bits.Load()
		if v.bits.CompareAndSwap(old, old|violation) {
			break
		}
	}
}

// Has checks if a specific violation occurred.
func (v *ViolationVector) Has(violation uint64) bool {
	return (v.bits.Load() & violation) != 0
}

// HeaderDetector inspects HTTP headers for anomalies.
type HeaderDetector struct {
	MaxHeaderLength int
}

func NewHeaderDetector(maxLen int) *HeaderDetector {
	return &HeaderDetector{MaxHeaderLength: maxLen}
}

// Detect inspects a header key-value pair and records violations into the provided vector.
func (d *HeaderDetector) Detect(key, value string, vector *ViolationVector) {
	if len(key) > d.MaxHeaderLength || len(value) > d.MaxHeaderLength {
		vector.Record(ViolationLengthOverflow)
	}

	for i := 0; i < len(key); i++ {
		if key[i] < 32 || key[i] == 127 || key[i] == ':' {
			vector.Record(ViolationForbiddenChar)
			break
		}
	}
}
