package datagram

import (
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding cache lines.
type CacheLinePad [8]uint64

// StripperMetrics tracks high-performance HTTP/3 datagram padding stats using lock-free atomic counters.
type StripperMetrics struct {
	_                  [56]byte
	TotalDatagrams     atomic.Uint64
	StrippedPadding    atomic.Uint64
	AmplificationDrops atomic.Uint64
	BytesProcessed     atomic.Uint64
	_                  [56]byte
}

var globalMetrics = &StripperMetrics{}

// GetMetrics returns the shared global instance of StripperMetrics.
func GetMetrics() *StripperMetrics {
	return globalMetrics
}
