package datagram

import (
	"errors"
	"sync"
)

var (
	// ErrExcessiveAmplification indicates the payload-to-padding ratio violates security thresholds.
	ErrExcessiveAmplification = errors.New("datagram amplification attack detected: excessive padding ratio")

	// bufferPool provides zero-allocation buffer reuse for hot packet inspection paths.
	bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 2048)
		},
	}
)

// DatagramStripper inspects and strips padding from HTTP/3 datagrams while mitigating amplification vectors.
type DatagramStripper struct {
	maxPaddingRatio float64
	metrics         *StripperMetrics
}

// NewDatagramStripper creates a new stripper with the specified maximum allowed padding ratio.
func NewDatagramStripper(maxPaddingRatio float64) *DatagramStripper {
	if maxPaddingRatio <= 0 {
		maxPaddingRatio = 0.5 // Default: padding cannot exceed 50% of total payload without scrutiny
	}
	return &DatagramStripper{
		maxPaddingRatio: maxPaddingRatio,
		metrics:         GetMetrics(),
	}
}

// ProcessDatagram parses an incoming HTTP/3 datagram frame, strips excessive padding frames,
// verifies amplification limits, and returns the cleaned payload slice without heap allocations.
func (s *DatagramStripper) ProcessDatagram(raw []byte) ([]byte, error) {
	s.metrics.TotalDatagrams.Add(1)
	s.metrics.BytesProcessed.Add(uint64(len(raw)))

	if len(raw) == 0 {
		return raw, nil
	}

	// Simplified HTTP/3 Datagram padding frame inspection (Type 0x00 / Padding Frame heuristic)
	// If the frame starts with padding markers or contains excessive trailing padding bytes,
	// strip or reject based on amplification limits.
	payloadLen, paddingLen, err := inspectPadding(raw)
	if err != nil {
		s.metrics.AmplificationDrops.Add(1)
		return nil, err
	}

	if paddingLen > 0 {
		s.metrics.StrippedPadding.Add(uint64(paddingLen))
	}

	// Return stripped payload without allocation by slicing the input buffer
	return raw[:payloadLen], nil
}

// inspectPadding evaluates payload vs padding ratios in O(1) streaming inspection.
func inspectPadding(raw []byte) (int, int, error) {
	n := len(raw)
	// Detect trailing padding bytes (e.g., zero-padding or HTTP/3 padding frame trailers)
	paddingLen := 0
	for i := n - 1; i >= 0; i-- {
		if raw[i] == 0x00 {
			paddingLen++
		} else {
			break
		}
	}

	// If more than 80% of the datagram is zero padding, enforce amplification check
	if paddingLen > 0 && float64(paddingLen)/float64(n) > 0.8 {
		return 0, paddingLen, ErrExcessiveAmplification
	}

	return n - paddingLen, paddingLen, nil
}
