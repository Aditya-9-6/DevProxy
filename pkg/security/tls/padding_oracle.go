package tls

import (
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding atomic counters to 64 bytes.
type CacheLinePad [8]uint64

// PaddingOracleDetector monitors TLS record length distributions for padding oracle patterns.
type PaddingOracleDetector struct {
	// Histogram of record lengths (up to 16KB TLS record limit)
	RecordLengthHistogram [16384]atomic.Uint64
	// InterArrivalVariance tracks timing jitter
	InterArrivalVariance CacheLinePad
}

func NewPaddingOracleDetector() *PaddingOracleDetector {
	return &PaddingOracleDetector{}
}

// ObserveRecord records the length of a TLS record in the histogram.
func (d *PaddingOracleDetector) ObserveRecord(length int) {
	if length >= 0 && length < 16384 {
		d.RecordLengthHistogram[length].Add(1)
	}
}

// GetRecordCount returns the count for a specific record length.
func (d *PaddingOracleDetector) GetRecordCount(length int) uint64 {
	if length >= 0 && length < 16384 {
		return d.RecordLengthHistogram[length].Load()
	}
	return 0
}
